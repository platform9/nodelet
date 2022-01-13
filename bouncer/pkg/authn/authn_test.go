package authn_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	bouncer "github.com/platform9/pf9-qbert/bouncer/pkg/api"
	"github.com/platform9/pf9-qbert/bouncer/pkg/authn"
	"github.com/platform9/pf9-qbert/bouncer/pkg/mock"
	"github.com/platform9/pf9-qbert/bouncer/pkg/policy"
)

const (
	username   = "dummy-username"
	password   = "dummy-password"
	UID        = "dummy-user-id"
	projectID  = "dummy-project-id"
	tokenID    = "gAAAAABYt5PWnXGdQUWq6sXo8sj_n_1cOHbQ13F_a6sCMUkqjMjxPEgdtuFnTC4E8HXHwHTENpkn_NEnZKmsO7B8t4v1VL8PRaoILIiyNq-JrvRhDk911QVfb_SzQupPLbiieNGvddQCDve8mnbJkPj4bA_ikem7q-KFz2IhZj7nWFpWxgtVSrU"
	authTTL    = 300 * time.Second
	unauthTTL  = 60 * time.Second
	cacheSize  = 1024
	bcryptCost = 4
)

// TODO
// tokenid - auth - cache hit
// tokenid - auth - cache miss
// tokenid - unauth - cache hit
// tokenid - unauth - cache miss
// credentials - auth - cache hit
// credentials - auth - cache miss
// credentials - unauth - cache hit
// credentials - unauth - cache miss

func TestAuthWithTokenID(t *testing.T) {
	k := mock.Keystone{
		TokenWrapper: bouncer.KeystoneTokenWrapper{
			TokenID: tokenID,
			Token: bouncer.KeystoneToken{
				User: bouncer.KeystoneUser{
					Name: username,
					ID:   UID,
				},
			},
		},
		ProjectID: projectID,
	}
	reqBody := `
	{
	  "apiVersion": "authentication.k8s.io/v1beta1",
	  "kind": "TokenReview",
	  "spec": {
	    "token": "gAAAAABYt5PWnXGdQUWq6sXo8sj_n_1cOHbQ13F_a6sCMUkqjMjxPEgdtuFnTC4E8HXHwHTENpkn_NEnZKmsO7B8t4v1VL8PRaoILIiyNq-JrvRhDk911QVfb_SzQupPLbiieNGvddQCDve8mnbJkPj4bA_ikem7q-KFz2IhZj7nWFpWxgtVSrU"
	  }
	}`
	r := policy.New()
	a, _ := authn.New(k, projectID, authTTL, unauthTTL, cacheSize, bcryptCost, r)
	req, _ := http.NewRequest("POST", "http://example.com/v1/authn", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("authenticator returned an HTTP error code:", w.Code)
	}

	expJson := `
	{
	    "apiVersion": "authentication.k8s.io/v1beta1",
	    "kind": "TokenReview",
	    "status": {
			"authenticated": true,
			"user": {
				"uid": "dummy-user-id",
				"username": "dummy-username"
			}
	    }
	}`
	expected := &authn.TokenReview{}
	if err := json.NewDecoder(strings.NewReader(expJson)).Decode(expected); err != nil {
		t.Fatal("failed to decode expected response")
	}
	response := &authn.TokenReview{}
	if err := json.NewDecoder(w.Body).Decode(response); err != nil {
		t.Fatal("failed to decode response")
	}

	// Compare the decoded response and expected response, in part
	// because comparing JSON has pitfalls (unordered keys, whitespace)
	if !reflect.DeepEqual(expected, response) {
		t.Fatalf("\nexpected: %v\nresponse: %v", expected, response)
	}
}

func TestAuthWithCredentials(t *testing.T) {
	k := mock.Keystone{
		Username: username,
		Password: password,
		TokenWrapper: bouncer.KeystoneTokenWrapper{
			Token: bouncer.KeystoneToken{
				User: bouncer.KeystoneUser{
					Name: username,
					ID:   UID,
				},
			},
		},
		ProjectID: projectID,
	}
	reqBody := `
	{
	  "apiVersion": "authentication.k8s.io/v1beta1",
	  "kind": "TokenReview",
	  "spec": {
	    "token": "eyJ1c2VybmFtZSI6ICJkdW1teS11c2VybmFtZSIsICJwYXNzd29yZCI6ICJkdW1teS1wYXNzd29yZCJ9"
	  }
	}`
	r := policy.New()
	a, _ := authn.New(k, projectID, authTTL, unauthTTL, cacheSize, bcryptCost, r)
	req, _ := http.NewRequest("POST", "http://example.com/v1/authn", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("authenticator returned an HTTP error code:", w.Code)
	}

	expJson := `
	{
	    "apiVersion": "authentication.k8s.io/v1beta1",
	    "kind": "TokenReview",
	    "status": {
			"authenticated": true,
			"user": {
				"uid": "dummy-user-id",
				"username": "dummy-username"
			}
	    }
	}`
	expected := &authn.TokenReview{}
	if err := json.NewDecoder(strings.NewReader(expJson)).Decode(expected); err != nil {
		t.Fatal("failed to decode expected response")
	}
	response := &authn.TokenReview{}
	if err := json.NewDecoder(w.Body).Decode(response); err != nil {
		t.Fatal("failed to decode response")
	}

	// Compare the decoded response and expected response, in part
	// because comparing JSON has pitfalls (unordered keys, whitespace)
	if !reflect.DeepEqual(expected, response) {
		t.Fatalf("\nexpected: %v\nresponse: %v", expected, response)
	}
}
