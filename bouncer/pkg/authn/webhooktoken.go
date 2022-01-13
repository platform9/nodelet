package authn

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const (
	fernetVersion = 128
)

var IsKeystoneTokenID = IsUnpaddedFernetToken

// Attempts to decode an unpadded Fernet token and verify that
// the first byte contains the Fernet version. The official Fernet
// spec requires padding, but Keystone omits it by design.
func IsUnpaddedFernetToken(t string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(t)
	if err != nil {
		return false
	}
	if len(decoded) < 1 {
		return false
	}
	magic := byte(decoded[0])
	return magic == fernetVersion
}

func Credentials(t string) (string, string, error) {
	var creds struct {
		Username string
		Password string
	}
	decoded, err := base64.StdEncoding.DecodeString(t)
	if err != nil {
		return "", "", fmt.Errorf("decode credentials: %v", err)
	}
	if err := json.Unmarshal(decoded, &creds); err != nil {
		return "", "", fmt.Errorf("unmarshal credentials: %v", err)
	}
	return creds.Username, creds.Password, nil
}
