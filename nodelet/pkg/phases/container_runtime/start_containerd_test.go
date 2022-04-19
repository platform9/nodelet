package containerruntime

import (
	"context"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("Test Configure Containerd phase", func() {

	var (
		mockCtrl         *gomock.Controller
		fakePhase        *ContainerdRunPhase
		ctx              context.Context
		fakeCfg          *config.Config
		fakecmd          *mocks.MockCLI
		fakeServiceUtils *mocks.MockServiceUtil
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewContainerdRunPhase()
		ctx = context.TODO()
		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
		fakecmd = mocks.NewMockCLI(mockCtrl)
		fakePhase.cmd = fakecmd
		fakeServiceUtils = mocks.NewMockServiceUtil(mockCtrl)
		fakePhase.serviceUtil = fakeServiceUtils
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("Validates start command", func() {
		It("fails if could not start service", func() {
			err := errors.New("fake")
			fakeServiceUtils.EXPECT().RunAction(ctx, constants.StartOp).Return(nil, err).AnyTimes()
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("succeds if service starts", func() {
			fakeServiceUtils.EXPECT().RunAction(ctx, constants.StartOp).Return(nil, nil).AnyTimes()
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})
	})
	Context("Validates status command", func() {
		It("fails if could not check status of containerd", func() {
			err := errors.New("fake")
			fakeServiceUtils.EXPECT().RunAction(ctx, constants.IsActiveOp).Return(nil, err).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("fails if service status is not active", func() {
			fakeServiceUtils.EXPECT().RunAction(ctx, constants.IsActiveOp).Return([]string{"inactive"}, nil).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
		})
		It("succeds if service status is active", func() {
			fakeServiceUtils.EXPECT().RunAction(ctx, constants.IsActiveOp).Return([]string{"active"}, nil).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})
	})
	Context("Validates stop command", func() {
		It("suceeds if containerd is not present and ignores deleting containers as service not present", func() {
			err := errors.New("fake")
			fakecmd.EXPECT().RunCommand(ctx, nil, 0, "", constants.RuntimeContainerd, "--version").Return(1, err).AnyTimes()
			reterr := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})
	})
})
