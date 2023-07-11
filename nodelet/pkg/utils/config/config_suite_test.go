package config_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/spf13/viper"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test config.go", func() {
	var (
		ctx            context.Context
		expectedConfig *config.Config
	)

	BeforeEach(func() {
		ctx = context.TODO()
		expectedConfig = &config.Config{
			Debug:                     "false",
			ClusterRole:               "master",
			ClusterID:                 "fake-id",
			HostID:                    "fake-id",
			TransportURL:              "localhost:6264",
			ConnectTimeout:            30,
			KubeServiceState:          constants.ServiceStateTrue,
			FullRetryCount:            10,
			UseCgroups:                true,
			CgroupsV2:                 false,
			PhaseRetry:                3,
			CPULimit:                  40,
			PF9StatusThresholdSeconds: 30,
		}
	})

	AfterEach(func() {
		viper.Reset()
		ctx.Done()
	})

	Context("Test GetConfigFromDir", func() {
		It("should successfully load a single config file", func() {
			config.DefaultConfig = config.Config{}
			cfg, err := config.GetConfigFromDir("testdata/singleConfig/")
			assert.Equal(GinkgoT(), expectedConfig, cfg)
			assert.Nil(GinkgoT(), err)
		})

		It("should successfully load multiple config files", func() {
			config.DefaultConfig = config.Config{}
			expectedConfig.Debug = "true"
			cfg, err := config.GetConfigFromDir("testdata/multipleConfig/")
			assert.Equal(GinkgoT(), expectedConfig, cfg)
			assert.Nil(GinkgoT(), err)
		})

		It("should return empty config when config dir is missing", func() {
			config.DefaultConfig = config.Config{}
			emptyCfg := &config.Config{}
			cfg, err := config.GetConfigFromDir("testdata/absentDir/")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), cfg, emptyCfg)
		})

		It("should return empty config when config dir is empty", func() {
			config.DefaultConfig = config.Config{}
			emptyCfg := &config.Config{}
			cfg, err := config.GetConfigFromDir("testdata/emptyConfig/")
			assert.Error(GinkgoT(), err)
			assert.Equal(GinkgoT(), cfg, emptyCfg)
		})
	})

	Context("Test IsDebug", func() {
		It("should return true for a valid true string", func() {
			fakeCfg := config.Config{}
			fakeCfg.Debug = "true"
			assert.True(GinkgoT(), fakeCfg.IsDebug())
		})

		It("should return false for a valid false string", func() {
			fakeCfg := config.Config{}
			fakeCfg.Debug = "false"
			assert.False(GinkgoT(), fakeCfg.IsDebug())
		})

		It("should return false for an invalid false string", func() {
			fakeCfg := config.Config{}
			fakeCfg.Debug = "false123"
			assert.False(GinkgoT(), fakeCfg.IsDebug())
		})

		It("should return false for an invalid true string", func() {
			fakeCfg := config.Config{}
			fakeCfg.Debug = "true123"
			assert.False(GinkgoT(), fakeCfg.IsDebug())
		})
	})
})
