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
	"time"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"

	"github.com/pkg/errors"
	"golang.org/x/net/nettest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/drain"
)

type Utils interface {
	GetNodeIP() (string, error)
	GetRoutedNetworkInterFace() (string, error)
	GetIPv4ForInterfaceName(string) (string, error)
	GetNodeIdentifier(config.Config) (string, error)
	GetNodeFromK8sApi(context.Context, string) (*v1.Node, error)
	AddLabelsToNode(context.Context, string, map[string]string) error
	AddAnnotationsToNode(context.Context, string, map[string]string) error
	RemoveAnnotationsFromNode(context.Context, string, []string) error
	AddTaintsToNode(context.Context, string, []*v1.Taint) error
	DrainNodeFromApiServer(context.Context, string) error
	UncordonNode(context.Context, string) error
	K8sApiAvailable(config.Config) error
	PreventAutoReattach() error
	IpForHttp(string) (string, error)
	IsInterfaceNil() bool
}

type UtilsImpl struct {
	Clientset kubernetes.Interface
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

func GetClientset() (kubernetes.Interface, error) {
	var clientset kubernetes.Interface
	config, err := clientcmd.BuildConfigFromFlags("", constants.KubeConfig)
	if err != nil {
		return clientset, errors.Wrapf(err, "failed to build config from kubeconfig")
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return clientset, errors.Wrapf(err, "failed to create clientset")
	}
	return clientset, nil
}

func (u *UtilsImpl) IsInterfaceNil() bool {
	return u == nil
}

func (u *UtilsImpl) AddLabelsToNode(ctx context.Context, nodeName string, labelsToAdd map[string]string) error {

	node, err := u.GetNodeFromK8sApi(ctx, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to add labels: %v to node: %v", labelsToAdd, nodeName)
	}
	metaData := node.ObjectMeta
	if metaData.Labels == nil {
		metaData.Labels = make(map[string]string)
	}
	for k, v := range labelsToAdd {
		metaData.Labels[k] = v
	}
	node.ObjectMeta = metaData
	_, err = u.Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update node with newly added labels")
	}
	return nil
}

func (u *UtilsImpl) AddAnnotationsToNode(ctx context.Context, nodeName string, annotsToAdd map[string]string) error {
	node, err := u.GetNodeFromK8sApi(ctx, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to add annotations: %v to node: %v", annotsToAdd, nodeName)
	}
	metaData := node.ObjectMeta
	if metaData.Annotations == nil {
		metaData.Annotations = make(map[string]string)
	}
	for k, v := range annotsToAdd {
		metaData.Annotations[k] = v
	}
	node.ObjectMeta = metaData
	_, err = u.Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update node with newly added annotations")
	}
	return nil
}

func (u *UtilsImpl) RemoveAnnotationsFromNode(ctx context.Context, nodeName string, annotsToRemove []string) error {
	node, err := u.GetNodeFromK8sApi(ctx, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to remove annotations: %v from node: %v", annotsToRemove, nodeName)
	}
	metaData := node.ObjectMeta
	if metaData.Annotations == nil {
		return nil
	}
	for _, v := range annotsToRemove {
		delete(metaData.Annotations, v)
	}
	node.ObjectMeta = metaData
	_, err = u.Clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to update node with removed annotations")
	}
	return nil
}

func (u *UtilsImpl) AddTaintsToNode(ctx context.Context, nodeName string, taintsToAdd []*v1.Taint) error {
	node, err := u.GetNodeFromK8sApi(ctx, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to add taints: %v to node: %v", taintsToAdd, nodeName)
	}
	for _, taint := range taintsToAdd {
		taintedNode, updated, err := AddOrUpdateTaint(node, taint)
		if err != nil {
			return errors.Wrapf(err, "failed to add taints: %v to node: %v", taintsToAdd, nodeName)
		}
		if !updated {
			return errors.Wrapf(err, "failed to add taints: %v to node: %v", taintsToAdd, nodeName)
		}
		_, err = u.Clientset.CoreV1().Nodes().Update(ctx, taintedNode, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to update node with newly added taints")
		}
	}
	return nil
}

func (u *UtilsImpl) DrainNodeFromApiServer(ctx context.Context, nodeName string) error {

	helper := drain.Helper{
		Ctx:                 ctx,
		Client:              u.Clientset,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             time.Duration(300) * time.Second,
		DeleteEmptyDirData:  true,
		Out:                 os.Stdout,
		ErrOut:              os.Stdout,
		DisableEviction:     true,
	}
	node, err := u.GetNodeFromK8sApi(ctx, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to drain node")
	}
	err = drain.RunCordonOrUncordon(&helper, node, true)
	if err != nil {
		return errors.Wrapf(err, "failed to cordon node")
	}
	err = drain.RunNodeDrain(&helper, node.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to drain node")
	}
	return nil
}

func (u *UtilsImpl) GetNodeFromK8sApi(ctx context.Context, nodeName string) (*v1.Node, error) {
	var node *v1.Node
	node, err := u.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return node, errors.Wrapf(err, "failed to get node")
	}
	return node, nil
}

