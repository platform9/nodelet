package phaseutils

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Phase Utils", func() {

	var (
		fakeHostPhase *sunpikev1alpha1.HostPhase
		status        string
		message       string
	)

	Context("Validates Status of HostPhase ", func() {

		It("Should update status and message of hostphase", func() {
			fakeHostPhase = &sunpikev1alpha1.HostPhase{}
			status = constants.RunningState
			message = "new message"
			SetHostStatus(fakeHostPhase, status, message)
			assert.Equal(GinkgoT(), status, fakeHostPhase.Status)
			assert.Equal(GinkgoT(), message, fakeHostPhase.Message)
		})

	})
})
