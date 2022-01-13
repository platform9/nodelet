package phases_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"

	"github.com/platform9/nodelet/mocks"
	"github.com/platform9/nodelet/pkg/phases"
	"github.com/platform9/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test phases.go", func() {
	var (
		mockCtrl        *gomock.Controller
		origCmd         = phases.GetLocalCmd
		origBaseCommand []string
		ctx             context.Context
		fakeCfg         *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		origCmd = phases.GetLocalCmd
		origBaseCommand = constants.BaseCommand
		ctx = context.TODO()

		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
	})

	AfterEach(func() {
		mockCtrl.Finish()
		viper.Reset()
		phases.GetLocalCmd = origCmd
		constants.BaseCommand = origBaseCommand
		ctx.Done()
	})

	Context("with cgroups enabled", func() {
		It("fetches master phases", func() {
			fakeCfg.UseCgroups = true
			fakeCfg.DisableScripts = false
			fakeCfg.ClusterRole = "master"
			setupCgroupCmdMocks(true, mockCtrl, ctx)
			phases, err := phases.InitAndLoadRolePhases(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
			assert.NotEmpty(GinkgoT(), phases)
		})
		It("fetches worker phases", func() {
			fakeCfg.UseCgroups = true
			fakeCfg.DisableScripts = false
			fakeCfg.ClusterRole = "worker"
			setupCgroupCmdMocks(true, mockCtrl, ctx)
			phases, err := phases.InitAndLoadRolePhases(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
			assert.NotEmpty(GinkgoT(), phases)
		})
		It("fetches no role phases", func() {
			fakeCfg.UseCgroups = true
			fakeCfg.DisableScripts = false
			fakeCfg.ClusterRole = "none"
			setupCgroupCmdMocks(true, mockCtrl, ctx)
			phases, err := phases.InitAndLoadRolePhases(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
			assert.NotEmpty(GinkgoT(), phases)
		})
	})

	Context("with cgroups disabled", func() {
		It("fetches master phases", func() {
			fakeCfg.UseCgroups = false
			fakeCfg.DisableScripts = false
			fakeCfg.ClusterRole = "master"
			setupCgroupCmdMocks(false, mockCtrl, ctx)
			phases, err := phases.InitAndLoadRolePhases(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
			assert.NotEmpty(GinkgoT(), phases)
		})
		It("fetches worker phases", func() {
			fakeCfg.UseCgroups = false
			fakeCfg.DisableScripts = false
			fakeCfg.ClusterRole = "worker"
			setupCgroupCmdMocks(false, mockCtrl, ctx)
			phases, err := phases.InitAndLoadRolePhases(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
			assert.NotEmpty(GinkgoT(), phases)
		})
		It("fetches no role phases", func() {
			fakeCfg.UseCgroups = false
			fakeCfg.DisableScripts = false
			fakeCfg.ClusterRole = "none"
			setupCgroupCmdMocks(false, mockCtrl, ctx)
			phases, err := phases.InitAndLoadRolePhases(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
			assert.NotEmpty(GinkgoT(), phases)
		})
	})
})

func setupCgroupCmdMocks(enabled bool, mockCtrl *gomock.Controller, ctx context.Context) *mocks.MockCLI {
	mockCmd := mocks.NewMockCLI(mockCtrl)
	phases.GetLocalCmd = func() command.CLI {
		return mockCmd
	}
	cmdCount := 0
	if enabled {
		cmdCount = 1
	}
	mockCmd.EXPECT().RunCommand(ctx, nil, -1, "", constants.CgroupCreateCmd[0], constants.CgroupCreateCmd[1], gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil).Times(cmdCount)
	mockCmd.EXPECT().RunCommand(ctx, nil, -1, "", constants.CgroupPeriodCmd[0], constants.CgroupPeriodCmd[1], gomock.Any(), gomock.Any(), gomock.Any()).Return(0, nil).Times(cmdCount)
	defaultCPUQuota := fmt.Sprintf(constants.CgroupQuotaParam, 400000)
	mockCmd.EXPECT().RunCommand(ctx, nil, -1, "", constants.CgroupQuotaCmd[0], constants.CgroupQuotaCmd[1], gomock.Any(), defaultCPUQuota, gomock.Any()).Return(0, nil).Times(cmdCount)
	return mockCmd
}
