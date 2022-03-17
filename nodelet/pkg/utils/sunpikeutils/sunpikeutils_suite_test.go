package sunpikeutils_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/sunpikeutils"
	"github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestSunpikeUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test sunpikeutils.go", func() {
	var (
		mockCtrl   *gomock.Controller
		ctx        context.Context
		cfg        *config.Config
		phaseNames []string = []string{"phase1", "phase2", "phase3"}
	)

	BeforeEach(func() {
		ctx = context.TODO()
		mockCtrl = gomock.NewController(GinkgoT())
		cfg = &config.Config{
			Debug:                     "false",
			ClusterRole:               constants.RoleMaster,
			ClusterID:                 "fake-id",
			HostID:                    "fake-id",
			TransportURL:              "localhost:6264",
			ConnectTimeout:            1,
			KubeServiceState:          constants.ServiceStateTrue,
			FullRetryCount:            10,
			UseCgroups:                true,
			PhaseRetry:                3,
			CPULimit:                  40,
			PF9StatusThresholdSeconds: 30,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("when running InitOrGetSunpikeClient", func() {

		It("should return error since sunpike unavailable but host object is populated", func() {
			spClient, err := getClient(cfg, mockCtrl, phaseNames)
			// No sunpike server to connect to so err should NOT be nil
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), spClient.Host.Name, cfg.HostID)
			assert.NotEmpty(GinkgoT(), spClient.Host.Name)
			for _, spPhase := range spClient.Host.Status.Phases {
				assert.Contains(GinkgoT(), phaseNames, spPhase.Name)
			}
		})
	})

	Context("When executing GetOrderForPhaseName", func() {
		It("should return correct order for valid phase name", func() {
			spClient, _ := getClient(cfg, mockCtrl, phaseNames)
			phaseOrders := []int32{10, 20, 30}
			for i, name := range phaseNames {
				o := spClient.GetOrderForPhaseName(name)
				assert.Equal(GinkgoT(), phaseOrders[i], o)
			}
		})
		It("should return -1 for invalid phase name", func() {
			spClient, _ := getClient(cfg, mockCtrl, phaseNames)
			o := spClient.GetOrderForPhaseName("invalid")
			assert.Equal(GinkgoT(), int32(-1), o)
		})

		It("should return -1 for empty string", func() {
			spClient, _ := getClient(cfg, mockCtrl, phaseNames)
			o := spClient.GetOrderForPhaseName("")
			assert.Equal(GinkgoT(), int32(-1), o)
		})
	})

	// TODO: Add unit tests for Update() function
})

func getClient(cfg *config.Config, mockCtrl *gomock.Controller, phaseNames []string) (*sunpikeutils.Wrapper, error) {
	phases := []phases.PhaseInterface{}
	var i int32 = 10
	for i = 10; i <= 30; i += 10 {
		phase := mocks.NewMockPhaseInterface(mockCtrl)
		name := phaseNames[i/10-1]
		hostphase := v1alpha1.HostPhase{
			Name:  name,
			Order: int32(i),
		}
		phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
		phase.EXPECT().GetHostPhase().Return(hostphase).AnyTimes()
		phases = append(phases, phase)
	}
	return sunpikeutils.InitOrGetSunpikeClient(phases, *cfg, v1alpha1.HostSpec{})
}
