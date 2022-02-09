package kubeutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/drain"
)

func kubernetes_api_available() bool {

	cacertificate := AdminCerts + "/ca.crt"
	clientcertificate := AdminCerts + "/request.crt"
	keyfile := AdminCerts + "/request.key"
	api_endpoint := ""

	//https://github.com/kubernetes/kubernetes/pull/46589 for role bindings to appear
	//     #healthz is better indication for availability, use https

	if os.Getenv("ROLE") == "master" {
		api_endpoint = "localhost"
	} else {
		api_endpoint = ip_for_http(os.Getenv("MASTER_IP"))
	}

	healthzUrl := "https://" + api_endpoint + ":" + os.Getenv("K8S_API_PORT") + "/healthz"

	caCert, err := ioutil.ReadFile(cacertificate)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	clientCert, err := ioutil.ReadFile(clientcertificate)
	if err != nil {
		log.Fatal(err)
	}
	clientCertPool := x509.NewCertPool()
	clientCertPool.AppendCertsFromPEM(clientCert)

	cert, err := tls.LoadX509KeyPair(clientcertificate, keyfile)

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
		panic(err)
	}

	switch res.StatusCode {
	case 500, 501, 502, 503, 504, 505, 506, 507, 508, 510, 511:
		return false
	}
	return true
}

func ip_for_http(master_ip string) string {

	if net.ParseIP(master_ip).To4() != nil {
		return master_ip
	} else if net.ParseIP(master_ip).To16() != nil {
		return "[" + master_ip + "]"
	}
	return ""
}

func drain_node_from_apiserver(name string) error {

	kubeconfig := "/etc/pf9/kube.d/kubeconfigs/admin.yaml"

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
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

	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})

	err = drain.RunCordonOrUncordon(&helper, &node, true)
	if err != nil {
		return err
	}
	err = drain.RunNodeDrain(&helper, node.Name)
	if err != nil {
		return err
	}

	// TODO :
	// Add KubeStackShutDown annotation to the node on successful node drain
	// add_annotation_to_node ${node_ip} KubeStackShutDown

	return nil
}

//============================================

// function add_annotation_to_node()
// {
//     local node_identifier=$1
//     local annotation=$2
//     if ! err=$(${KUBECTL} annotate --overwrite node ${node_identifier} ${annotation}=true 2>&1 1>/dev/null ); then
//             echo "Warning: failed to annotate node ${node_identifier}: ${err}" >&2
//     fi
// }

//========================================
