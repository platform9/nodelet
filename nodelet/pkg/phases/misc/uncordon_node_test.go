package misc

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/stretchr/testify/assert"

	"github.com/onsi/ginkgo/reporters"
)

func TestCommandUncordonNode(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Apply and validate node taints phase", func() {

	var (
		mockCtrl      *gomock.Controller
		fakePhase     *UncordonNodePhasev2
		ctx           context.Context
		fakeCfg       *config.Config
		fakeKubeUtils *mocks.MockUtils
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewUncordonNodePhaseV2()
		ctx = context.TODO()

		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
		fakeKubeUtils = mocks.NewMockUtils(mockCtrl)
		fakePhase.kubeUtils = fakeKubeUtils
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("validates stop command", func() {
		It("to succeed", func() {
			ret := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("validates status command", func() {
		It("fails when nodeIdentifier is null", func() {
			err := errors.New("fake error")

			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("", err).Times(1)

			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})

	})

})
