package containerruntime

import (
	"context"
	"os"

	"github.com/containerd/containerd"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type ContainerdImpl struct {
	Service string
	Cli     string
	Crictl  string
	Socket  string
	Client  *containerd.Client
	Proxies Proxy
}

type Proxy struct {
	http_proxy  string
	https_proxy string
	HTTP_PROXY  string
	HTTPS_PROXY string
	no_proxy    string
	NO_PROXY    string
}

var (
	proxy            Proxy
	containerdclient *containerd.Client
)

func NewContainerd() Runtime {

	if constants.Pf9KubeHttpProxyConfigured == "true" {
		proxy = Proxy{
			http_proxy:  os.Getenv("http_proxy"),
			https_proxy: os.Getenv("https_proxy"),
			HTTP_PROXY:  os.Getenv("HTTP_PROXY"),
			HTTPS_PROXY: os.Getenv("HTTPS_PROXY"),
			no_proxy:    os.Getenv("no_proxy"),
			NO_PROXY:    os.Getenv("NO_PROXY"),
		}
	}
	containerdclient, err = containerd.New(constants.ContainerdSocket)
	if err != nil {
		zap.S().Info("failed to create container runtime client")
	}
	return &ContainerdImpl{
		Service: "containerd",
		Cli:     "/opt/pf9/pf9-kube/bin/nerdctl",
		Crictl:  "/opt/pf9/pf9-kube/bin/crictl",
		Socket:  constants.ContainerdSocket,
		Client:  containerdclient,
		Proxies: proxy,
	}
}

func (r *ContainerdImpl) EnsureFreshContainerRunning(ctx context.Context, cfg config.Config, containerName string, containerImage string, Ip string, port string) error {
	return nil
}

func (r *ContainerdImpl) EnsureContainerDestroyed(ctx context.Context, cfg config.Config, containerName string) error {
	return nil
}

func (r *ContainerdImpl) EnsureContainerStoppedOrNonExistent(ctx context.Context, cfg config.Config, containerName string) error {
	return nil
}
