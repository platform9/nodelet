package kubeutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/platform9/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/pkg/utils/constants"

	"golang.org/x/net/nettest"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/drain"
)

type Utils interface {
	AddLabelsToNode(string, map[string]string) error
	AddAnnotationsToNode(string, map[string]string) error
	RemoveAnnotationsFromNode(string, []string) error
	AddTaintsToNode(string, []*corev1.Taint) error
	DrainNodeFromApiServer(string) error
	GetNodeFromK8sApi(string) (*corev1.Node, error)
	UncordonNode(string) error
}
type UtilsImpl struct {
	Clientset *kubernetes.Clientset
}

func (c *UtilsImpl) AddLabelsToNode(nodeName string, labelsToAdd map[string]string) error {
	//implement waituntil
	node, err := c.GetNodeFromK8sApi(nodeName)
	if err != nil {
		return err
	}
	metadata := node.ObjectMeta
	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}
	for k, v := range labelsToAdd {
		metadata.Labels[k] = v
	}
	return nil
}

func (c *UtilsImpl) AddAnnotationsToNode(nodeName string, annotsToAdd map[string]string) error {
	node, err := c.GetNodeFromK8sApi(nodeName)
	if err != nil {
		return err
	}
	metadata := node.ObjectMeta
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	for k, v := range annotsToAdd {
		metadata.Annotations[k] = v
	}
	_, err = c.Clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *UtilsImpl) RemoveAnnotationsFromNode(nodeName string, annotsToRemove []string) error {
	node, err := c.GetNodeFromK8sApi(nodeName)
	if err != nil {
		return err
	}
	metadata := node.ObjectMeta
	if metadata.Annotations == nil {
		//metadata.Labels = make(map[string]string)
		return nil
	}
	for _, v := range annotsToRemove {
		delete(metadata.Annotations, v)
	}
	_, err = c.Clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
func (c *UtilsImpl) AddTaintsToNode(nodename string, taintsToadd []*corev1.Taint) error {
	node, _ := c.GetNodeFromK8sApi(nodename)

	for _, taint := range taintsToadd {
		_, updated, err := AddOrUpdateTaint(node, taint)
		if err != nil {
			return err
		}
		if !updated {
			return fmt.Errorf("taint not added")
		}
	}
	_, err := c.Clientset.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (c *UtilsImpl) DrainNodeFromApiServer(nodeName string) error {

	helper := drain.Helper{
		Ctx:                 context.TODO(),
		Client:              c.Clientset,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             time.Duration(300) * time.Second,
		DeleteEmptyDirData:  true,
		Out:                 os.Stdout,
		ErrOut:              os.Stdout,
		DisableEviction:     true,
	}
	node, err := c.GetNodeFromK8sApi(nodeName)
	if err != nil {
		return err
	}
	err = drain.RunCordonOrUncordon(&helper, node, true)
	if err != nil {
		return fmt.Errorf("failed to cordon node")
	}
	err = drain.RunNodeDrain(&helper, node.Name)
	if err != nil {
		return fmt.Errorf("failed to drain node")
	}
	// TODO :
	// Add KubeStackShutDown annotation to the node on successful node drain
	// add_annotation_to_node ${node_ip} KubeStackShutDown
	return nil
}

func (c *UtilsImpl) GetNodeFromK8sApi(nodeName string) (*corev1.Node, error) {
	var node *corev1.Node
	node, err := c.Clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		return node, err
	}
	return node, nil
}

func (c *UtilsImpl) UncordonNode(nodename string) error {
	//implement wait_until
	helper := drain.Helper{
		Ctx:                 context.TODO(),
		Client:              c.Clientset,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             time.Duration(300) * time.Second,
		DeleteEmptyDirData:  true,
		Out:                 os.Stdout,
		ErrOut:              os.Stdout,
		DisableEviction:     true,
	}

	node, err := c.GetNodeFromK8sApi(nodename)
	if err != nil {
		return err
	}
	err = drain.RunCordonOrUncordon(&helper, node, false)
	if err != nil {
		return err
	}

	if !node.Spec.Unschedulable {
		return fmt.Errorf("warning: Node %v is still cordoned or cannot be fetched", nodename)
	}
	return nil
}

func PreventAutoReattach() error {

	// Unconditionally delete the qbert metadata file to prevent re-auth
	err := os.Remove("/opt/pf9/hostagent/extensions/fetch_qbert_metadata")
	return err
}

func NewClient() (*UtilsImpl, error) {
	var client *UtilsImpl
	clientset, err := GetClientset()
	if err != nil {
		return client, err
	}

	client = &UtilsImpl{
		Clientset: clientset,
	}
	return client, nil
}
func GetClientset() (*kubernetes.Clientset, error) {
	var clientset *kubernetes.Clientset
	config, err := clientcmd.BuildConfigFromFlags("", constants.KubeConfig)
	if err != nil {
		return clientset, err
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return clientset, err
	}
	return clientset, nil
}

func GetRoutedNetworkInterFace() (string, error) {
	routedInterface, err := nettest.RoutedInterface("ip", net.FlagUp|net.FlagBroadcast)
	if err != nil {
		return "", err
	}
	routedInterfaceName := routedInterface.Name
	return routedInterfaceName, nil
}

func GetIPv4ForInterfaceName(interfaceName string) (string, error) {
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if inter.Name == interfaceName {
			if addrs, err := inter.Addrs(); err == nil {
				for _, addr := range addrs {
					switch ip := addr.(type) {
					case *net.IPNet:
						if ip.IP.DefaultMask() != nil {
							return ip.IP.String(), nil
						}
					}
				}
			} else {
				return "", err
			}
		}
	}
	return "", fmt.Errorf("routedinterface not found so can't find ip")

}

