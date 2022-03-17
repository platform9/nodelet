package bashscriptbasedphases_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"

	"github.com/platform9/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/phases"
	bashscript "github.com/platform9/nodelet/nodelet/pkg/phases/bash_script_based_phases"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test phases.go", func() {
	var (
		mockCtrl        *gomock.Controller
		origCmd         = bashscript.LocalCmd
		origBaseCommand []string
		ctx             context.Context
		fakeCfg         *config.Config
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		origCmd = bashscript.LocalCmd
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
		bashscript.LocalCmd = origCmd
		constants.BaseCommand = origBaseCommand
		ctx.Done()
	})

	It("Should be able to stop the phase on first attempt", func() {
		// Verifies that stop is called only once
		mockCmd := mocks.NewMockCLI(mockCtrl)
		bashscript.LocalCmd = mockCmd
		firstStop := mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"stop")
		firstStop.Return(0, []string{}, nil).Times(1)
		fakeCfg.PhaseRetry = 3
		phase := fakePhase(fakeCfg.PhaseRetry)
		err := phase.Stop(ctx, *fakeCfg)
		Expect(err).To(BeNil())
	})

	It("Should be able to stop the phase on second attempt", func() {
		// Verifies that 3rd call to stop is not invoked
		mockCmd := mocks.NewMockCLI(mockCtrl)
		bashscript.LocalCmd = mockCmd
		firstStop := mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"stop")
		firstStop.Return(1, []string{}, nil).Times(1)
		mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"stop").After(firstStop).Return(0, []string{}, nil).Times(1)
		fakeCfg.PhaseRetry = 3
		phase := fakePhase(fakeCfg.PhaseRetry)
		err := phase.Stop(ctx, *fakeCfg)
		Expect(err).To(BeNil())
	})

	It("Should NOT try to invoke stop more than PHASE_RETRY times", func() {
		fakeCfg.PhaseRetry = 3
		mockCmd := mocks.NewMockCLI(mockCtrl)
		bashscript.LocalCmd = mockCmd
		firstStop := mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"stop")
		firstStop.Return(1, []string{}, nil).MaxTimes(fakeCfg.PhaseRetry)
		phase := fakePhase(fakeCfg.PhaseRetry)
		err := phase.Stop(ctx, *fakeCfg)
		Expect(err).ToNot(BeNil())
	})

	It("Should start the phase", func() {
		fakeCfg.PhaseRetry = 3
		mockCmd := mocks.NewMockCLI(mockCtrl)
		bashscript.LocalCmd = mockCmd
		mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"start").Return(0, []string{}, nil).Times(1)
		phase := fakePhase(fakeCfg.PhaseRetry)
		err := phase.Start(ctx, *fakeCfg)
		Expect(err).To(BeNil())
	})

	It("Should fail to start the phase", func() {
		fakeCfg.PhaseRetry = 3
		errorCode := 127
		mockCmd := mocks.NewMockCLI(mockCtrl)
		bashscript.LocalCmd = mockCmd
		mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"start").Return(errorCode, []string{}, nil).Times(1)
		phase := fakePhase(fakeCfg.PhaseRetry)
		err := phase.Start(ctx, *fakeCfg)
		Expect(err).ToNot(BeNil())
	})

	It("Should check status of a phase", func() {
		fakeCfg.PhaseRetry = 3
		mockCmd := mocks.NewMockCLI(mockCtrl)
		bashscript.LocalCmd = mockCmd
		mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"status").Return(0, []string{}, nil).Times(1)
		phase := fakePhase(fakeCfg.PhaseRetry)
		err := phase.Status(ctx, *fakeCfg)
		Expect(err).To(BeNil())
	})

	It("Should NOT invoke status more than PHASE_RETRY times", func() {
		fakeCfg.PhaseRetry = 3
		mockCmd := mocks.NewMockCLI(mockCtrl)
		bashscript.LocalCmd = mockCmd
		mockCmd.EXPECT().RunCommandWithStdOut(
			ctx, nil, -1, "", constants.BaseCommand[0],
			constants.BaseCommand[1],
			gomock.Any(),
			"status").Return(1, []string{}, nil).MaxTimes(fakeCfg.PhaseRetry)
		phase := fakePhase(fakeCfg.PhaseRetry)
		err := phase.Status(ctx, *fakeCfg)
		Expect(err).ToNot(BeNil())
	})
	// phase.RunCommand is implicitly tested multiple times in all the functions above
})

func fakePhase(retry int) phases.PhaseInterface {
	return &bashscript.Phase{
		Filename: "fake1.sh",
		HostPhase: &v1alpha1.HostPhase{
			Name:  "Fake 1",
			Order: 10,
		},
		Retry: retry,
	}
}
