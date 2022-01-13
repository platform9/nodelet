package keystone_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	bouncer "github.com/platform9/pf9-qbert/bouncer/pkg/api"
	"github.com/platform9/pf9-qbert/bouncer/pkg/keystone"
)

var timeout = time.Duration(5) * time.Second

func TestAuthWithTokenID(t *testing.T) {
	tokenID := "gAAAAABYt5PWnXGdQUWq6sXo8sj_n_1cOHbQ13F_a6sCMUkqjMjxPEgdtuFnTC4E8HXHwHTENpkn_NEnZKmsO7B8t4v1VL8PRaoILIiyNq-JrvRhDk911QVfb_SzQupPLbiieNGvddQCDve8mnbJkPj4bA_ikem7q-KFz2IhZj7nWFpWxgtVSrU"
	projectID := "dummy-project-id"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupResponse := `
		{
			"groups":
			[
				{
					"id": "dummy-group-id",
					"name": "dummy-group-name"
				}
			]
		}`
		authResponse := `
		{
			"token":
			{
				"audit_ids": [ "dummy-audit-id" ],
				"expires_at": "2017-03-09T03:14:40.000000Z",
				"issued_at": "2017-03-08T03:14:40.000000Z",
				"methods": [ "token" ],
				"project": {
					"domain": { "id": "default", "name": "Default" },
					"id": "dummy-project-id",
					"name": "dummy-project-name"
					},
				"roles": [
					{ "id": "dummy-role-id", "name": "dummy-role-name" }
				],
				"user": {
					"domain": { "id": "default", "name": "Default" },
					"id": "dummy-user-id",
					"name": "dummy-username"
				}
			}
		}`
		if strings.Contains(r.URL.Path, "token") {
			w.Header().Set("X-Subject-Token", tokenID)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(authResponse))
		}
		if strings.Contains(r.URL.Path, "groups") {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(groupResponse))
		}
	}))
	defer server.Close()

	k, err := keystone.New(server.URL, timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	token, err := k.ProjectTokenFromTokenID(tokenID, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bouncer.KeystoneToken{}
	expected.AuditIds = []string{"dummy-audit-id"}
	expected.ExpiresAt, _ = time.Parse(time.RFC3339Nano, "2017-03-09T03:14:40.000000Z")
	expected.IssuedAt, _ = time.Parse(time.RFC3339Nano, "2017-03-08T03:14:40.000000Z")
	expected.Methods = []string{"token"}
	expected.Project.ID = "dummy-project-id"
	expected.Project.Name = "dummy-project-name"
	expected.Project.Domain.ID = "default"
	expected.Project.Domain.Name = "Default"
	expected.Roles = []bouncer.KeystoneRole{{ID: "dummy-role-id", Name: "dummy-role-name"}}
	expected.User.Domain.ID = "default"
	expected.User.Domain.Name = "Default"
	expected.User.ID = "dummy-user-id"
	expected.User.Name = "dummy-username"

	expectedWrapper := bouncer.KeystoneTokenWrapper{}
	expectedWrapper.Token = expected
	expectedWrapper.TokenID = tokenID

	if !reflect.DeepEqual(expectedWrapper, token) {
		t.Fatalf("expected: %v\nactual%v", expected, token)
	}
}

func TestAuthWithCredentials(t *testing.T) {
	username := "dummy-username"
	password := "dummy-password"
	projectID := "dummy-project-id"
	projectName := "dummy-project-name"
	projectDomainId := "dummy-project-domain-id"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		groupResponse := `
		{
			"groups":
			[
				{
					"id": "dummy-group-id",
					"name": "dummy-group-name"
				}
			]
		}`
		authResponse := `
		{
			"token":
			{
				"audit_ids": [ "dummy-audit-id" ],
				"expires_at": "2017-03-09T03:14:40.000000Z",
				"issued_at": "2017-03-08T03:14:40.000000Z",
				"methods": [ "password" ],
				"project": {
					"domain": { "id": "dummy-project-domain-id", "name": "Default" },
					"id": "dummy-project-id",
					"name": "dummy-project-name"
					},
				"roles": [
					{ "id": "dummy-role-id", "name": "dummy-role-name" }
				],
				"user": {
					"domain": { "id": "default", "name": "Default" },
					"id": "dummy-user-id",
					"name": "dummy-username"
				}
			}
		}`
		if strings.Contains(r.URL.Path, "token") {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(authResponse))
		}
		if strings.Contains(r.URL.Path, "groups") {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(groupResponse))
		}
	}))
	defer server.Close()

	k, err := keystone.New(server.URL, timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	token, err := k.ProjectTokenFromCredentialsWithProjectId(username,
		password, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bouncer.KeystoneToken{}
	expected.AuditIds = []string{"dummy-audit-id"}
	expected.ExpiresAt, _ = time.Parse(time.RFC3339Nano, "2017-03-09T03:14:40.000000Z")
	expected.IssuedAt, _ = time.Parse(time.RFC3339Nano, "2017-03-08T03:14:40.000000Z")
	expected.Methods = []string{"password"}
	expected.Project.ID = "dummy-project-id"
	expected.Project.Name = projectName
	expected.Project.Domain.ID = projectDomainId
	expected.Project.Domain.Name = "Default"
	expected.Roles = []bouncer.KeystoneRole{{ID: "dummy-role-id", Name: "dummy-role-name"}}
	expected.User.Domain.ID = "default"
	expected.User.Domain.Name = "Default"
	expected.User.ID = "dummy-user-id"
	expected.User.Name = "dummy-username"

	expectedWrapper := bouncer.KeystoneTokenWrapper{}
	expectedWrapper.Token = expected

	if !reflect.DeepEqual(expectedWrapper, token) {
		t.Fatalf("expected: %v\nactual%v", expected, token)
	}

	token, err = k.ProjectTokenFromCredentialsWithProjectName(username,
		password, projectName, projectDomainId)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expectedWrapper, token) {
		t.Fatalf("expected: %v\nactual%v", expected, token)
	}
}

