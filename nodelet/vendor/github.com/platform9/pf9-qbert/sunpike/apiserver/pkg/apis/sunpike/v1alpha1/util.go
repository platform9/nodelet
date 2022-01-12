package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Convenience function to avoid the need to import the metav1 package for the
// common time conversion.
func NewTime(time time.Time) metav1.Time {
	return metav1.NewTime(time)
}
