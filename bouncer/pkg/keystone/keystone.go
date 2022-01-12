package keystone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	bouncer "github.com/platform9/pf9-qbert/bouncer/pkg/api"
)

const (
	defaultDomainName = "default"
	tokensRelativeURL = "v3/auth/tokens?nocatalog"
)

type (
	keystone struct {
		baseURL    string
		httpClient http.Client
	}

	keystoneAuthRequest struct {
		Auth struct {
			Identity struct {
				Methods  []string        `json:"methods"`
				Password *PasswordMethod `json:"password,omitempty"`
				TokenID  *TokenIDMethod  `json:"token,omitempty"`
			} `json:"identity"`
			Scope struct {
				Project struct {
					ID     string         `json:"id,omitempty"`
					Name   string         `json:"name,omitempty"`
					Domain *ProjectDomain `json:"domain,omitempty"`
				} `json:"project"`
			} `json:"scope"`
		} `json:"auth"`
	}

	ProjectDomain struct {
		ID string `json:"id"`
	}

	PasswordMethod struct {
		User struct {
			Name   string `json:"name"`
			Domain struct {
				Name string `json:"name"`
			} `json:"domain"`
			Password string `json:"password"`
		} `json:"user"`
	}

	TokenIDMethod struct {
		ID string `json:"id"`
	}

	keystoneAuthResponse struct {
		Token *bouncer.KeystoneToken `json:"token"`
	}

	keystoneGroupResponse struct {
		Groups []bouncer.KeystoneGroup `json:"groups"`
	}
)

// New takes a keystone URL (e.g., "http://pf9.platform9.net/keystone/") and
// returns a Keystone client. Note that the client supports only HTTP.
func New(keystoneURL string, timeout time.Duration) (*keystone, error) {
	baseURL, err := url.Parse(keystoneURL)
	if err != nil {
		return nil, fmt.Errorf("parse keystone url: %s", err)
	}
	if !strings.HasSuffix(baseURL.Path, "/") {
		baseURL.Path = baseURL.Path + "/"
	}
	httpClient := http.Client{Timeout: timeout}
	return &keystone{baseURL.String(), httpClient}, nil
}

// ProjectTokenFromTokenID requests a project-scoped token as described in
// https://developer.openstack.org/api-ref/identity/v3/?expanded=#password-authentication-with-scoped-authorization
// If the tokenID is invalid or expired, or if the user has no roles in the project,
// Keystone does not return a project-scoped token, and the method returns an error.
func (k *keystone) ProjectTokenFromTokenID(tokenID, projectID string) (bouncer.KeystoneTokenWrapper, error) {
	req := keystoneAuthRequest{}
	req.Auth.Identity.Methods = []string{"token"}
	req.Auth.Identity.TokenID = &TokenIDMethod{tokenID}
	req.Auth.Scope.Project.ID = projectID

	token, err := k.Auth(&req)
	if err != nil {
		return bouncer.KeystoneTokenWrapper{}, fmt.Errorf("obtain project token from tokenID: %s", err)
	}
	return token, nil
}

// AuthWithTokenID requests a project-scoped token as described in
// https://developer.openstack.org/api-ref/identity/v3/?expanded=#token-authentication-with-scoped-authorization
// If the credentials are invalid, or if the user has no roles in the project,
// Keystone does not return a project-scoped token, and the method returns an error.
func (k *keystone) ProjectTokenFromCredentialsWithProjectId(
	username,
	password,
	projectID string,
) (bouncer.KeystoneTokenWrapper, error) {
	req := keystoneAuthRequest{}
	req.Auth.Identity.Methods = []string{"password"}
	req.Auth.Identity.Password = &PasswordMethod{}
	req.Auth.Identity.Password.User.Name = username
	req.Auth.Identity.Password.User.Password = password
	req.Auth.Identity.Password.User.Domain.Name = defaultDomainName
	req.Auth.Scope.Project.ID = projectID
	token, err := k.Auth(&req)
	if err != nil {
		return bouncer.KeystoneTokenWrapper{}, fmt.Errorf("obtain project token from credentials: %s", err)
	}
	return token, nil
}

// Similar to ProjectTokenFromCredentialsWithProjectId, except that the project
// is specified using a project name and domain id
func (k *keystone) ProjectTokenFromCredentialsWithProjectName(
	username,
	password,
	projectName string,
	domainId string,
) (bouncer.KeystoneTokenWrapper, error) {
	req := keystoneAuthRequest{}
	req.Auth.Identity.Methods = []string{"password"}
	req.Auth.Identity.Password = &PasswordMethod{}
	req.Auth.Identity.Password.User.Name = username
	req.Auth.Identity.Password.User.Password = password
	req.Auth.Identity.Password.User.Domain.Name = defaultDomainName
	req.Auth.Scope.Project.Name = projectName
	req.Auth.Scope.Project.Domain = &ProjectDomain{domainId}
	token, err := k.Auth(&req)
	if err != nil {
		return bouncer.KeystoneTokenWrapper{}, fmt.Errorf("obtain project token from credentials: %s", err)
	}
	return token, nil
}

