package authn_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/platform9/pf9-qbert/bouncer/pkg/authn"
)

var (
	fernetToken = "gAAAAABYt5PWnXGdQUWq6sXo8sj_n_1cOHbQ13F_a6sCMUkqjMjxPEgdtuFnTC4E8HXHwHTENpkn_NEnZKmsO7B8t4v1VL8PRaoILIiyNq-JrvRhDk911QVfb_SzQupPLbiieNGvddQCDve8mnbJkPj4bA_ikem7q-KFz2IhZj7nWFpWxgtVSrU"
	credsToken  = encodeCredentials("dummy-username", "dummy-password")
)

func encodeCredentials(username, password string) string {
	j := fmt.Sprintf(`{ "username": "%s", "password": "%s" }`, username, password)
	return base64.StdEncoding.EncodeToString([]byte(j))
}

func TestIsFernetToken(t *testing.T) {
	if !authn.IsUnpaddedFernetToken(fernetToken) {
		t.Errorf("IsFernetToken did not recognize fernet token")
	}
	if authn.IsUnpaddedFernetToken(credsToken) {
		t.Errorf("IsFernetToken mistook credentials token for a fernet token")
	}
}

func TestCredentials(t *testing.T) {
	username, password, err := authn.Credentials(credsToken)
	if err != nil {
		t.Errorf("error: %v", err)
	}
	if username != "dummy-username" || password != "dummy-password" {
		t.Errorf("decoded wrong credentials")
	}
}
