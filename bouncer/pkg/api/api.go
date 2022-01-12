package api

import (
	"fmt"
	"time"
)

type (
	Keystone interface {
		ProjectTokenFromTokenID(tokenID, projectID string) (KeystoneTokenWrapper, error)
		ProjectTokenFromCredentialsWithProjectId(username, password, projectID string) (KeystoneTokenWrapper, error)
		ProjectTokenFromCredentialsWithProjectName(username, password, projectName string, domainId string) (KeystoneTokenWrapper, error)
		GroupsFromProjectToken(token *KeystoneTokenWrapper) ([]string, error)
	}

	KeystoneTokenWrapper struct {
		Token   KeystoneToken `json:"token"`
		TokenID string        `json:"string"`
	}

	// Project-scoped token
	KeystoneToken struct {
		AuditIds  []string        `json:"audit_ids"`
		ExpiresAt time.Time       `json:"expires_at"`
		IssuedAt  time.Time       `json:"issued_at"`
		Methods   []string        `json:"methods"`
		Project   KeystoneProject `json:"project"`
		User      KeystoneUser    `json:"user"`
		Roles     []KeystoneRole  `json:"roles"`
	}
	KeystoneProject struct {
		Domain struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"domain"`
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	KeystoneUser struct {
		Domain struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"domain"`
		ID           string `json:"id"`
		Name         string `json:"name"`
		OSFederation struct {
			Groups           []KeystoneGroup `json:"groups"`
			IdentityProvider struct {
				ID string `json:"id"`
			} `json:"identity_provider"`
			Protocol struct {
				ID string `json:"id"`
			} `json:"protocol"`
		} `json:"OS-FEDERATION"`
	}
	KeystoneRole struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	KeystoneGroup struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
)

// Error that includes a message and an HTTP status
type KeystoneResponseError struct {
	Message    string
	StatusCode int
}

// Returns a string representation of a KeystoneResponseError
func (e KeystoneResponseError) Error() string {
	return fmt.Sprintf("keystone response error: %d - %s", e.StatusCode, e.Message)
}
