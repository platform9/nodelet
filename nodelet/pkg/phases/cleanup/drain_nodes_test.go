package cleanup

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/stretchr/testify/assert"

	"github.com/onsi/ginkgo/reporters"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Drain nodes phase", func() {

	var (
		mockCtrl      *gomock.Controller
		fakePhase     *DrainNodePhasev2
		ctx           context.Context
		fakeCfg       *config.Config
		fakeKubeUtils *mocks.MockUtils
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewDrainNodePhaseV2()
		fakeKubeUtils = mocks.NewMockUtils(mockCtrl)
		fakePhase.kubeUtils = fakeKubeUtils
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
			err := errors.New("fake error")

			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(err).Times(1)

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("fails when nodeIdentifier is null", func() {
			err := errors.New("fake error")

			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("", err).Times(1)

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("succeeds when k8s api is available and drain node works", func() {

			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("10.28.243.97", nil).Times(1)
			fakeKubeUtils.EXPECT().DrainNodeFromApiServer(ctx, "10.28.243.97").Return(nil).Times(1)

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})

	})

	Context("validates start command", func() {
		BeforeEach(func() {
			_, _ = os.Create("testdata/dummy.txt")
		})
		AfterEach(func() {
			constants.KubeStackStartFileMarker = "var/opt/pf9/is_node_booting_up"
		})

		It("fails if KubeStackStartFileMarker is not present", func() {
			constants.KubeStackStartFileMarker = "testdata/abc.txt"
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), ret)
		})
		It("succeeds if KubeStackStartFileMarker is present", func() {
			constants.KubeStackStartFileMarker = "testdata/dummy.txt"
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})
})
