package containerruntime

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
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
		fakeRuntimeUtil *mocks.MockImageUtils
		fakeFileUtils   *mocks.MockFileInterface
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
		fakeRuntimeUtil = mocks.NewMockImageUtils(mockCtrl)
		fakeFileUtils = mocks.NewMockFileInterface(mockCtrl)
		fakePhase.imageUtil = fakeRuntimeUtil
		fakePhase.fileUtils = fakeFileUtils
		fakeCfg.UserImagesDir = "testdata"
		constants.UserImagesDir = "testdata"
	})
	AfterEach(func() {
		err := os.RemoveAll("testdata")
		assert.Nil(GinkgoT(), err)
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("Validates status command", func() {
		BeforeEach(func() {
			fileInpOut := fileio.New()
			err := os.MkdirAll("testdata/checksum", os.ModePerm)
			assert.Nil(GinkgoT(), err)
			err = fileInpOut.WriteToFile("testdata/checksum/sha256sums.txt", "demo content", false)
			assert.Nil(GinkgoT(), err)
		})
		AfterEach(func() {
			err := os.Remove("testdata/checksum/sha256sums.txt")
			assert.Nil(GinkgoT(), err)
		})
		It("Fails if could not verify checksum", func() {
			err := errors.New("error")
			fakeFileUtils.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(false, err).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), err, reterr)
		})
		It("Fails if check is false but could not load images", func() {
			err := errors.New("error")
			fakeFileUtils.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(false, nil).AnyTimes()
			fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(err).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), err, reterr)
		})
		It("succeds if check is false and successfully loads images", func() {
			fakeFileUtils.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(false, nil).AnyTimes()
			fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(nil).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})
		It("succeds if check is true", func() {
			fakeFileUtils.EXPECT().VerifyChecksum(fakeCfg.UserImagesDir).Return(true, nil).AnyTimes()
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
		})

	})

	Context("Validates start command", func() {
		Context("When Checksum file does not exists", func() {

			It("Fails if could not generates checksum", func() {
				err := errors.New("error")
				fakeFileUtils.EXPECT().GenerateChecksum(fakeCfg.UserImagesDir).Return(err).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), err, reterr)
			})
			It("Fails if generates checksum but could not load images", func() {
				err := errors.New("error")
				fakeFileUtils.EXPECT().GenerateChecksum(fakeCfg.UserImagesDir).Return(nil).AnyTimes()
				fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(err).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), err, reterr)
			})
			It("Succeeds if generates checksum and loads images", func() {
				fakeFileUtils.EXPECT().GenerateChecksum(fakeCfg.UserImagesDir).Return(nil).AnyTimes()
				fakeRuntimeUtil.EXPECT().LoadImagesFromDir(ctx, fakeCfg.UserImagesDir, constants.K8sNamespace).Return(nil).AnyTimes()
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), reterr)
			})
		})
		Context("When Checksum file exists", func() {
			var fileInpOut fileio.FileInterface
			BeforeEach(func() {
				fileInpOut = fileio.New()
				err := os.MkdirAll("testdata/checksum", os.ModePerm)
				assert.Nil(GinkgoT(), err)
				err = fileInpOut.WriteToFile("testdata/checksum/sha256sums.txt", "demo content", false)
				assert.Nil(GinkgoT(), err)
			})
			AfterEach(func() {
				err := os.Remove("testdata/checksum/sha256sums.txt")
				assert.Nil(GinkgoT(), err)
			})
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