func (u *UtilsImpl) UncordonNode(ctx context.Context, nodeName string) error {

	helper := drain.Helper{
		Ctx:                 ctx,
		Client:              u.Clientset,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             time.Duration(300) * time.Second,
		DeleteEmptyDirData:  true,
		Out:                 os.Stdout,
		ErrOut:              os.Stdout,
		DisableEviction:     true,
	}

	node, err := u.GetNodeFromK8sApi(ctx, nodeName)
	if err != nil {
		return errors.Wrapf(err, "failed to uncordon node")
	}
	err = drain.RunCordonOrUncordon(&helper, node, false)
	if err != nil {
		return errors.Wrapf(err, "failed to uncordon node")
	}
	if node.Spec.Unschedulable {
		return errors.Wrapf(err, "warning: node is still cordoned or cannot be fetched")
	}
	return nil
}

func (u *UtilsImpl) PreventAutoReattach() error {

	// Unconditionally delete the qbert metaData file to prevent re-auth
	err := os.Remove("/opt/pf9/hostagent/extensions/fetch_qbert_metadata")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to remove qbert metadata file")
	}
	return nil
}

func (u *UtilsImpl) GetRoutedNetworkInterFace() (string, error) {
	routedInterface, err := nettest.RoutedInterface("ip", net.FlagUp|net.FlagBroadcast)
	if err != nil {
		return "", err
	}
	routedInterfaceName := routedInterface.Name
	return routedInterfaceName, nil
}

func (u *UtilsImpl) GetIPv4ForInterfaceName(interfaceName string) (string, error) {
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if inter.Name == interfaceName {
			addrs, err := inter.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addrs {
				switch ip := addr.(type) {
				case *net.IPNet:
					if ip.IP.DefaultMask() != nil {
						return ip.IP.String(), nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("routedinterface not found so can't find ip")
}

func (u *UtilsImpl) GetNodeIP() (string, error) {
	var err error
	routedInterfaceName, err := u.GetRoutedNetworkInterFace()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get routed network interface")
	}
	routedIp, err := u.GetIPv4ForInterfaceName(routedInterfaceName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get node IP")
	}
	return routedIp, nil
}

func (u *UtilsImpl) IpForHttp(masterIp string) (string, error) {

	if net.ParseIP(masterIp).To4() != nil {
		return masterIp, nil
	} else if net.ParseIP(masterIp).To16() != nil {
		return "[" + masterIp + "]", nil
	}
	return "", fmt.Errorf("invalid IP")
}

func (u *UtilsImpl) K8sApiAvailable(cfg config.Config) error {

	caCertificate := fmt.Sprintf("%s/ca.crt", constants.AdminCerts)
	clientCertificate := fmt.Sprintf("%s/request.crt", constants.AdminCerts)
	keyFile := fmt.Sprintf("%s/request.key", constants.AdminCerts)
	apiEndpoint := ""
	var err error

	if cfg.ClusterRole == constants.RoleMaster {
		apiEndpoint = constants.LocalHostString
	} else {
		apiEndpoint, err = u.IpForHttp(cfg.MasterIp)
		if err != nil {
			return errors.Wrapf(err, "failed to check K8s API available")
		}
	}

	//https://github.com/kubernetes/kubernetes/pull/46589 for role bindings to appear
	//     #healthz is better indication for availability, use https

	healthzUrl := fmt.Sprintf("https://%s:%s/healthz", apiEndpoint, cfg.K8sApiPort)
	caCert, err := ioutil.ReadFile(caCertificate)
	if err != nil {
		return errors.Wrapf(err, "failed to read ca certificate")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCert, err := ioutil.ReadFile(clientCertificate)
	if err != nil {
		return errors.Wrapf(err, "failed to read client certificate")
	}
	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCert)

	cert, err := tls.LoadX509KeyPair(clientCertificate, keyFile)
	if err != nil {
		return errors.Wrapf(err, "could not load x509 key pair")
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
		return errors.Wrapf(err, "could not get to healthz")
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("api server not available")
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

func (u *UtilsImpl) GetNodeIdentifier(cfg config.Config) (string, error) {

	var err error
	var nodeIdentifier string
	if cfg.CloudProviderType == constants.LocalCloudProvider && cfg.UseHostname == constants.TrueString {
		nodeIdentifier, err = os.Hostname()
		if err != nil {
			return nodeIdentifier, errors.Wrapf(err, "failed to get hostName for node identification")
		}
		if nodeIdentifier == "" {
			return nodeIdentifier, fmt.Errorf("nodeIdentifier is null")
		}
	} else {
		nodeIdentifier, err = u.GetNodeIP()
		if err != nil {
			return nodeIdentifier, errors.Wrapf(err, "failed to get node IP address for node identification")
		}
		if nodeIdentifier == "" {
			return nodeIdentifier, fmt.Errorf("nodeIdentifier is null")
		}
	}
	return nodeIdentifier, nil
}
