package containerruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Load user images to container runtime phase", func() {

	var (
		mockCtrl        *gomock.Controller
		fakePhase       *LoadImagePhase
		ctx             context.Context
		fakeCfg         *config.Config
		fakeRuntimeUtil *mocks.MockRuntime
		fakeFileUtility *mocks.MockFileInterface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewLoadImagePhase()
		ctx = context.TODO()
		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
		fakeRuntimeUtil = mocks.NewMockRuntime(mockCtrl)
		fakeFileUtility = mocks.NewMockFileInterface(mockCtrl)
		fakePhase.runtime = fakeRuntimeUtil
		fakePhase.fileUtility = fakeFileUtility
		fakeCfg.UserImagesDir = "testdata"
	})
	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("Validates status command", func() {
		It("Fails if could not verify checksum", func() {
			err := errors.New("error")
			fakeFileUtility.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(false, err).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), err, reterr)
		})
		It("Fails if check is false but could not load images", func() {
			err := errors.New("error")
			fakeFileUtility.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(false, nil).AnyTimes()
			fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(err).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), err, reterr)
		})
		It("succeds if check is false and successfully loads images", func() {
			fakeFileUtility.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(false, nil).AnyTimes()
			fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(nil).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})
		It("succeds if check is true", func() {
			fakeFileUtility.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(true, nil).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})

	})

	Context("Validates start command", func() {

		Context("When Checksum file does not exists", func() {
			BeforeEach(func() {
				fakeCfg.UserImagesDir = "absentdir"
			})

			It("Fails if could not generates checksum", func() {
				err := errors.New("error")
				fakeFileUtility.EXPECT().GenerateChecksum(fakeCfg.UserImagesDir).Return(err).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), err, reterr)
			})
			It("Fails if generates checksum but could not load images", func() {
				err := errors.New("error")
				fakeFileUtility.EXPECT().GenerateChecksum(fakeCfg.UserImagesDir).Return(nil).AnyTimes()
				fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(err).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), err, reterr)
			})
			It("Succeeds if generates checksum and loads images", func() {
				fakeFileUtility.EXPECT().GenerateChecksum(fakeCfg.UserImagesDir).Return(nil).AnyTimes()
				fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(nil).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), reterr)
			})
		})
		Context("When Checksum file exists", func() {

			It("Fails if could not load images", func() {
				err := errors.New("error")
				fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(err).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), err, reterr)
			})
			It("Succeeds if loads images", func() {
				fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(nil).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), reterr)
			})
		})
	})
})