func TestSSOAuthWithTokenID(t *testing.T) {
	tokenID := "gAAAAABYt5PWnXGdQUWq6sXo8sj_n_1cOHbQ13F_a6sCMUkqjMjxPEgdtuFnTC4E8HXHwHTENpkn_NEnZKmsO7B8t4v1VL8PRaoILIiyNq-JrvRhDk911QVfb_SzQupPLbiieNGvddQCDve8mnbJkPj4bA_ikem7q-KFz2IhZj7nWFpWxgtVSrU"
	projectID := "dummy-project-id"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		listGroupsResponse := `
		{
			"groups":
			[
				{
					"id": "dummy-group-id-1",
					"name": "dummy-group-name-1"
				},
				{
					"id": "dummy-group-id-2",
					"name": "dummy-group-name-2"
				},
				{
					"id": "dummy-group-id-3",
					"name": "dummy-group-name-3"
				}
			]
		}`

		localUserGroupsResponse := `
		{
			"groups": []
		}`
		ssoAuthResponse := `
		{
			"token": {
				"methods": ["token", "saml2"],
				"user": {
					"domain": {
						"id": "Federated",
						"name": "Federated"
					},
					"id": "dummy-sso-user-id",
					"name": "dummy-sso-user-name",
					"OS-FEDERATION": {
						"groups": [{
							"id": "dummy-group-id-1"
						}, {
							"id": "dummy-group-id-3"
						}],
						"identity_provider": {
							"id": "IDP1"
						},
						"protocol": {
							"id": "saml2"
						}
					}
				},
				"audit_ids": ["dummy-audit-id"],
				"expires_at": "2021-09-21T20:10:01.000000Z",
				"issued_at": "2021-09-20T20:10:01.000000Z",
				"project": {
					"domain": {
						"id": "default",
						"name": "Default"
					},
					"id": "dummy-project-id",
					"name": "dummy-project-name"
				},
				"is_domain": false,
				"roles": [{
					"displayName": "Reader",
					"id": "reader-role-id",
					"name": "reader",
					"domain_id": null,
					"description": "A read-only role grants limited access to only view resources in the cloud.\n"
				}, {
					"displayName": "Self-Service User",
					"id": "ssu-role-id",
					"name": "_member_",
					"domain_id": null,
					"description": "A Self-service user has limited access to the cloud. He can create new instances using Images, Flavors and Networks he has access to. Instances are created on the infrastructure assigned to the Tenant that the Self-service user is a member of.\n"
				}]
			}
		}`

		if strings.Contains(r.URL.Path, "token") {
			w.Header().Set("X-Subject-Token", tokenID)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(ssoAuthResponse))
		}
		if strings.Contains(r.URL.Path, "v3/groups") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(listGroupsResponse))
		}
		if strings.Contains(r.URL.Path, "dummy-sso-user-id/groups") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(localUserGroupsResponse))
		}
	}))
	defer server.Close()

	k, err := keystone.New(server.URL, timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	token, err := k.ProjectTokenFromTokenID(tokenID, projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bouncer.KeystoneToken{}
	expected.AuditIds = []string{"dummy-audit-id"}
	expected.ExpiresAt, _ = time.Parse(time.RFC3339Nano, "2021-09-21T20:10:01.000000Z")
	expected.IssuedAt, _ = time.Parse(time.RFC3339Nano, "2021-09-20T20:10:01.000000Z")
	expected.Methods = []string{"token", "saml2"}
	expected.Project.ID = "dummy-project-id"
	expected.Project.Name = "dummy-project-name"
	expected.Project.Domain.ID = "default"
	expected.Project.Domain.Name = "Default"
	expected.Roles = []bouncer.KeystoneRole{{ID: "reader-role-id", Name: "reader"}, {ID: "ssu-role-id", Name: "_member_"}}
	expected.User.Domain.ID = "Federated"
	expected.User.Domain.Name = "Federated"
	expected.User.ID = "dummy-sso-user-id"
	expected.User.Name = "dummy-sso-user-name"
	expected.User.OSFederation.Groups = []bouncer.KeystoneGroup{{ID: "dummy-group-id-1"},{ID: "dummy-group-id-3"}}
	expected.User.OSFederation.Protocol.ID = "saml2"
	expected.User.OSFederation.IdentityProvider.ID = "IDP1"

	expectedWrapper := bouncer.KeystoneTokenWrapper{}
	expectedWrapper.Token = expected
	expectedWrapper.TokenID = tokenID

	if !reflect.DeepEqual(expectedWrapper, token) {
		t.Fatalf("expected: %v\nactual%v", expectedWrapper, token)
	}

	groups, err := k.GroupsFromProjectToken(&token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGroups := []string{"dummy-group-name-1", "dummy-group-name-3"}
	if !reflect.DeepEqual(expectedGroups, groups) {
		t.Fatalf("expected: %v\nactual%v", expectedGroups, groups)
	}
}

func TestFailedAuthWithTokenID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	k, err := keystone.New(server.URL, timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = k.ProjectTokenFromTokenID("", "")
	if err == nil {
		t.Errorf("expected error from AuthWithTokenID")
	}
}

func TestFailedAuthWithCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	k, err := keystone.New(server.URL, timeout)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = k.ProjectTokenFromCredentialsWithProjectId("", "", "")
	if err == nil {
		t.Errorf("expected error from AuthWithCredentials")
	}
}
