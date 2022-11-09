package kubelet

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/stretchr/testify/assert"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Kubelet Configure Start Phase", func() {

	var (
		ctx              context.Context
		mockCtrl         *gomock.Controller
		fakePhase        *KubeletConfigureStartPhase
		fakeCfg          *config.Config
		fakeKubeletUtils *mocks.MockKubeletUtilsInterface
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewKubeletConfigureStartPhase()
		fakeKubeletUtils = mocks.NewMockKubeletUtilsInterface(mockCtrl)
		fakePhase.kubeletUtils = fakeKubeletUtils
		ctx = context.TODO()
		// Setup Config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("validate status command", func() {
		It("should fail if pf9-kubelet not running", func() {
			err := fmt.Errorf("pf9-kubelet is not active")
			fakeKubeletUtils.EXPECT().IsKubeletRunning().Return(false).Times(1)
			retErr := fakePhase.Status(ctx, *fakeCfg)
			Expect(retErr).ToNot(BeNil())
			Expect(retErr).To(Equal(err))
		})

		It("should succeed if pf9-kubelet is running", func() {
			fakeKubeletUtils.EXPECT().IsKubeletRunning().Return(true).Times(1)
			ret := fakePhase.Status(ctx, *fakeCfg)
			Expect(ret).To(BeNil())
		})

	})

	Context("validate start command", func() {
		It("should fail if it is not able to ensure kubelet is running", func() {
			err := fmt.Errorf("fake error")
			fakeKubeletUtils.EXPECT().EnsureKubeletRunning(*fakeCfg).Return(err).Times(1)
			retErr := fakePhase.Start(ctx, *fakeCfg)
			Expect(retErr).ToNot(BeNil())
			Expect(retErr).To(Equal(err))
		})
		It("should succeed if it can ensure kubelet is running", func() {
			fakeKubeletUtils.EXPECT().EnsureKubeletRunning(*fakeCfg).Return(nil).Times(1)
			retErr := fakePhase.Start(ctx, *fakeCfg)
			Expect(retErr).To(BeNil())
		})
	})

	Context("validate stop command", func() {
		It("should fail if it is not able to ensure kubelet is stopped", func() {
			err := fmt.Errorf("fake error")
			fakeKubeletUtils.EXPECT().EnsureKubeletStopped().Return(err).Times(1)
			retErr := fakePhase.Stop(ctx, *fakeCfg)
			Expect(retErr).ToNot(BeNil())
			Expect(retErr).To(Equal(err))
		})
		It("should succeed if it can ensure kubelet is stopped", func() {
			fakeKubeletUtils.EXPECT().EnsureKubeletStopped().Return(nil).Times(1)
			retErr := fakePhase.Stop(ctx, *fakeCfg)
			Expect(retErr).To(BeNil())
		})
	})

})
