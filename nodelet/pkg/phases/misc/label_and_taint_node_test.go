package misc

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/stretchr/testify/assert"

	"github.com/onsi/ginkgo/reporters"
	v1 "k8s.io/api/core/v1"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Apply and validate node taints phase", func() {

	var (
		mockCtrl      *gomock.Controller
		fakePhase     *LabelTaintNodePhasev2
		ctx           context.Context
		fakeCfg       *config.Config
		fakeKubeUtils *mocks.MockUtils
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewLabelTaintNodePhaseV2()
		ctx = context.TODO()

		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
		fakeKubeUtils = mocks.NewMockUtils(mockCtrl)
		fakePhase.kubeUtils = fakeKubeUtils
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("validates status command", func() {
		It("to succeed", func() {
			ret := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("validates stop command", func() {
		It("to succeed", func() {
			ret := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("validates start command", func() {
		It("fails when can't get nodeIdentifier or it's null", func() {
			err := errors.New("fake error")

			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("", err).Times(1)

			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("fails when nodeIdentifier is 127.0.0.1", func() {
			err := errors.New("node interface might have lost IP address. Failing")

			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("127.0.0.1", nil).Times(1)

			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("fails when labels are not added", func() {
			err := errors.New("fake error")
			labels := map[string]string{
				"node-role.kubernetes.io/master": "",
			}
			fakeCfg.ClusterRole = "master"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("10.128.240.67", nil).Times(1)
			fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, "10.128.240.67", labels).Return(err).Times(1)

			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})

		Context("when role is master", func() {
			var (
				labels map[string]string
				taints []*v1.Taint
			)
			BeforeEach(func() {
				fakeCfg.ClusterRole = "master"
				labels = map[string]string{
					"node-role.kubernetes.io/master": "",
				}
				taints = []*v1.Taint{
					{
						Key:    "node-role.kubernetes.io/master",
						Value:  "true",
						Effect: "NoSchedule",
					},
				}
			})
			It("should not add taints when it is schedulable", func() {
				fakeCfg.MasterSchedulable = true

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("10.128.240.67", nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, "10.128.240.67", labels).Return(nil).Times(1)

				err := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), err)
			})
			It("should add taints when it is not schedulable", func() {
				fakeCfg.MasterSchedulable = false

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("10.128.240.67", nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, "10.128.240.67", labels).Return(nil).Times(1)
				fakeKubeUtils.EXPECT().AddTaintsToNode(ctx, "10.128.240.67", taints).Return(nil).Times(1)
				err := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), err)
			})
			It("fails when cant add taint", func() {
				fakeCfg.MasterSchedulable = false

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("10.128.240.67", nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, "10.128.240.67", labels).Return(nil).Times(1)
				err := errors.New("fake error")
				fakeKubeUtils.EXPECT().AddTaintsToNode(ctx, "10.128.240.67", taints).Return(err).Times(1)
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), reterr, err)
			})

		})
		Context("when role is worker", func() {
			var labels map[string]string
			BeforeEach(func() {
				labels = map[string]string{
					"node-role.kubernetes.io/worker": "",
				}
				fakeCfg.ClusterRole = "worker"
			})
			It("should not add taints", func() {
				fakeCfg.MasterSchedulable = true

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("10.128.240.67", nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, "10.128.240.67", labels).Return(nil).Times(1)

				err := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), err)
			})

		})

	})

})
