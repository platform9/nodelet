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
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
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
		mockCtrl       *gomock.Controller
		fakePhase      *LabelTaintNodePhase
		ctx            context.Context
		fakeCfg        *config.Config
		fakeKubeUtils  *mocks.MockUtils
		nodeIdentifier string
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewLabelTaintNodePhase()
		ctx = context.TODO()
		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
		fakeKubeUtils = mocks.NewMockUtils(mockCtrl)
		fakePhase.kubeUtils = fakeKubeUtils
		nodeIdentifier = "10.128.240.67"
	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("Validates status command", func() {
		It("To succeed", func() {
			ret := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("Validates stop command", func() {
		It("To succeed", func() {
			ret := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("Validates start command", func() {
		It("Fails when can't get nodeIdentifier or it's null", func() {
			err := errors.New("fake error")

			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("", err).Times(1)

			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when nodeIdentifier is 127.0.0.1", func() {
			err := errors.New("node interface might have lost IP address. Failing")

			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return("127.0.0.1", nil).Times(1)

			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when labels are not added", func() {
			err := errors.New("fake error")
			labels := map[string]string{
				"node-role.kubernetes.io/master": "",
			}
			fakeCfg.ClusterRole = constants.RoleMaster
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, nodeIdentifier, labels).Return(err).Times(1)

			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})

		Context("When role is master", func() {
			var (
				labels map[string]string
				taints []*v1.Taint
			)
			BeforeEach(func() {
				fakeCfg.ClusterRole = constants.RoleMaster
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
			It("Should not add taints when it is schedulable", func() {
				fakeCfg.MasterSchedulable = true

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, nodeIdentifier, labels).Return(nil).Times(1)

				err := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), err)
			})
			It("Should add taints when it is not schedulable", func() {
				fakeCfg.MasterSchedulable = false

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, nodeIdentifier, labels).Return(nil).Times(1)
				fakeKubeUtils.EXPECT().AddTaintsToNode(ctx, nodeIdentifier, taints).Return(nil).Times(1)
				err := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), err)
			})
			It("Fails when can't add taint", func() {
				fakeCfg.MasterSchedulable = false

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, nodeIdentifier, labels).Return(nil).Times(1)
				err := errors.New("fake error")
				fakeKubeUtils.EXPECT().AddTaintsToNode(ctx, nodeIdentifier, taints).Return(err).Times(1)
				reterr := fakePhase.Start(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), reterr, err)
			})

		})
		Context("When role is worker", func() {
			var labels map[string]string
			BeforeEach(func() {
				labels = map[string]string{
					"node-role.kubernetes.io/worker": "",
				}
				fakeCfg.ClusterRole = constants.RoleWorker
			})
			It("Should not add taints", func() {
				fakeCfg.MasterSchedulable = true

				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(nodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().AddLabelsToNode(ctx, nodeIdentifier, labels).Return(nil).Times(1)

				err := fakePhase.Start(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), err)
			})

		})

	})

})