func (k *keystone) Auth(req *keystoneAuthRequest) (bouncer.KeystoneTokenWrapper, error) {
	tokenWrapper := bouncer.KeystoneTokenWrapper{}
	token := bouncer.KeystoneToken{}

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return tokenWrapper, fmt.Errorf("encode keystone auth request: %s", err)
	}

	tokensURL := k.baseURL + tokensRelativeURL
	httpReq, err := http.NewRequest("POST", tokensURL, body)
	if err != nil {
		return tokenWrapper, fmt.Errorf("create keystone http request: %s", err)
	}
	httpReq.Header.Set("User-Agent", fmt.Sprintf("bouncer:%s keystone client", bouncer.Version))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := k.httpClient.Do(httpReq)
	// handle non-http status errors
	if err != nil {
		return tokenWrapper, fmt.Errorf("send keystone request: %s", err)
	}
	// use func to suppress IDE warning about unused return value
	defer func() { _ = resp.Body.Close() }()
	// BUG(daniel) Manual testing shows that the client follows a 302 redirect
	// and reports 200, while curl following the same redirect reports 405
	// handle http status errors
	if resp.StatusCode != http.StatusCreated {
		return tokenWrapper, bouncer.KeystoneResponseError{Message: resp.Status, StatusCode: resp.StatusCode}
	}

	// To unmarshal into KeystoneToken, we must wrap it in keystoneAuthResponse,
	// since the response body is `{ "token": <token object> }`, but we can
	// discard the wrapper struct
	authResp := keystoneAuthResponse{&token}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return tokenWrapper, fmt.Errorf("decode auth response: %s", err)
	}
	tokenWrapper.Token = token
	tokenWrapper.TokenID = resp.Header.Get("X-Subject-Token")
	return tokenWrapper, nil
}

func (k *keystone) groupsLocalUserBelongsTo(token *bouncer.KeystoneTokenWrapper) ([]bouncer.KeystoneGroup, error) {
	body := new(bytes.Buffer)
	groupsURL := k.baseURL + "/v3/users/" + token.Token.User.ID + "/groups"
	groupReq, err := http.NewRequest("GET", groupsURL, body)
	if err != nil {
		return nil, fmt.Errorf("create keystone http request: %v", err)
	}
	groupReq.Header.Set("X-Auth-Token", token.TokenID)
	groupResp, err := k.httpClient.Do(groupReq)
	if err != nil {
		return nil, fmt.Errorf("send keystone request: %v", err)
	}
	groupRespObj := keystoneGroupResponse{}
	if err := json.NewDecoder(groupResp.Body).Decode(&groupRespObj); err != nil {
		return nil, fmt.Errorf("decode keystone response: %v", err)
	}
	return groupRespObj.Groups, nil
}

func (k *keystone) listGroups(token *bouncer.KeystoneTokenWrapper) ([]bouncer.KeystoneGroup, error) {
	body := new(bytes.Buffer)
	groupListURL := k.baseURL + "/v3/groups"
	groupListReq, err := http.NewRequest("GET", groupListURL, body)
	if err != nil {
		return nil, fmt.Errorf("create keystone http request: %v", err)
	}
	groupListReq.Header.Set("X-Auth-Token", token.TokenID)
	groupsListResp, err := k.httpClient.Do(groupListReq)
	if err != nil {
		return nil, fmt.Errorf("send keystone request: %v", err)
	}
	groupRespObj := keystoneGroupResponse{}
	if err := json.NewDecoder(groupsListResp.Body).Decode(&groupRespObj); err != nil {
		return nil, fmt.Errorf("decode list groups response: %v", err)
	}
	return groupRespObj.Groups, nil
}

// GroupsFromProjectToken Gets Keystone groups that the user is a member of, considers both SSO/federated user and local user
func (k *keystone) GroupsFromProjectToken(token *bouncer.KeystoneTokenWrapper) ([]string, error) {

	// Get groups assuming SSO user
	groups := token.Token.User.OSFederation.Groups
	groupNameMap := make(map[string]string)
	// Add these group Ids to groupNameMap
	for _, group := range groups {
		groupNameMap[group.ID] = ""
	}

	// Get groups assuming local user
	keystoneGroupsForUser, err := k.groupsLocalUserBelongsTo(token)
	if err != nil {
		return nil, fmt.Errorf("error fetching groupsLocalUserBelongsTo %v", err)
	}
	// Add these group Ids to groupNameMap
	for _, group := range keystoneGroupsForUser {
		groupNameMap[group.ID] = ""
	}

	// Get list of groups from keystone, needed to convert groupId to groupName
	keystoneGroups, err := k.listGroups(token)
	if err != nil {
		return nil, fmt.Errorf("error fetching listGroups %v", err)
	}

	// map groupName to groupIds in groupNameMap
	for _, group := range keystoneGroups {
		_, ok := groupNameMap[group.ID]
		if ok {
			groupNameMap[group.ID] = group.Name
		}
	}

	// Build the groupNames array to return
	groupNames := []string{}
	for _, groupName := range groupNameMap {
		groupNames = append(groupNames, groupName)
	}

	return groupNames, nil
}
