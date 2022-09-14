package containerruntime

import (
	"context"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("Test Configure Containerd phase", func() {

	var (
		mockCtrl      *gomock.Controller
		fakePhase     *ContainerdConfigPhase
		ctx           context.Context
		fakeCfg       *config.Config
		fakecmd       *mocks.MockCLI
		fakeFileUtils *mocks.MockFileInterface
		configFile    string
		file          fileio.FileInterface
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewContainerdConfigPhase()
		ctx = context.TODO()

		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false

		fakecmd = mocks.NewMockCLI(mockCtrl)
		fakePhase.cmd = fakecmd
		fakeFileUtils = mocks.NewMockFileInterface(mockCtrl)
		file = fileio.New()
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
		It("to succeed", func() {
			ret := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})
	Context("Validates start command", func() {
		BeforeEach(func() {
			file := fileio.New()
			err := os.MkdirAll("testdata", os.ModePerm)
			assert.Nil(GinkgoT(), err)
			configFile = "testdata/config.toml"
			constants.ContainerdCgroup = "systemd"
			err = file.TouchFile(configFile)
			assert.Nil(GinkgoT(), err)
			constants.ContainerdConfigFile = configFile
		})
		AfterEach(func() {
			err := os.RemoveAll("testdata")
			assert.Nil(GinkgoT(), err)
		})
		It("fails if containerd is not installed ", func() {
			err := errors.New("fake error")
			exitcode := 1
			fakecmd.EXPECT().RunCommandWithStdOut(ctx, nil, 0, "", constants.ContainerdBinPath, "--version").Return(exitcode, nil, err).AnyTimes()
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
		})
		It("fails if containerd config file is absent ", func() {
			fakecmd.EXPECT().RunCommandWithStdOut(ctx, nil, 0, "", constants.ContainerdBinPath, "--version").Return(0, nil, nil).AnyTimes()
			constants.ContainerdConfigFile = "testdata/abc.toml"
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
		})
		It("should write systemd cgroup as true in config if containerd cgroup is systemd", func() {
			fakecmd.EXPECT().RunCommandWithStdOut(ctx, nil, 0, "", constants.ContainerdBinPath, "--version").Return(0, nil, nil).AnyTimes()
			fakeFileUtils.EXPECT().WriteToFile("", "", true).Return(nil).AnyTimes()
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
			data := "\n\t[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runc.options]\n\t\tSystemdCgroup = true"
			fileData, _ := file.ReadFile(configFile)
			written := string(fileData)
			assert.Equal(GinkgoT(), data, written)
		})
		It("should write systemd cgroup as false in config if containerd cgroup is not systemd", func() {
			fakecmd.EXPECT().RunCommandWithStdOut(ctx, nil, 0, "", constants.ContainerdBinPath, "--version").Return(0, nil, nil).AnyTimes()
			fakeFileUtils.EXPECT().WriteToFile("", "", true).Return(nil).AnyTimes()
			constants.ContainerdCgroup = ""
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), reterr)
			data := "\n\t[plugins.\"io.containerd.grpc.v1.cri\".containerd.runtimes.runc.options]\n\t\tSystemdCgroup = false"
			fileData, _ := file.ReadFile(configFile)
			written := string(fileData)
			assert.Equal(GinkgoT(), data, written)
		})
	})
})
