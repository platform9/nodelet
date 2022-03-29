package misc

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/mocks"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Test Uncordon node phase", func() {

	var (
		mockCtrl           *gomock.Controller
		fakePhase          *UncordonNodePhase
		ctx                context.Context
		fakeCfg            *config.Config
		fakeKubeUtils      *mocks.MockUtils
		fakeNode           *v1.Node
		fakeNodeIdentifier string
	)
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		fakePhase = NewUncordonNodePhase()
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
					"UserNodeCordon":    "false",
				},
			},
		}

	})

	AfterEach(func() {
		mockCtrl.Finish()
		ctx.Done()
	})

	Context("Validates stop command", func() {
		It("To succeed", func() {
			ret := fakePhase.Stop(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})

	Context("Validates status command", func() {
		BeforeEach(func() {
			constants.KubeStackStartFileMarker = "testdata/absent.txt"
			fakeNode.ObjectMeta.Annotations["KubeStackShutDown"] = "false"
		})

		It("Fails when can't get nodeIdentifier or its null", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, err).Times(1)
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("When Kube stack is still booting up it does nothing and returns nil", func() {
			constants.KubeStackStartFileMarker = "testdata/dummy.txt"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			err := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)

		})
		It("Fails when it can't get node from k8s api", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, err).Times(1)
			reterr := fakePhase.Status(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("If KubeStackShutDown annotation is present it does nothing and returns nil as node was cordoned by PF9", func() {
			fakeNode.ObjectMeta.Annotations["KubeStackShutDown"] = "true"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
			err := fakePhase.Status(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), err)
		})

		Context("When node is unschedulable", func() {
			var annotsToAdd map[string]string
			BeforeEach(func() {
				fakeNode.Spec.Unschedulable = true
				annotsToAdd = map[string]string{
					"UserNodeCordon": "true",
				}
			})
			It("It adds userNodeCordon annotation", func() {
				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
				fakeKubeUtils.EXPECT().AddAnnotationsToNode(ctx, fakeNodeIdentifier, annotsToAdd).Return(nil).Times(1)
				ret := fakePhase.Status(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), ret)
			})
			It("It fails to add userNodeCordon annotation when add annotation fails", func() {
				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
				err := errors.New("fake error")
				fakeKubeUtils.EXPECT().AddAnnotationsToNode(ctx, fakeNodeIdentifier, annotsToAdd).Return(err).Times(1)
				reterr := fakePhase.Status(ctx, *fakeCfg)
				assert.NotNil(GinkgoT(), reterr)
				assert.Equal(GinkgoT(), reterr, err)
			})
		})
		Context("When node is schedulable", func() {
			var annotsToRemove []string
			BeforeEach(func() {
				fakeNode.Spec.Unschedulable = false
				annotsToRemove = []string{"UserNodeCordon"}
			})
			It("It removes userNodeCordon annotation", func() {
				fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
				fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
				fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(nil).Times(1)
				ret := fakePhase.Status(ctx, *fakeCfg)
				assert.Nil(GinkgoT(), ret)
			})
			It("It fails to remove userNodeCordon annotation when remove annotation fails", func() {
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
	Context("Validates start command", func() {
		var annotsToRemove []string
		BeforeEach(func() {
			annotsToRemove = []string{"KubeStackShutDown"}
		})
		It("Fails when can't get nodeIdentifier or its null", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, err).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when nodeIdentifier is 127.0.0.1", func() {
			err := fmt.Errorf("node interface might have lost IP address. Failing")
			fakeNodeIdentifier = "127.0.0.1"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when can't remove KubeStackShutDown annotation (if present) as this is kube stack startup", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(err).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when can't get node from k8s api", func() {

			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, err).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("If node cordoned (By User) DO NOT uncordon, exit", func() {
			fakeNode.ObjectMeta.Annotations["UserNodeCordon"] = "true"
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
		It("Fails when can't uncordon node", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
			fakeKubeUtils.EXPECT().UncordonNode(ctx, fakeNodeIdentifier).Return(err).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("Fails when can't prevent auto reattach (i.e. can't delete the qbert metadata file)", func() {
			err := errors.New("fake error")
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
			fakeKubeUtils.EXPECT().UncordonNode(ctx, fakeNodeIdentifier).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().PreventAutoReattach().Return(err).Times(1)
			reterr := fakePhase.Start(ctx, *fakeCfg)
			assert.NotNil(GinkgoT(), reterr)
			assert.Equal(GinkgoT(), reterr, err)
		})
		It("To succeed)", func() {
			fakeKubeUtils.EXPECT().GetNodeIdentifier(*fakeCfg).Return(fakeNodeIdentifier, nil).Times(1)
			fakeKubeUtils.EXPECT().RemoveAnnotationsFromNode(ctx, fakeNodeIdentifier, annotsToRemove).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().GetNodeFromK8sApi(ctx, fakeNodeIdentifier).Return(fakeNode, nil).Times(1)
			fakeKubeUtils.EXPECT().UncordonNode(ctx, fakeNodeIdentifier).Return(nil).Times(1)
			fakeKubeUtils.EXPECT().PreventAutoReattach().Return(nil).Times(1)
			ret := fakePhase.Start(ctx, *fakeCfg)
			assert.Nil(GinkgoT(), ret)
		})
	})
})
