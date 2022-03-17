package nodelet

import (
	"context"
	"errors"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/platform9/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/extensionfile"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/platform9/nodelet/nodelet/pkg/utils/sunpikeutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func TestNodeletd(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Nodeletd Suite", []Reporter{junitReporter})
}
func getHostPhase(name string, order int) sunpikev1alpha1.HostPhase {
	return sunpikev1alpha1.HostPhase{
		Name:  name,
		Order: int32(order),
	}
}

var _ = Describe("Nodeletd Tests", func() {
	var (
		mockCtrl             *gomock.Controller
		fakeKubeService      Nodelet
		fakefile             *mocks.MockFileInterface
		fakefileWriteCall    *gomock.Call
		fakefileReadJSONCall *gomock.Call
		fakeState            extensionfile.ExtensionData
		phaseNames           []string
		ctx                  context.Context
		origDefaults         config.Config
		fakeError            error = errors.New("fake error")
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakeKubeService = Nodelet{}
		fakefile = mocks.NewMockFileInterface(mockCtrl)
		fakefileWriteCall = fakefile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
		fakefileReadJSONCall = fakefile.EXPECT().ReadJSONFile(gomock.Any(), gomock.Any()).AnyTimes()
		fakeState = extensionfile.New(fakefile, constants.ExtensionOutputFile, &config.DefaultConfig)
		fakeKubeService.currentState = &fakeState
		phaseNames = []string{"phase1", "phase2", "phase3"}
		fakeKubeService.currentState.AllStatusChecks = phaseNames
		fakeKubeService.currentState.AllPhases = phaseNames
		fakeKubeService.phases = []phases.PhaseInterface{}
		fakeKubeService.config = &config.Config{}
		fakeKubeService.config.TransportURL = "localhost:6264"
		fakeKubeService.config.ConnectTimeout = 1
		fakeKubeService.config.GRPCRetryMax = 1
		fakeKubeService.config.DisableSunpike = true
		fakeKubeService.config.NumCmdOutputLinesToLog = 0
		fakeKubeService.config.ClusterRole = ""
		fakeKubeService.config.ClusterID = ""
		fakeKubeService.config.UseCgroups = false
		origDefaults = config.DefaultConfig
		ctx = context.TODO()
		logger, _ := zap.NewDevelopment()
		zap.ReplaceGlobals(logger)
		fakeKubeService.log = logger.Sugar()
		constants.GenCertsPhaseOrder = 0
	})

	AfterEach(func() {
		mockCtrl.Finish()
		viper.Reset()
		ctx.Done()
		config.DefaultConfig = origDefaults
	})

	It("Should stop entire chain of kube service", func() {
		// Should write to extension file at least once at the end of the stop chain
		fakefileWriteCall.MinTimes(1)
		fakefileReadJSONCall.MaxTimes(0)
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		err = fakeKubeService.Stop(ctx, 0, false)
		Expect(err).To(BeNil())
		Expect(fakeKubeService.currentState.StartAttempts).To(Equal(0))
		Expect(len(fakeKubeService.currentState.CompletedPhases)).To(Equal(0))
		Expect(fakeKubeService.currentState.CurrentPhase).To(Equal(""))
		Expect(fakeKubeService.currentState.LastFailedPhase).To(Equal(""))
	})

	It("Should stop entire chain of kube service with force set", func() {
		// Should write to extension file at least once at the end of the stop chain
		fakefileWriteCall.MinTimes(1)
		fakefileReadJSONCall.MaxTimes(0)
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		err = fakeKubeService.Stop(ctx, 0, true)
		Expect(err).To(BeNil())
		Expect(fakeKubeService.currentState.StartAttempts).To(Equal(0))
		Expect(len(fakeKubeService.currentState.CompletedPhases)).To(Equal(0))
		Expect(fakeKubeService.currentState.CurrentPhase).To(Equal(""))
		Expect(fakeKubeService.currentState.LastFailedPhase).To(Equal(""))
	})

	It("Should stop kube service chain partially", func() {
		// Should write to extension file at least once at the end of the stop chain
		fakefileWriteCall.MinTimes(1)
		fakefileReadJSONCall.MaxTimes(0)
		partialStopPhaseIndex := 1
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			if i >= partialStopPhaseIndex {
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
			}
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		err = fakeKubeService.Stop(ctx, partialStopPhaseIndex, false)
		Expect(err).To(BeNil())
	})

	It("Should fail the stop of last phase in stop kube", func() {
		// Should write to extension file at least once at the end of the stop chain
		fakefileWriteCall.MinTimes(1)
		fakefileReadJSONCall.MaxTimes(0)
		// 0th phase is the last phase to run in stop chain
		failPhaseIndex := 0
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			if i == failPhaseIndex {
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			} else {
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
			}
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		err = fakeKubeService.Stop(ctx, 0, false)
		Expect(err).ToNot(BeNil())
	})

	It("Should fail the stop of first phase in stop kube", func() {
		// Should write to extension file at least once at the end of the stop chain
		fakefileWriteCall.MinTimes(1)
		fakefileReadJSONCall.MaxTimes(0)
		// phase with highest order is the first phase to run in stop chain
		// None of the other phases should NOT even run.
		failPhaseIndex := 2
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			if i == failPhaseIndex {
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			}
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		err = fakeKubeService.Stop(ctx, 0, false)
		Expect(err).ToNot(BeNil())
	})

	It("Should fail the stop of first phase in stop kube but continue as force is set", func() {
		// Should write to extension file at least once at the end of the stop chain
		fakefileWriteCall.MinTimes(1)
		fakefileReadJSONCall.MaxTimes(0)
		// phase with highest order is the first phase to run in stop chain
		// None of the other phases should NOT even run.
		failPhaseIndex := 2
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			if i == failPhaseIndex {
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			} else {
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
			}
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		err = fakeKubeService.Stop(ctx, 0, true)
		Expect(err).To(BeNil())
	})

	It("Should successfully start kube", func() {
		// Should write 3 times at least to extension file for 3 phases
		fakefileWriteCall.MinTimes(3)
		fakefileReadJSONCall.MaxTimes(0)
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		lastPhaseIndex, err := fakeKubeService.Start(ctx, 0)
		Expect(err).To(BeNil())
		Expect(len(fakeKubeService.phases)).To(Equal(lastPhaseIndex))
		Expect(len(fakeKubeService.currentState.CompletedPhases)).To(Equal(lastPhaseIndex))
		Expect(fakeKubeService.currentState.CurrentPhase).To(Equal(""))
	})

	It("Should start kube from phase order 20 [index 1 in phases array]", func() {
		// Should write 3 times at least to extension file for 3 phases
		fakefileWriteCall.MinTimes(3)
		fakefileReadJSONCall.MaxTimes(0)
		startFromIndex := 1
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			if i >= startFromIndex {
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			}
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		lastPhaseIndex, err := fakeKubeService.Start(ctx, startFromIndex)
		Expect(err).To(BeNil())
		Expect(len(fakeKubeService.phases)).To(Equal(lastPhaseIndex))
		Expect(len(fakeKubeService.currentState.CompletedPhases)).To(Equal(lastPhaseIndex))
		Expect(fakeKubeService.currentState.CurrentPhase).To(Equal(""))
	})

	It("Should fail to execute start of first phase while starting kube", func() {
		// Should write 3 times at least to extension file for 3 phases
		fakefileWriteCall.MinTimes(3)
		fakefileReadJSONCall.MaxTimes(0)
		startFromIndex := 0
		failPhaseIndex := 0
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			if i == failPhaseIndex {
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			}
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		lastPhaseIndex, err := fakeKubeService.Start(ctx, startFromIndex)
		Expect(err).ToNot(BeNil())
		Expect(len(fakeKubeService.currentState.CompletedPhases)).To(Equal(lastPhaseIndex))
		Expect(fakeKubeService.currentState.CurrentPhase).To(Equal(""))
		Expect(fakeKubeService.currentState.LastFailedPhase).To(Equal(phaseNames[failPhaseIndex]))
	})

	It("Should fail to execute start of the 2nd phase [array index 1] while starting kube", func() {
		// Should write 3 times at least to extension file for 3 phases
		fakefileWriteCall.MinTimes(3)
		fakefileReadJSONCall.MaxTimes(0)
		startFromIndex := 0
		failPhaseIndex := 1
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			if i <= failPhaseIndex {
				if i == failPhaseIndex {
					phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
				} else {
					phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
				}
			}
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		lastPhaseIndex, err := fakeKubeService.Start(ctx, startFromIndex)
		Expect(err).ToNot(BeNil())
		Expect(failPhaseIndex).To(Equal(lastPhaseIndex))
		Expect(fakeKubeService.currentState.LastFailedPhase).To(Equal(phaseNames[failPhaseIndex]))
	})

	It("Is checking status of kube service when it is running", func() {
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			// If service is running all phases return exit code 0 - success
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		// Service was started successfully.
		fakeKubeService.currentState.StartFailStep = -1
		fakeKubeService.currentState.ServiceState = constants.ServiceStateTrue
		fakeKubeService.Status(ctx)
		Expect(fakeKubeService.currentState.KubeRunning).To(BeTrue())
		Expect(fakeKubeService.currentState.FailedStatusCheck).To(Equal(0))
	})

	It("Is checking status of kube service when status check fails for first phase", func() {
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			if i == 0 {
				// If service is running all phases return exit code 0 - success
				phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			}
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.currentState.StartFailStep = -1
		fakeKubeService.currentState.ServiceState = constants.ServiceStateTrue
		fakeKubeService.Status(ctx)
		Expect(fakeKubeService.currentState.KubeRunning).To(BeFalse())
		Expect(fakeKubeService.currentState.FailedStatusCheck).To(Equal(0))
		Expect(fakeKubeService.currentState.CurrentStatusCheckTime).ToNot(Equal(0))
		Expect(fakeKubeService.currentState.LastFailedCheckTime).ToNot(Equal(0))
		Expect(fakeKubeService.currentState.LastFailedCheck).To(Equal(phaseNames[0]))
	})

	It("Is checking kube service status when status check fails for middle phase", func() {
		failedphase := 1
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			if i <= failedphase {
				if i == failedphase {
					phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
				} else {
					phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).Times(1)
				}
			}
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.currentState.StartFailStep = -1
		fakeKubeService.currentState.ServiceState = constants.ServiceStateTrue
		fakeKubeService.Status(ctx)
		Expect(fakeKubeService.currentState.KubeRunning).To(BeFalse())
		Expect(fakeKubeService.currentState.FailedStatusCheck).To(Equal(failedphase))
		Expect(fakeKubeService.currentState.LastFailedCheck).To(Equal(phaseNames[failedphase]))
	})

	It("Is checking kube service status when preceeding start failed at phase at array index 1", func() {
		phaseFailedToStart := 1
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Times(0)
			phase.EXPECT().GetPhaseName().AnyTimes()
			// If service is running all phases return exit code 0 - success
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		// Service was started successfully.
		fakeKubeService.currentState.StartFailStep = phaseFailedToStart
		fakeKubeService.currentState.ServiceState = constants.ServiceStateTrue
		fakeKubeService.Status(ctx)
		Expect(fakeKubeService.currentState.FailedStatusCheck).To(Equal(phaseFailedToStart))
		Expect(fakeKubeService.currentState.KubeRunning).To(BeFalse())
		Expect(fakeKubeService.currentState.StartFailStep).To(Equal(phaseFailedToStart))
	})

	It("Is checking status fails and threshold is not reached", func() {
		fakeKubeService.config.PF9StatusThresholdSeconds = 10
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			if i == 0 {
				// If service is running all phases return exit code 0 - success
				phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			}
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.currentState.StartFailStep = -1
		fakeKubeService.currentState.LastSuccessfulStatus = time.Now()
		fakeKubeService.currentState.ServiceState = constants.ServiceStateTrue
		fakeKubeService.Status(ctx)
		// Since threshold is not reached, Status should return that service is running even though first phase is mocked to fail.
		Expect(fakeKubeService.currentState.KubeRunning).To(BeTrue())
		Expect(fakeKubeService.currentState.FailedStatusCheck).To(BeZero())
		Expect(fakeKubeService.currentState.CurrentStatusCheckTime).ToNot(BeZero())
		Expect(fakeKubeService.currentState.LastFailedCheck).To(Equal(phaseNames[0]))
	})

	It("Is checking status of kube service when it is NOT running", func() {
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			if i == 0 {
				// If service is running all phases return exit code 0 - success
				phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			}
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.currentState.StartFailStep = -1
		fakeKubeService.currentState.ServiceState = constants.ServiceStateFalse
		fakeKubeService.Status(ctx)
		Expect(fakeKubeService.currentState.KubeRunning).To(BeFalse())
		Expect(fakeKubeService.currentState.FailedStatusCheck).To(BeZero())
		Expect(fakeKubeService.currentState.CurrentStatusCheckTime).ToNot(BeZero())
		Expect(fakeKubeService.currentState.LastFailedCheckTime).ToNot(BeZero())
		Expect(fakeKubeService.currentState.LastFailedCheck).To(Equal(phaseNames[0]))
	})

	It("Is testing handleServiceStopState when service is already stopped", func() {
		// Simplest way to fake a stopped kube service is to set kubeService.currentState.StartFailStep to >= 0
		// This indicates that start was not successful
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())
		fakeKubeService.currentState.StartFailStep = 0
		fakeKubeService.handleServiceStopState(ctx)
		Expect(fakeKubeService.currentState.StartFailStep).To(Equal(0))
	})

	It("Is testing handleServiceStopState when service is running", func() {
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			// If service is running all phases return exit code 0 - success
			phasestatucCall := phase.EXPECT().Status(ctx, *fakeKubeService.config)
			phasestatucCall.Return(nil).Times(1)
			phase.EXPECT().Stop(ctx, *fakeKubeService.config).After(phasestatucCall).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		// Service was started successfully.
		fakeKubeService.currentState.StartFailStep = -1
		fakeKubeService.Status(ctx) // Ensure status is up-to-date
		fakeKubeService.handleServiceStopState(ctx)
		Expect(fakeKubeService.currentState.StartFailStep).To(Equal(-1))
	})

	It("Is testing handleServiceStartState when service is running", func() {
		clusterID := "fakeID"
		clusterRole := "master"
		fakeKubeService.config.ClusterRole = clusterRole
		fakeKubeService.config.ClusterID = clusterID
		fakeKubeService.config.FullRetryCount = 10
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			// If service is running all phases return exit code 0 - success
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.currentState.ClusterID = clusterID
		fakeKubeService.currentState.ClusterRole = clusterRole
		fakeKubeService.currentState.StartFailStep = -1
		fakeKubeService.Status(ctx) // Ensure status is up-to-date
		fakeKubeService.handleServiceStartState(ctx)
		Expect(fakeKubeService.currentState.StartFailStep).To(Equal(-1))
		Expect(fakeKubeService.currentState.StartAttempts).To(Equal(0))
		Expect(fakeKubeService.currentState.ServiceState).To(Equal(constants.ServiceStateTrue))
	})

	It("Is testing handleServiceStartState when service running with wrong config", func() {
		actualID := "actualID"
		expectedID := "expectedID"
		clusterRole := "master"
		fakeKubeService.config.ClusterRole = clusterRole
		fakeKubeService.config.ClusterID = expectedID
		fakeKubeService.config.FullRetryCount = 10
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			// If service is running all phases return exit code 0 - success
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).AnyTimes()
			// Stop should be called when restarting service with correct config
			phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
			phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.Status(ctx) // Ensure status is up-to-date
		fakeKubeService.currentState.ClusterID = actualID
		fakeKubeService.currentState.ClusterRole = clusterRole
		fakeKubeService.currentState.StartFailStep = -1

		fakeKubeService.handleServiceStartState(ctx)
		Expect(fakeKubeService.currentState.StartFailStep).To(Equal(-1))
		Expect(fakeKubeService.currentState.StartAttempts).To(Equal(0))
		Expect(fakeKubeService.currentState.ServiceState).To(Equal(constants.ServiceStateTrue))
		Expect(fakeKubeService.currentState.ClusterID).To(Equal(expectedID))
		Expect(fakeKubeService.currentState.ClusterRole).To(Equal(clusterRole))
	})

	It("Is testing handleServiceStartState when service it NOT running", func() {
		clusterID := "fakeID"
		clusterRole := "master"
		fakeKubeService.config.ClusterRole = clusterRole
		fakeKubeService.config.ClusterID = clusterID
		fakeKubeService.config.FullRetryCount = 10
		for i := 0; i < 3; i++ {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			if i == 0 {
				phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(fakeError).Times(1)
			}
			phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
			phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}

		fakeKubeService.currentState.ClusterID = clusterID
		fakeKubeService.currentState.ClusterRole = clusterRole
		fakeKubeService.currentState.StartFailStep = -1

		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.Status(ctx) // Ensure status is up-to-date
		fakeKubeService.handleServiceStartState(ctx)
		Expect(fakeKubeService.currentState.StartFailStep).To(Equal(-1))
		Expect(fakeKubeService.currentState.StartAttempts).To(BeZero())
		Expect(fakeKubeService.currentState.ServiceState).To(Equal(constants.ServiceStateTrue))
	})

	It("Is testing handleServiceStartState when service fails to start on first attempt", func() {
		// This test is to validate that on the second attempt we do NOT restart all the phases
		// loop increment is set to 20 so that we can fine tune failure and success of the middle phase
		clusterID := "fakeID"
		clusterRole := "master"
		fakeKubeService.currentState.ClusterID = clusterID
		fakeKubeService.currentState.ClusterRole = clusterRole
		fakeKubeService.config.ClusterRole = clusterRole
		fakeKubeService.config.ClusterID = clusterID
		fakeKubeService.config.FullRetryCount = 10
		fakeKubeService.phases = make([]phases.PhaseInterface, 3)
		for i := 0; i < 3; i += 2 {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).AnyTimes()
			if i == 0 {
				// Since only middle phase fails to start, first phase should not be stopped and started again
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			} else {
				// Stop of last phase should be called 2 times -
				// 1. When running the clean stop before 1st start
				// 2. When stopping the chain partially till phase ordered 20 as it failed once
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(2)
				// Start of last phase should only be called once during the partial start
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			}
			fakeKubeService.phases[i] = phase
		}
		phase := mocks.NewMockPhaseInterface(mockCtrl)
		name := phaseNames[1]
		phase.EXPECT().GetHostPhase().Return(getHostPhase(name, 20)).AnyTimes()
		phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
		phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).AnyTimes()
		firstStopCall := phase.EXPECT().Stop(ctx, *fakeKubeService.config)
		firstStopCall.Return(nil).Times(1)
		firstStartCall := phase.EXPECT().Start(ctx, *fakeKubeService.config).After(firstStopCall)
		firstStartCall.Return(fakeError).Times(1)
		// Stop will be called again as part of partially stopping chain till failed phase
		secondStopCall := phase.EXPECT().Stop(ctx, *fakeKubeService.config).After(firstStartCall)
		secondStopCall.Return(nil).Times(1)
		secondStartCall := phase.EXPECT().Start(ctx, *fakeKubeService.config).After(secondStopCall)
		// This time we make the phase successfully to indicate chain started successfully the second time
		secondStartCall.Return(nil).Times(1)
		fakeKubeService.phases[1] = phase

		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.Status(ctx) // Ensure status is up-to-date
		fakeKubeService.handleServiceStartState(ctx)
		// Actual service will sleep 30s after first call completes. But this is unit tests and we don't want to wait that long.
		fakeKubeService.Status(ctx) // Ensure status is up-to-date
		fakeKubeService.handleServiceStartState(ctx)
		Expect(fakeKubeService.currentState.StartFailStep).To(Equal(-1))
		Expect(fakeKubeService.currentState.StartAttempts).To(Equal(0))
		Expect(fakeKubeService.currentState.ServiceState).To(Equal(constants.ServiceStateTrue))
	})

	It("Is testing handleServiceStartState when service fails to start on 9th attempt", func() {
		clusterID := "fakeID"
		clusterRole := "master"
		fakeKubeService.currentState.ClusterID = clusterID
		fakeKubeService.currentState.ClusterRole = clusterRole
		fakeKubeService.config.ClusterRole = clusterRole
		fakeKubeService.config.ClusterID = clusterID
		fakeKubeService.config.FullRetryCount = 10
		fakeKubeService.phases = make([]phases.PhaseInterface, 3)
		// This test is to validate that on the tenth attempt we do restart all the phases
		// loop increment is set to 20 so that we can fine tune failure and success of the middle phase
		for i := 0; i < 3; i += 2 {
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).AnyTimes()
			if i == 0 {
				// Since only middle phase fails to start, first phase should not be stopped and started again
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(2)
			} else {
				// Stop of last phase should be called 2 times -
				// 1. When running the clean stop before 1st start
				// 2. When stopping the chain partially till phase ordered 20 as it failed once
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(2)
				// Start of last phase should only be called once during the 10th start
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			}
			fakeKubeService.phases[i] = phase
		}
		phase := mocks.NewMockPhaseInterface(mockCtrl)
		name := phaseNames[1]
		phase.EXPECT().GetHostPhase().Return(getHostPhase(name, 20)).AnyTimes()
		phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
		phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).AnyTimes()
		firstStopCall := phase.EXPECT().Stop(ctx, *fakeKubeService.config)
		firstStopCall.Return(nil).Times(1)
		firstStartCall := phase.EXPECT().Start(ctx, *fakeKubeService.config).After(firstStopCall)
		firstStartCall.Return(fakeError).Times(1)
		// Stop will be called again as part of partially stopping chain till failed phase
		secondStopCall := phase.EXPECT().Stop(ctx, *fakeKubeService.config).After(firstStartCall)
		secondStopCall.Return(nil).Times(1)
		secondStartCall := phase.EXPECT().Start(ctx, *fakeKubeService.config).After(secondStopCall)
		// This time we make the phase successfully to indicate chain started successfully the second time
		secondStartCall.Return(nil).Times(1)
		fakeKubeService.phases[1] = phase

		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.handleServiceStartState(ctx)
		fakeKubeService.currentState.StartAttempts = 9
		fakeKubeService.Status(ctx)
		// Actual service will sleep 30s after first call completes. But this is unit tests and we don't want to wait that long.
		fakeKubeService.handleServiceStartState(ctx)
	})

	It("validates that 0th phase is not invoked on a full restart triggered by 9 failed partial restarts", func() {
		clusterID := "fakeID"
		clusterRole := "master"
		phaseNames = []string{"phase1", "phase2", "phase3", "phase4"}
		fakeKubeService.currentState.AllStatusChecks = phaseNames
		fakeKubeService.currentState.AllPhases = phaseNames
		fakeKubeService.phases = []phases.PhaseInterface{}
		fakeKubeService.currentState.ClusterID = clusterID
		fakeKubeService.currentState.ClusterRole = clusterRole
		fakeKubeService.config.ClusterRole = clusterRole
		fakeKubeService.config.ClusterID = clusterID
		fakeKubeService.config.FullRetryCount = 10
		fakeKubeService.phases = make([]phases.PhaseInterface, 4)
		// This test is to validate that on the tenth attempt we do restart all the phases
		// skip the phase with index 20 so that we can fine tune failure and success of the middle phase
		for i := 0; i < 4; i++ {
			if i == 1 {
				continue
			}
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).AnyTimes()
			if i == 0 {
				// Since only middle phase fails to start, first phase should not be stopped
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(1)
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(2)
			} else {
				// Stop of last phase should be called 2 times -
				// 1. When running the clean stop before 1st start
				// 2. When stopping the chain partially till phase ordered 20 as it failed once
				phase.EXPECT().Stop(ctx, *fakeKubeService.config).Return(nil).Times(2)
				// Start of last phase should only be called once during the 10th start
				phase.EXPECT().Start(ctx, *fakeKubeService.config).Return(nil).Times(1)
			}
			fakeKubeService.phases[i] = phase
		}
		phase := mocks.NewMockPhaseInterface(mockCtrl)
		name := phaseNames[1]
		phase.EXPECT().GetHostPhase().Return(getHostPhase(name, 10)).AnyTimes()
		phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
		phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).AnyTimes()
		firstStopCall := phase.EXPECT().Stop(ctx, *fakeKubeService.config)
		firstStopCall.Return(nil).Times(1)
		firstStartCall := phase.EXPECT().Start(ctx, *fakeKubeService.config).After(firstStopCall)
		firstStartCall.Return(fakeError).Times(1)
		// Stop will be called again as part of partially stopping chain till failed phase
		secondStopCall := phase.EXPECT().Stop(ctx, *fakeKubeService.config).After(firstStartCall)
		secondStopCall.Return(nil).Times(1)
		secondStartCall := phase.EXPECT().Start(ctx, *fakeKubeService.config).After(secondStopCall)
		// This time we make the phase successfully to indicate chain started successfully the second time
		secondStartCall.Return(nil).Times(1)
		fakeKubeService.phases[1] = phase

		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.handleServiceStartState(ctx)
		fakeKubeService.currentState.StartAttempts = 9
		fakeKubeService.Status(ctx)
		// Actual service will sleep 30s after first call completes. But this is unit tests and we don't want to wait that long.
		fakeKubeService.handleServiceStartState(ctx)
	})

	/* Not testing the corner cases for numStartAttempts arg because -
	1. The value is set to 0 by default and also in code it is only set to 0. We do not perform "--"
	   operation on this arg so the value will never go -ve.
	2. On 64-bit machine max int is 9,223,372,036,854,775,807. Even if the actual status/start/stop
	   takes no time nodelet sleeps 30s between every iteration.
	   So to reach that iteration count it will take 8.7741363e+12 years.
	   The assumption is we don't wait that long to remediate the issue :)
	*/
	It("Is testing handleExpectedServiceState with expected state as True", func() {
		clusterID := "fakeID"
		clusterRole := "master"
		fakeKubeService.config.ClusterRole = clusterRole
		fakeKubeService.config.ClusterID = clusterID
		fakeKubeService.config.FullRetryCount = 10
		fakeKubeService.config.KubeServiceState = constants.ServiceStateTrue
		for i := 0; i < 3; i++ {
			// Mock the service to be running. Need to validate that handleServiceStartState is called.
			// Inner working of handleServiceStartState already unit tested above.
			// Start / Stop should not be called for any phase
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			// If service is running all phases return exit code 0 - success
			phase.EXPECT().Status(ctx, *fakeKubeService.config).Return(nil).Times(1)
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		fakeKubeService.currentState.ClusterID = clusterID
		fakeKubeService.currentState.ClusterRole = clusterRole
		fakeKubeService.currentState.StartFailStep = -1

		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.Reconcile(ctx)
	})

	It("Is testing handleExpectedServiceState with expected state as False", func() {
		// Mock the service to be stopped. Need to validate that handleServiceStopState is called.
		// Inner working of handleServiceStopState already unit tested above.
		for i := 0; i < 3; i++ {
			// Start / Stop should not be called for any phase
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		fakeKubeService.currentState.StartFailStep = 0
		viper.Set("KUBE_SERVICE_STATE", "false")

		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.Reconcile(ctx)
	})

	It("Is testing handleExpectedServiceState called with unexpected service state", func() {
		for i := 0; i < 3; i++ {
			// Start / Stop should not be called for any phase
			phase := mocks.NewMockPhaseInterface(mockCtrl)
			name := phaseNames[i]
			phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i*10)).AnyTimes()
			phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
			fakeKubeService.phases = append(fakeKubeService.phases, phase)
		}
		viper.Set("KUBE_SERVICE_STATE", "unexpected")
		var err error
		fakeKubeService.sunpike, err = sunpikeutils.InitOrGetSunpikeClient(fakeKubeService.phases, *fakeKubeService.config, sunpikev1alpha1.HostSpec{})
		Expect(err).To(BeNil())

		fakeKubeService.Reconcile(ctx)
	})

	It("Is testing CreateNodeletFromConfig", func() {
		// Store original functions to restore later
		origLoadphases := loadRolePhases
		origFileio := getFileIO
		defer func() {
			loadRolePhases = origLoadphases
			getFileIO = origFileio
		}()
		loadRolePhases = func(ctx context.Context, cfg config.Config) ([]phases.PhaseInterface, error) {
			fakePhaseList := []phases.PhaseInterface{}
			for i := 10; i <= 30; i += 10 {
				phase := mocks.NewMockPhaseInterface(mockCtrl)
				name := phaseNames[i/10-1]
				phase.EXPECT().GetHostPhase().Return(getHostPhase(name, i)).AnyTimes()
				phase.EXPECT().GetPhaseName().Return(name).AnyTimes()
				fakePhaseList = append(fakePhaseList, phase)
			}
			return fakePhaseList, nil
		}
		getFileIO = func() fileio.FileInterface {
			mockFile := mocks.NewMockFileInterface(mockCtrl)
			mockFile.EXPECT().WriteToFile(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			mockFile.EXPECT().ReadJSONFile(gomock.Any(), gomock.Any()).AnyTimes()
			return mockFile
		}
		config.DefaultConfig.ConnectTimeout = 1
		fakeService, err := CreateNodeletFromConfig(ctx, &config.DefaultConfig)
		Expect(err).To(BeNil())
		Expect(fakeService.currentState.CompletedPhases).To(BeEmpty())
		Expect(fakeService.currentState.AllPhases).To(Equal(phaseNames))
		Expect(fakeService.currentState.AllStatusChecks).To(Equal(phaseNames))
	})
})
