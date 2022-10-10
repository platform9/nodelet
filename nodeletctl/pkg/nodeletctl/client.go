package nodeletctl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterClient struct {
	client     *kubernetes.Clientset
	timeout    time.Duration
	clusterCfg *BootstrapConfig
}

func GetClient(clusterCfg *BootstrapConfig) (*ClusterClient, error) {
	var kubeconfig string
	if clusterCfg.KubeConfig == "" {
		kubeconfigPath := filepath.Join(ClusterStateDir, clusterCfg.ClusterId, "certs", AdminKubeconfig)
		kubeconfig = kubeconfigPath
		if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("kubeconfig not specified and not found in default path %s", kubeconfigPath)
		}
	} else {
		kubeconfig = clusterCfg.KubeConfig
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to get client for kubeconfig %s: err: %s", kubeconfig, err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to get client for kubeconfig %s: err: %s", kubeconfig, err)
	}
	return newClusterClient(clientset, clusterCfg), nil
}

func newClusterClient(k8sClient *kubernetes.Clientset, clusterCfg *BootstrapConfig) *ClusterClient {
	return &ClusterClient{
		client:     k8sClient,
		timeout:    ClusterTimeout,
		clusterCfg: clusterCfg,
	}
}

func (c *ClusterClient) GetMatchingNodes(labels ...string) ([]string, error) {
	nodes, err := c.client.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to list nodes: %s", err)
	}

	var matchingNodes []string
	for _, node := range nodes.Items {
		match := true
		for _, label := range labels {
			if _, ok := node.Labels[label]; !ok {
				match = false
				break
			}
		}
		if !match {
			zap.S().Debugf("Skipping node %s", node.Name)
			continue
		}
		zap.S().Infof("Matched node %s", node.Name)
		matchingNodes = append(matchingNodes, node.Name)
	}
	return matchingNodes, nil
}