func GetNodeIP() (string, error) {
	var err error
	routedInterfaceName, err := GetRoutedNetworkInterFace()
	if err != nil {
		return "", fmt.Errorf("failed to get routedNetworkinterface: %v", err)
	}
	routedIp, err := GetIPv4ForInterfaceName(routedInterfaceName)
	if err != nil {
		return "", fmt.Errorf("failed to get IPv4 for node_identification: %v", err)
	}
	return routedIp, nil
}

func IpForHttp(masterIp string) (string, error) {

	if net.ParseIP(masterIp).To4() != nil {
		return masterIp, nil
	} else if net.ParseIP(masterIp).To16() != nil {
		return "[" + masterIp + "]", nil
	}
	return "", fmt.Errorf("IP is invalid")
}

func KubernetesApiAvailable(cfg config.Config) error {

	caCertificate := constants.AdminCerts + "/ca.crt"
	clientCertificate := constants.AdminCerts + "/request.crt"
	keyFile := constants.AdminCerts + "/request.key"
	apiEndpoint := ""
	var err error

	//https://github.com/kubernetes/kubernetes/pull/46589 for role bindings to appear
	//     #healthz is better indication for availability, use https

	if cfg.ClusterRole == "master" {
		apiEndpoint = "localhost"
	} else {
		apiEndpoint, err = IpForHttp(cfg.MasterIp)
		if err != nil {
			return fmt.Errorf("failed to get apiendpoint for healthz : %v", err)
		}
	}

	healthzUrl := "https://" + apiEndpoint + ":" + strconv.Itoa(cfg.K8sApiPort) + "/healthz"

	caCert, err := ioutil.ReadFile(caCertificate)
	if err != nil {
		return fmt.Errorf("failed to readfile cacertificate")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCert, err := ioutil.ReadFile(clientCertificate)
	if err != nil {
		return fmt.Errorf("failed to readfile clientcertificate")
	}
	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCert)

	cert, err := tls.LoadX509KeyPair(clientCertificate, keyFile)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				ClientCAs:    clientCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}

	res, err := client.Get(healthzUrl)
	if err != nil {
		return err
	}

	if res.StatusCode >= 500 {
		return fmt.Errorf("apiServer not available")
	}
	return nil
}

// Copied from https://github.com/kubernetes/kubernetes/blob/39c76ba2edeadb84a115cc3fbd9204a2177f1c28/pkg/util/taints/taints.go#L241
// to avoid importing k8s.io/kubernetes as it leads to import errors and is not supported by upstream community.
// This function is not an exact copy. The difference is in how the taint on node is compared against the taint argument.
// AddOrUpdateTaint tries to add a taint to annotations list. Returns a new copy of updated Node and true if something was updated
// false otherwise.
func AddOrUpdateTaint(node *v1.Node, taint *v1.Taint) (*v1.Node, bool, error) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints

	var newTaints []v1.Taint
	updated := false
	for i := range nodeTaints {
		if taint.MatchTaint(&nodeTaints[i]) {
			if reflect.DeepEqual(*taint, nodeTaints[i]) {
				return newNode, false, nil
			}
			newTaints = append(newTaints, *taint)
			updated = true
			continue
		}

		newTaints = append(newTaints, nodeTaints[i])
	}

	if !updated {
		newTaints = append(newTaints, *taint)
	}

	newNode.Spec.Taints = newTaints
	return newNode, true, nil
}
