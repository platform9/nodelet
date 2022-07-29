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
	"strings"
	"time"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"github.com/platform9/nodelet/nodelet/pkg/utils/netutils"

	"github.com/pkg/errors"
	"github.com/shipengqi/kube"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/drain"
)

type Utils interface {
	GetNodeFromK8sApi(context.Context, string) (*v1.Node, error)
	AddLabelsToNode(context.Context, string, map[string]string) error
	AddAnnotationsToNode(context.Context, string, map[string]string) error
	RemoveAnnotationsFromNode(context.Context, string, []string) error
	AddTaintsToNode(context.Context, string, []*v1.Taint) error
	DrainNodeFromApiServer(context.Context, string) error
	UncordonNode(context.Context, string) error
	K8sApiAvailable(config.Config) error
	PreventAutoReattach() error
	IsInterfaceNil() bool
	EnsureDns(config.Config) error
	EnsureAppCatalog() error
	ApplyYamlConfigFiles([]string) error
	WriteCloudProviderConfig(config.Config) error
}

type UtilsImpl struct {
	Clientset kubernetes.Interface
}

var netUtil = netutils.New()
var file = fileio.New()

// NewCient initialize UtilsImpl with new creted clientset
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

// GetClientset returns clientset created from config
func GetClientset() (kubernetes.Interface, error) {
	var clientset kubernetes.Interface
	config, err := clientcmd.BuildConfigFromFlags("", constants.KubeConfig)
	if err != nil {
		return clientset, errors.Wrap(err, "failed to build config from kubeconfig")
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return clientset, errors.Wrap(err, "failed to create clientset")
	}
	return clientset, nil
}

func (u *UtilsImpl) IsInterfaceNil() bool {
	return u == nil
}

// AddLabelsToNode adds labels to node
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
		return errors.Wrap(err, "failed to update node with newly added labels")
	}
	return nil
}

// AddAnnotationsToNode adds annotations to node
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
		return errors.Wrap(err, "failed to update node with newly added annotations")
	}
	return nil
}

// RemoveAnnotationsFromNode removes annotations from node
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
		return errors.Wrap(err, "failed to update node with removed annotations")
	}
	return nil
}

// AddTaintsToNode adds taints to node
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
			return errors.Wrap(err, "failed to update node with newly added taints")
		}
	}
	return nil
}

// DrainNodeFromApiServer drains node from K8s server
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

// GetNodeFromK8sApi returns node from K8s api with given nodename
func (u *UtilsImpl) GetNodeFromK8sApi(ctx context.Context, nodeName string) (*v1.Node, error) {
	var node *v1.Node
	node, err := u.Clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return node, errors.Wrap(err, "failed to get node")
	}
	return node, nil
}

// UncordonNode uncordons node
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

// PreventAutoReattach removes qbert metaData file if present
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

// K8sApiAvailable checks if K8s api server is available
func (u *UtilsImpl) K8sApiAvailable(cfg config.Config) error {

	caCertificate := fmt.Sprintf("%s/ca.crt", constants.AdminCerts)
	clientCertificate := fmt.Sprintf("%s/request.crt", constants.AdminCerts)
	keyFile := fmt.Sprintf("%s/request.key", constants.AdminCerts)
	apiEndpoint := ""
	var err error

	if cfg.ClusterRole == constants.RoleMaster {
		apiEndpoint = constants.LocalHostString
	} else {
		apiEndpoint, err = netUtil.IpForHttp(cfg.MasterIp)
		if err != nil {
			return errors.Wrap(err, "failed to check K8s API available")
		}
	}

	//https://github.com/kubernetes/kubernetes/pull/46589 for role bindings to appear
	//     #healthz is better indication for availability, use https

	healthzUrl := fmt.Sprintf("https://%s:%s/healthz", apiEndpoint, cfg.K8sApiPort)
	caCert, err := ioutil.ReadFile(caCertificate)
	if err != nil {
		return errors.Wrap(err, "failed to read ca certificate")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCert, err := ioutil.ReadFile(clientCertificate)
	if err != nil {
		return errors.Wrap(err, "failed to read client certificate")
	}
	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCert)

	cert, err := tls.LoadX509KeyPair(clientCertificate, keyFile)
	if err != nil {
		return errors.Wrap(err, "could not load x509 key pair")
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
		return errors.Wrap(err, "could not get to healthz")
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("api server not available")
	}
	return nil
}

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

