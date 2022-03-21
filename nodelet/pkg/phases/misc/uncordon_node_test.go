package misc

import (
	"context"
	"errors"

	//"os"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCommandOfUncordonNode(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Uncordon node phase", func() {

	var (
		mockCtrl           *gomock.Controller
		fakePhase          *UncordonNodePhasev2
		ctx                context.Context
		fakeCfg            *config.Config
		fakeKubeUtils      *mocks.MockUtils
		fakeNode           *v1.Node
		fakeNodeIdentifier string
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewUncordonNodePhaseV2()
		ctx = context.TODO()

		// Setup config
		var err error
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false

		fakeKubeUtils = mocks.NewMockUtils(mockCtrl)
		fakePhase.kubeUtils = fakeKubeUtils
		fakeNodeIdentifier = "10.128.242.67"
		fakeNode = &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"KubeStackShutDown": "false",
				},
			},
		}

	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("validates stop command", func() {
		It("to succeed", func() {
			ret := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("validates status command", func() {
		BeforeEach(func() {
			constants.KubeStackStartFileMarker = "testdata/absent.txt"
			fakeNode.ObjectMeta.Annotations["KubeStackShutDown"] = "false"
		})

		It("fails when can't get nodeIdentifier or its null", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, err).Times(1)
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("when Kube stack is still booting up it does nothing and returns nil", func() {
			constants.KubeStackStartFileMarker = "testdata/dummy.txt"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			err := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)

		})
		It("fails when it can't get node from k8s api", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, err).Times(1)
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("if KubeStackShutDown annotation is present it does nothing and returns nil as node was cordoned by PF9", func() {
			fakeNode.ObjectMeta.Annotations["KubeStackShutDown"] = "true"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
			err := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
		})

		Context("when node is unschedulable", func() {
			var annotsToAdd map[string]string
			BeforeEach(func() {
				fakeNode.Spec.Unschedulable = true
				annotsToAdd = map[string]string{
					"UserNodeCordon": "true",
				}
			})
			It("it adds userNodeCordon annotation", func() {
				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
				fakeKubeUtils.EXPECT().AddAnnotationsToNode(ctx, fakeNodeIdentifier, annotsToAdd).Return(nil).Times(1)
				ret := fakePhase.Status(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), ret)
			})
			It("it fails to add userNodeCordon annotation when add annotation fails", func() {
				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
				err := errors.New("fake error")
				fakeKubeUtils.EXPECT().AddAnnotationsToNode(ctx, fakeNodeIdentifier, annotsToAdd).Return(err).Times(1)
				reterr := fakePhase.Status(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), reterr, err)
			})
		})
		Context("when node is schedulable", func() {
			var annotsToRemove []string
			BeforeEach(func() {
				fakeNode.Spec.Unschedulable = false
				annotsToRemove = []string{"UserNodeCordon"}
			})
			It("it removes userNodeCordon annotation", func() {
				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
				fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(nil).Times(1)
				ret := fakePhase.Status(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), ret)
			})
			It("it fails to remove userNodeCordon annotation when add annotation fails", func() {
				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
				err := errors.New("fake error")
				fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(err).Times(1)
				reterr := fakePhase.Status(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), reterr, err)
			})
		})

	})
	Context("validates start command", func() {
		It("fails when can't get nodeIdentifier or its null", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, err).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("fails when nodeIdentifier is 127.0.0.1", func() {
			err := errors.New("node interface might have lost IP address. Failing")
			fakeNodeIdentifier = "127.0.0.1"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("fails when can't get node from k8s api", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, err).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
	})
})
