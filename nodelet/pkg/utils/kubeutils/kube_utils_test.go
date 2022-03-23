package kubeutils

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"
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
		ctx     context.Context
		fakeCfg *config.Config

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
		ctx.Done()
	})

	Context("Validates Node", func() {
		It("Gives node from k8s api", func() {
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			Expect(node).To(Equal(fakeNode))
		})
		It("fails to get node from k8s api if nodename is empty", func() {
			_, err := utilsImpl.GetNodeFromK8sApi(ctx, "")
			Expect(err).ToNot(BeNil())
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
		It("should add labels to node", func() {
			err := utilsImpl.AddLabelsToNode(ctx, nodeName, labelsToAdd)
			Expect(err).To(BeNil())
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			Expect(node).To(Equal(fakeNode))
		})
		It("fails to add labels to node if nodename is empty", func() {
			err := utilsImpl.AddLabelsToNode(ctx, "", labelsToAdd)
			Expect(err).NotTo(BeNil())
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
		It("should add single taint to node", func() {
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			taintedNode, updated, err := AddOrUpdateTaint(node, taint)
			Expect(err).To(BeNil())
			Expect(taintedNode).To(Equal(fakeNode))
			Expect(updated).To(Equal(true))
		})
		It("should add slice of taints to node", func() {
			err := utilsImpl.AddTaintsToNode(ctx, nodeName, taintsToAdd)
			Expect(err).To(BeNil())
			taintedNode, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			Expect(taintedNode).To(Equal(fakeNode))
		})
		It("fails to add taints to node if nodename is empty", func() {
			err := utilsImpl.AddTaintsToNode(ctx, "", taintsToAdd)
			Expect(err).NotTo(BeNil())
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
		It("should add annots to node", func() {
			fakeNode.ObjectMeta.Annotations = map[string]string{
				"UserNodeCordon": "true",
			}
			err := utilsImpl.AddAnnotationsToNode(ctx, nodeName, annotsToAdd)
			Expect(err).To(BeNil())
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			Expect(node).To(Equal(fakeNode))
		})
		It("fails to add annots to node if nodename is empty", func() {
			err := utilsImpl.AddAnnotationsToNode(ctx, "", annotsToAdd)
			Expect(err).NotTo(BeNil())
		})
		It("should remove annots from node", func() {
			fakeNode.ObjectMeta.Annotations = map[string]string{}
			err := utilsImpl.AddAnnotationsToNode(ctx, nodeName, annotsToAdd)
			Expect(err).To(BeNil())
			err = utilsImpl.RemoveAnnotationsFromNode(ctx, nodeName, annotsToRemove)
			Expect(err).To(BeNil())
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			Expect(node).To(Equal(fakeNode))
		})
		It("fails to remove annots from node if nodename is empty", func() {
			err := utilsImpl.RemoveAnnotationsFromNode(ctx, "", annotsToRemove)
			Expect(err).NotTo(BeNil())
		})
	})
	Context("Validates if Drain Nodes", func() {
		It("Drains node from k8s api", func() {
			err := utilsImpl.DrainNodeFromApiServer(ctx, nodeName)
			Expect(err).To(BeNil())
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			Expect(node.Spec.Unschedulable).To(Equal(true))
		})
		It("fails to drain node if nodename is empty", func() {
			err := utilsImpl.DrainNodeFromApiServer(ctx, "")
			Expect(err).ToNot(BeNil())
		})
	})
	Context("Validates if Uncordon Node", func() {
		It("uncordons node from k8s api", func() {
			err := utilsImpl.UncordonNode(ctx, nodeName)
			Expect(err).To(BeNil())
			node, err := utilsImpl.GetNodeFromK8sApi(ctx, nodeName)
			Expect(err).To(BeNil())
			Expect(node.Spec.Unschedulable).To(Equal(false))
		})
		It("fails to uncordon node if nodename is empty", func() {
			err := utilsImpl.DrainNodeFromApiServer(ctx, "")
			Expect(err).ToNot(BeNil())
		})
	})
	Context("Validates IP ", func() {
		It("if ipv4 it returns as it is", func() {
			ip, err := utilsImpl.IpForHttp("10.12.13.14")
			Expect(err).To(BeNil())
			Expect(ip).To(Equal("10.12.13.14"))
		})
		It("if ipv6 it adds bracket", func() {
			ip, err := utilsImpl.IpForHttp("2001:db8::1234:5678")
			Expect(err).To(BeNil())
			Expect(ip).To(Equal("[2001:db8::1234:5678]"))
		})
		It("fails if invalid ip ", func() {
			err := errors.New("IP is invalid")
			_, reterr := utilsImpl.IpForHttp("10.12.1314")
			Expect(reterr).ToNot(BeNil())
			Expect(reterr).To(Equal(err))
		})
	})

})