// EnsureDns applies coredns yaml file
func (u *UtilsImpl) EnsureDns(cfg config.Config) error {
	k8sRegistry := constants.K8sRegistry
	if cfg.K8sPrivateRegistry != "" {
		k8sRegistry = cfg.K8sPrivateRegistry
	}

	dnsIP, _ := netUtil.AddrConv(cfg.ServicesCIDR, 10)
	type dataToAdd struct {
		DnsIP       string
		K8sRegistry string
		DnsEntries  map[string]string
	}

	hostFileLines, err := file.ReadFileByLine(cfg.CoreDNSHostsFile)
	hostEntries := make(map[string]string)
	for _, entry := range hostFileLines {
		if strings.HasPrefix(entry, "#") {
			// Is a comment, not a host entry
			continue
		}
		fields := strings.SplitN(entry, " ", 2)
		if len(fields) <= 1 {
			continue
		}
		ip := net.ParseIP(fields[0])
		if ip == nil {
			// Not a valid IP, and not a #comment - violates /etc/hosts format, error out or ignore?
			continue
		}
		hostEntries[fields[0]] = fields[1]
	}

	data := dataToAdd{
		DnsIP:       dnsIP,
		K8sRegistry: k8sRegistry,
		DnsEntries:  hostEntries,
	}
	err = file.NewYamlFromTemplateYaml(constants.CoreDNSTemplate, constants.CoreDNSFile, data)
	if err != nil {
		return errors.Wrap(err, "could not create Coredns yaml")
	}
	err = u.ApplyYamlConfigFiles([]string{constants.CoreDNSFile})
	if err != nil {
		return errors.Wrap(err, "could not apply Coredns yaml")
	}
	if cfg.Debug == "false" {
		err = os.Remove(constants.CoreDNSFile)
		if err != nil {
			return err
		}
	}

	pods, err := u.Clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{LabelSelector: "k8s-app=kube-dns"})
	if err != nil {
		return errors.Wrap(err, "Failed to list coredns Pods")
	}
	for _, pod := range pods.Items {
		zap.S().Infof("Deleting coreDNS Pod: %s", pod.Name)
		err = u.Clientset.CoreV1().Pods("kube-system").Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "Failed to delete CoreDNS replica %v", pod.Name)
		}
	}
	return nil
}

// EnsureAppCatalog applies app catalog yaml files
func (u *UtilsImpl) EnsureAppCatalog() error {

	appCatalog := fmt.Sprintf("%s/appcatalog", constants.ConfigDstDir)

	files, err := file.ListFilesWithPatterns(appCatalog, []string{"*.yaml", "*.yml"})
	if err != nil {
		return errors.Wrapf(err, "could not get files from:%s", appCatalog)
	}
	log := zap.S()
	log.Infof("applying files: %q", files)
	err = u.ApplyYamlConfigFiles(files)
	if err != nil {
		return errors.Wrap(err, "could not apply app catalog yamls")
	}

	return nil
}

// ApplyYamlConfigFiles applies the yaml files
func (u *UtilsImpl) ApplyYamlConfigFiles(files []string) error {

	flags := genericclioptions.NewConfigFlags(false)
	flags.KubeConfig = &constants.KubeConfig
	cfg := kube.NewConfig(flags)
	cli := kube.New(cfg)
	_, err := cli.Dial()
	if err != nil {
		return err
	}
	err = cli.Apply(files)
	if err != nil {
		return err
	}
	return nil
}

func (u *UtilsImpl) WriteCloudProviderConfig(cfg config.Config) error {
	if cfg.KubeletCloudConfig == "" {
		zap.S().Info("KubeletCloudConfig file is empty is not writing cloud config file")
	} else {
		zap.S().Infof("Writing kubelet cloud-config information for CloudProviderType: %s to path %s", cfg.CloudProviderType, constants.CloudConfigFile)
		err := file.WriteToFileWithBase64Decoding(constants.CloudConfigFile, cfg.KubeletCloudConfig)
		if err != nil {
			return err
		}
	}
	return nil
}
