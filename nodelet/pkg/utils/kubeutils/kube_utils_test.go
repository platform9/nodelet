package kubeutils

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Phases Suite", []Reporter{junitReporter})
}

var _ = Describe("Test Kube Utils", func() {

	var (
		ctx       context.Context
		fakeCfg   *config.Config
		utilsImpl *UtilsImpl
		nodeName  string
		fakeNode  *v1.Node
	)
	BeforeEach(func() {
		var err error
		utilsImpl = &UtilsImpl{
			Clientset: fake.NewSimpleClientset(),
		}
		ctx = context.TODO()
		nodeName = "10.128.243.126"
		// Setup config
		fakeCfg, err = config.GetDefaultConfig()
		assert.Nil(GinkgoT(), err)
		fakeCfg.UseCgroups = false
		fakeNode = &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
		}
		_, _ = utilsImpl.Clientset.CoreV1().Nodes().Create(context.TODO(), fakeNode, metav1.CreateOptions{})
	})

	AfterEach(func() {
		_ = utilsImpl.Clientset.CoreV1().Nodes().Delete(context.TODO(), nodeName, metav1.DeleteOptions{})
		ctx.Done()
	})

	Context("Validates Node", func() {
		It("Gets node from k8s API", func() {
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), node, fakeNode)
		})
		It("Fails to get node from k8s API if invalid (non-exist) nodename", func() {
			nodeName = "8.8.8.8"
			_, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.NotNil(GinkgoT(), err)
		})
	})
	Context("Validates labels", func() {
		var labelsToAdd map[string]string

		BeforeEach(func() {
			labelsToAdd = map[string]string{
				"node-role.kubernetes.io/master": "",
			}
			fakeNode.ObjectMeta.Labels = map[string]string{
				"node-role.kubernetes.io/master": "",
			}
		})
		It("Should add labels to node", func() {

			err := utilsImpl.AddLabelsToNode(ctx, nodeName, labelsToAdd)
			assert.Nil(GinkgoT(), err)
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), node, fakeNode)
		})
		It("Fails to add labels to node invalid (non-exist) nodename", func() {
			nodeName = "8.8.8.8"
			err := utilsImpl.AddLabelsToNode(ctx, nodeName, labelsToAdd)
			assert.NotNil(GinkgoT(), err)
		})
	})
	Context("Validates taints", func() {
		var taintsToAdd []*v1.Taint
		var taint *v1.Taint
		BeforeEach(func() {
			taint = &v1.Taint{
				Key:    "node-role.kubernetes.io/master",
				Value:  "true",
				Effect: "NoSchedule",
			}
			taintsToAdd = []*v1.Taint{
				{
					Key:    "node-role.kubernetes.io/master",
					Value:  "true",
					Effect: "NoSchedule",
				},
			}
			fakeNode.Spec.Taints = []v1.Taint{
				{
					Key:    "node-role.kubernetes.io/master",
					Value:  "true",
					Effect: "NoSchedule",
				},
			}
		})
		It("Should add single taint to node", func() {
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			taintedNode, updated, err := AddOrUpdateTaint(node, taint)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), taintedNode, fakeNode)
			assert.Equal(GinkgoT(), updated, true)
		})
		It("Should add slice of taints to node", func() {
			err := utilsImpl.AddTaintsToNode(ctx, nodeName, taintsToAdd)
			assert.Nil(GinkgoT(), err)
			taintedNode, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), taintedNode, fakeNode)
		})
		It("Fails to add taints to node invalid (non-exist) nodename", func() {
			nodeName = "8.8.8.8"
			err := utilsImpl.AddTaintsToNode(ctx, nodeName, taintsToAdd)
			assert.NotNil(GinkgoT(), err)
		})
	})
	Context("Validates Annotations", func() {
		var annotsToAdd map[string]string
		var annotsToRemove []string
		BeforeEach(func() {
			annotsToAdd = map[string]string{
				"UserNodeCordon": "true",
			}
			annotsToRemove = []string{"UserNodeCordon"}
		})
		It("Should add annots to node", func() {
			fakeNode.ObjectMeta.Annotations = map[string]string{
				"UserNodeCordon": "true",
			}
			err := utilsImpl.AddAnnotationsToNode(ctx, nodeName, annotsToAdd)
			assert.Nil(GinkgoT(), err)
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), node, fakeNode)
		})
		It("Fails to add annots to node invalid (non-exist) nodename", func() {
			nodeName = "8.8.8.8"
			err := utilsImpl.AddAnnotationsToNode(ctx, nodeName, annotsToAdd)
			assert.NotNil(GinkgoT(), err)
		})
		It("Should remove annots from node", func() {
			fakeNode.ObjectMeta.Annotations = map[string]string{}
			err := utilsImpl.AddAnnotationsToNode(ctx, nodeName, annotsToAdd)
			assert.Nil(GinkgoT(), err)
			err = utilsImpl.RemoveAnnotationsFromNode(ctx, nodeName, annotsToRemove)
			assert.Nil(GinkgoT(), err)
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), node, fakeNode)
		})
		It("Fails to remove annots from node invalid (non-exist) nodename", func() {
			nodeName = "8.8.8.8"
			err := utilsImpl.RemoveAnnotationsFromNode(ctx, nodeName, annotsToRemove)
			assert.NotNil(GinkgoT(), err)
		})
	})
	Context("Validates if Drain Nodes", func() {
		It("Drains node from k8s api", func() {
			err := utilsImpl.DrainNodeFromApiServer(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), node.Spec.Unschedulable, true)
		})
		It("Fails to drain node invalid (non-exist) nodename", func() {
			nodeName = "8.8.8.8"
			err := utilsImpl.DrainNodeFromApiServer(ctx, nodeName)
			assert.NotNil(GinkgoT(), err)
		})
	})
	Context("Validates if Uncordon Node", func() {
		It("Uncordons node from k8s api", func() {
			err := utilsImpl.UncordonNode(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), node.Spec.Unschedulable, false)
		})
		It("Fails to uncordon node invalid (non-exist) nodename", func() {
			nodeName = "8.8.8.8"
			err := utilsImpl.DrainNodeFromApiServer(ctx, nodeName)
			assert.NotNil(GinkgoT(), err)
		})
	})
	Context("Validates IP ", func() {
		It("If ipv4 it returns as it is", func() {
			ip, err := utilsImpl.IpForHttp("10.12.13.14")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), ip, "10.12.13.14")
		})
		It("If ipv6 it adds bracket", func() {
			ip, err := utilsImpl.IpForHttp("2001:db8::1234:5678")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), ip, "[2001:db8::1234:5678]")
		})
		It("Fails if invalid ip ", func() {
			err := errors.New("invalid IP")
			_, reterr := utilsImpl.IpForHttp("10.12.1314")
			assert.NotNil(GinkgoT(), err)
			assert.Equal(GinkgoT(), reterr.Error(), err.Error())
		})
	})

})
