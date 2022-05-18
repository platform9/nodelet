package misc

import (
	"context"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/kubeutils"
	"github.com/platform9/nodelet/nodelet/pkg/utils/phaseutils"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"

	"go.uber.org/zap"
)

type MiscPhase struct {
	HostPhase *sunpikev1alpha1.HostPhase
	log       *zap.SugaredLogger
	kubeUtils kubeutils.Utils
}

func NewMiscPhase() *MiscPhase {
	log := zap.S()
	return &MiscPhase{
		HostPhase: &sunpikev1alpha1.HostPhase{
			Name:  "Miscellaneous scripts and checks",
			Order: int32(constants.MiscPhaseOrder),
		},
		log:       log,
		kubeUtils: nil,
	}
}

func (m *MiscPhase) GetHostPhase() sunpikev1alpha1.HostPhase {
	return *m.HostPhase
}

func (m *MiscPhase) GetPhaseName() string {
	return m.HostPhase.Name
}

func (m *MiscPhase) GetOrder() int {
	return int(m.HostPhase.Order)
}

func (d *MiscPhase) Status(context.Context, config.Config) error {

	d.log.Infof("Running Status of phase: %s", d.HostPhase.Name)

	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *MiscPhase) Start(ctx context.Context, cfg config.Config) error {

	d.log.Infof("Running Start of phase: %s", d.HostPhase.Name)

	var err error
	if d.kubeUtils == nil || d.kubeUtils.IsInterfaceNil() {
		d.kubeUtils, err = kubeutils.NewClient()
		if err != nil {
			d.log.Error(errors.Wrap(err, "could not refresh k8s client"))
			phaseutils.SetHostStatus(d.HostPhase, constants.FailedState, err.Error())
			return err
		}
	}

	phaseutils.SetHostStatus(d.HostPhase, constants.RunningState, "")
	return nil
}

func (d *MiscPhase) Stop(ctx context.Context, cfg config.Config) error {

	d.log.Infof("Running Stop of phase: %s", d.HostPhase.Name)

	phaseutils.SetHostStatus(d.HostPhase, constants.StoppedState, "")
	return nil
}


function write_cloud_provider_config()
{
    if [[ -z "${KUBELET_CLOUD_CONFIG}" ]]; then
        echo "utils::write_cloud_provider_config: Env var KUBELET_CLOUD_CONFIG is empty, not writing a file"
    else
        echo "utils::write_cloud_provider_config: Writing kubelet cloud-config information for CLOUD_PROVIDER_TYPE: ${CLOUD_PROVIDER_TYPE} to path ${CLOUD_CONFIG_FILE}"
        echo $KUBELET_CLOUD_CONFIG | base64 --decode > ${CLOUD_CONFIG_FILE}
    fi
}

function node_is_up()
{
    local node_name=$1
    ${KUBECTL} get node ${node_name} #1> /dev/null
}

function ensure_fresh_container_running()
{
    local socket_name=$1
    local run_opts=$2
    local container_name=$3
    local container_img=$4
    local container_cmd=$5
    local container_cmd_args=$6

    ensure_container_destroyed $socket_name $container_name
    cmd="pf9ctr_run \
         run ${container_name:+--name ${container_name}} \
         ${run_opts} ${container_img} ${container_cmd} ${container_cmd_args}"

    # retry 2 times to handle spurious docker errors such as
    # https://github.com/docker/docker/issues/14048
    wait_until "${cmd}" 6 3
}

function ensure_container_destroyed()
{
    local socket_name=$1
    local container_name=$2
    echo "Ensuring container '$container_name' is destroyed"
    if pf9ctr_run inspect $container_name &> /dev/null; then
        stop_and_destroy_containers "$socket_name" "$container_name"
    fi
}

//containerd
function pf9ctr_run()
{
    if [ "$pf9_kube_http_proxy_configured" = "true" ]; then
        http_proxy=$http_proxy https_proxy=$https_proxy HTTP_PROXY=$HTTP_PROXY HTTPS_PROXY=$HTTPS_PROXY no_proxy=$no_proxy NO_PROXY=$NO_PROXY $cli -n k8s.io --cgroup-manager=$CONTAINERD_CGROUP -H unix://$socket "$@"
    else
        $cli -n k8s.io --cgroup-manager=$CONTAINERD_CGROUP -H unix://$socket "$@"
    fi
}
//docker
function pf9ctr_run()
{
    $cli -H unix://$socket "$@"
}
function stop_and_destroy_containers()
{
    local socket_name=$1
    local containers=$2
    echo "Stopping containers '$containers'"
    pf9ctr_run stop $containers
    echo "Destroying containers '$containers'"
    pf9ctr_run rm --force $containers
}