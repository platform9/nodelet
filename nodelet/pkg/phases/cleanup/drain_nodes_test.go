package cleanup

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/mocks"
	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/stretchr/testify/assert"

	"github.com/onsi/ginkgo/reporters"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Drain nodes phase", func() {

	var (
		mockCtrl  *gomock.Controller
		fakePhase *DrainNodePhasev2
		ctx       context.Context
		fakeCfg   *config.Config
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewDrainNodePhaseV2()
		ctx = context.TODO()

		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("validates status command", func() {
		It("to succeed", func() {
			ret := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("validates stop command", func() {
		It("fails when k8s API server is unavailable", func() {
			fakeKubeUtils := mocks.NewMockUtils(mockCtrl)
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().KubernetesApiAvailable(fakeCfg).Return(err).Times(1)
			fakePhase.kubeUtils = fakeKubeUtils

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr.Error(), err.Error())
		})
	})
})
