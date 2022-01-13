// Package hosts contains common helper function to do with hosts.
package hosts

import (
	"github.com/google/uuid"
)

// TODO(erwin) use this function to generate the UID of all hosts at some point
// GenerateID creates a new, valid UUID to use as a HostID.
func GenerateID() string {
	return uuid.New().String()
}
