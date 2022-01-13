package mock

import (
	"fmt"

	bouncer "github.com/platform9/pf9-qbert/bouncer/pkg/api"
)

type Keystone struct {
	Username        string
	Password        string
	ProjectID       string
	ProjectName     string
	ProjectDomainId string
	TokenWrapper    bouncer.KeystoneTokenWrapper
}

func (k Keystone) ProjectTokenFromTokenID(tokenID, projectID string) (bouncer.KeystoneTokenWrapper, error) {
	if tokenID == k.TokenWrapper.TokenID && projectID == k.ProjectID {
		return k.TokenWrapper, nil
	} else {
		return bouncer.KeystoneTokenWrapper{}, fmt.Errorf("expected tokenID %v and projectID %v, got %v and %v", k.TokenWrapper.TokenID, k.ProjectID, tokenID, projectID)
	}
}

func (k Keystone) ProjectTokenFromCredentialsWithProjectId(username, password, projectID string) (bouncer.KeystoneTokenWrapper, error) {
	if username == k.Username && password == k.Password && projectID == k.ProjectID {
		return k.TokenWrapper, nil
	} else {
		return bouncer.KeystoneTokenWrapper{}, fmt.Errorf("expected username %v, password %v, and projectID %v, got %v, %v, %v", k.Username, k.Password, k.ProjectID, username, password, projectID)
	}
}

func (k Keystone) ProjectTokenFromCredentialsWithProjectName(username, password, projectName string, domainId string) (bouncer.KeystoneTokenWrapper, error) {
	if username == k.Username && password == k.Password && projectName == k.ProjectName && domainId == k.ProjectDomainId {
		return k.TokenWrapper, nil
	} else {
		return bouncer.KeystoneTokenWrapper{}, fmt.Errorf("expected username %v, password %v, projectName %v and domainId %v, got %v, %v, %v", k.Username, k.Password, k.ProjectID, username, password, projectName, domainId)
	}
}

func (k Keystone) GroupsFromProjectToken(tokenWrapper *bouncer.KeystoneTokenWrapper) ([]string, error) {
	groups := []string{}
	return groups, nil
}
