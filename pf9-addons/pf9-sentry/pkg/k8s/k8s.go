package k8s

import (
	"log"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client ...
type Client struct {
	client kubernetes.Interface
}

//
// Newk8sClient returns a client object to query k8s API server
//
func Newk8sClient(inCluster bool) (*Client, error) {
	var (
		config    *rest.Config
		err       error
		k8sClient *Client
	)

	switch inCluster {
	case true:
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Println(err.Error())
			return k8sClient, err
		}
	case false:
		configFile := os.Getenv("KUBECONFIG")
		config, err = clientcmd.BuildConfigFromFlags("", configFile)
		if err != nil {
			log.Println(err.Error())
			return k8sClient, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println(err.Error())
		return k8sClient, err
	}

	k8sClient = &Client{clientset}
	return k8sClient, nil
}

// Ping checks connectivity with k8s API server
func (c *Client) Ping() bool {
	_, err := c.client.CoreV1().Services("default").List(metav1.ListOptions{})
	if errors.IsNotFound(err) {
		log.Printf("No services found in default namespace: %s\n", err.Error())
		return true
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting services in default namespace: %v\n", statusError.ErrStatus.Message)
		return true
	} else if err != nil {
		log.Printf("Error: %s\n", err.Error())
		return false
	} else {
		return true
	}
}
