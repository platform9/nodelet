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
	"strconv"
	"time"

	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"

	"golang.org/x/net/nettest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/drain"
)

func Kubernetes_api_available(cfg config.Config) error {

	cacertificate := constants.AdminCerts + "/ca.crt"
	clientcertificate := constants.AdminCerts + "/request.crt"
	keyfile := constants.AdminCerts + "/request.key"
	api_endpoint := ""
	var err error

	//https://github.com/kubernetes/kubernetes/pull/46589 for role bindings to appear
	//     #healthz is better indication for availability, use https

	if cfg.ClusterRole == "master" {
		api_endpoint = "localhost"
	} else {
		api_endpoint, err = Ip_for_http(cfg.MasterIp)
		if err != nil {
			return fmt.Errorf("failed to get apiendpoint for healthz : %v", err)
		}
	}

	healthzUrl := "https://" + api_endpoint + ":" + strconv.Itoa(cfg.K8sApiPort) + "/healthz"

	caCert, err := ioutil.ReadFile(cacertificate)
	if err != nil {
		return fmt.Errorf("failed to readfile cacertificate")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCert, err := ioutil.ReadFile(clientcertificate)
	if err != nil {
		return fmt.Errorf("failed to readfile clientcertificate")
	}
	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCert)

	cert, err := tls.LoadX509KeyPair(clientcertificate, keyfile)
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

func Ip_for_http(master_ip string) (string, error) {

	if net.ParseIP(master_ip).To4() != nil {
		return master_ip, nil
	} else if net.ParseIP(master_ip).To16() != nil {
		return "[" + master_ip + "]", nil
	}
	return "", fmt.Errorf("IP is invalid")
}

func Drain_node_from_apiserver(NodeName string) error {

	config, err := clientcmd.BuildConfigFromFlags("", constants.KubeConfig)
	if err != nil {
		return fmt.Errorf("error in building config from clientcmd using kubeconfig")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error in creating clienset")
	}

	helper := drain.Helper{
		Ctx:                 context.TODO(),
		Client:              clientset,
		Force:               true,
		GracePeriodSeconds:  -1,
		IgnoreAllDaemonSets: true,
		Timeout:             time.Duration(300) * time.Second,
		DeleteEmptyDirData:  true,
		Out:                 os.Stdout,
		ErrOut:              os.Stdout,
		DisableEviction:     true,
	}

	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), NodeName, metav1.GetOptions{})
	if err != nil {

		return fmt.Errorf("failed to create node from CoreV1")
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

func GetRoutedNetworkInterFace() (string, error) {
	rinterface, err := nettest.RoutedInterface("ip", net.FlagUp|net.FlagBroadcast)
	if err != nil {
		return "", err
	}
	routedInterfaceName := rinterface.Name
	return routedInterfaceName, nil
}

func GetIPv4ForInterfaceName(ifname string) (string, error) {
	interfaces, _ := net.Interfaces()
	for _, inter := range interfaces {
		if inter.Name == ifname {
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
