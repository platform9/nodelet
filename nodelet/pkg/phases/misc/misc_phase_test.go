package misc

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

var _ = Describe("Test Misc phase", func() {

	var (
		mockCtrl           *gomock.Controller
		fakePhase          *MiscPhase
		ctx                context.Context
		fakeCfg            *config.Config
		fakeKubeUtils      *mocks.MockUtils
		fakeNetUtils       *mocks.MockNetInterface
		fakeNode           *v1.Node
		fakeNodeIdentifier string
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewMiscPhase()
		ctx = context.TODO()

		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false

		fakeKubeUtils = mocks.NewMockUtils(mockCtrl)
		fakeNetUtils = mocks.NewMockNetInterface(mockCtrl)
		fakePhase.kubeUtils = fakeKubeUtils
		fakePhase.netUtils = fakeNetUtils

	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("Validates stop command", func() {
		It("To succeed", func() {
			ret := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("Validates status command", func() {

		It("Fails when can't get nodeIdentifier or its null", func() {
			err := errors.New("fake error")
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, err).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when nodeIdentifier is 127.0.0.1", func() {
			err := fmt.Errorf("node interface might have lost IP address. Failing")
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("127.0.0.1", nil).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when k8s API server is unavailable", func() {
			err := errors.New("fake error")
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(err).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()

			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when can't get node from k8s api", func() {

			err := errors.New("fake error")
			fakeNetUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().K8sApiAvailable(*fakeCfg).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, err).Times(1)
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})

	})
	Context("Validates start command", func() {
		It("fails if could not write cloud config file", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().WriteCloudProviderConfig(*fakeCfg).Return(err).AnyTimes()
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), ret)
		})
		It("succeds if writes cloud config file", func() {
			fakeKubeUtils.EXPECT().WriteCloudProviderConfig(*fakeCfg).Return(nil).AnyTimes()
			fakeKubeUtils.EXPECT().IsInterfaceNil().Return(false).AnyTimes()
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

})
