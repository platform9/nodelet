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
		fakePhase     *DrainNodePhase
		ctx           context.Context
		fakeCfg       *config.Config
		fakeKubeUtils *mocks.MockUtils
		fakeNetUtils  *mocks.MockNetInterface
		nodeName      string
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewDrainNodePhase()
		fakeKubeUtils = mocks.NewMockUtils(mockCtrl)
		fakeNetUtils = mocks.NewMockNetInterface(mockCtrl)
		fakePhase.kubeUtils = fakeKubeUtils
		fakePhase.netUtils = fakeNetUtils
		ctx = context.TODO()
		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
		nodeName = "10.28.243.97"
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("Validates status command", func() {
		It("to succeed", func() {
			ret := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("Validates stop command", func() {
		It("Fails when k8s API server is unavailable", func() {
			err := errors.New("fake error")

			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(err).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when nodeIdentifier is invalid (non-exist)", func() {
			err := errors.New("fake error")
			nodeName = "8.8.8.8"
			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(nil).Times(1)
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeName, err).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})

		It("Fails if can't drain node", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(nil).Times(1)
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeName, nil).Times(1)
			fakeKubeUtils.EXPECT().DrainNodeFromApiServer(ctx, nodeName).Return(err).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails if can't add 'KubeStackShutDown' annotation", func() {
			err := errors.New("fake error")
			annotsToAdd := map[string]string{
				"KubeStackShutDown": "true",
			}
			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(nil).Times(1)
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeName, nil).Times(1)
			fakeKubeUtils.EXPECT().DrainNodeFromApiServer(ctx, nodeName).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().AddAnnotationsToNode(ctx, nodeName, annotsToAdd).Return(err).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()

			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Succeeds", func() {
			annotsToAdd := map[string]string{
				"KubeStackShutDown": "true",
			}
			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(nil).Times(1)
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeName, nil).Times(1)
			fakeKubeUtils.EXPECT().DrainNodeFromApiServer(ctx, nodeName).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().AddAnnotationsToNode(ctx, nodeName, annotsToAdd).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()

			ret := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("validates start command", func() {
		BeforeEach(func() {
			_, _ = os.Create("testdata/dummy.txt")
		})
		It("succeeds if KubeStackStartFileMarker is not present", func() {
			constants.KubeStackStartFileMarker = "testdata/abc.txt"
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
		It("succeeds if KubeStackStartFileMarker is present", func() {
			constants.KubeStackStartFileMarker = "testdata/dummy.txt"
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})
})
