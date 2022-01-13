package authn

import (
	"fmt"
)

const (
	TokenReviewKind       = "TokenReview"
	TokenReviewAPIVersion = "authentication.k8s.io/v1beta1"
)

type (
	TokenReview struct {
		APIVersion string             `json:"apiVersion"`
		Kind       string             `json:"kind"`
		Spec       *TokenReviewSpec   `json:"spec,omitempty"`
		Status     *TokenReviewStatus `json:"status,omitempty"`
	}

	TokenReviewSpec struct {
		Token string `json:"token"`
	}

	TokenReviewStatus struct {
		Authenticated bool                   `json:"authenticated"`
		User          *TokenReviewStatusUser `json:"user,omitempty"`
	}

	TokenReviewStatusUser struct {
		Username string   `json:"username"`
		UID      string   `json:"uid"`
		Groups   []string `json:"groups,omitempty"`
	}
)

func (t *TokenReview) validateMetadata() error {
	if t.Kind != TokenReviewKind {
		return fmt.Errorf("validate metadata: invalid Kind %v", t.Kind)
	}
	if t.APIVersion != TokenReviewAPIVersion {
		return fmt.Errorf("validate metadata: invalid APIVersion %v", t.APIVersion)
	}
	return nil
}

func (t *TokenReview) ValidateRequest() error {
	if err := t.validateMetadata(); err != nil {
		return err
	}
	if t.Spec == nil {
		return fmt.Errorf("validate request: missing Spec")
	}
	return nil
}

func (t *TokenReview) ValidateResponse() error {
	if err := t.validateMetadata(); err != nil {
		return err
	}
	if t.Status == nil {
		return fmt.Errorf("validate response: missing Status")
	}
	return nil
}
