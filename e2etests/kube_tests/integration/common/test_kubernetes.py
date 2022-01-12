import json
import logging
import os

from proboscis.asserts import assert_true
from fabric.api import run
from pprint import pformat
import requests

from .test_etcd_backup import test_etcdbackup
from .test_rbac import test_kubernetes_rbac
from .test_k8s_network_policy import test_k8s_network_policy
from .test_k8s_metallb import test_k8s_metallb
from pf9lab.retry import retry
from pf9lab.utils import typical_du_fabric_settings
from kube_tests.lib.kubeconfig import get_kubeconfig
from kube_tests.lib.kubernetes import Kubernetes
from kube_tests.integration.common import constants
from kube_tests.lib.command_utils import run_command
from pf9lab.dependent_actions import Task, Action


THIS_DIR = os.path.dirname(__file__)
BUILD_DIR = os.path.abspath('%s/../../../../build' % THIS_DIR)
KUBERNETES_TEST_PATH = os.path.join(BUILD_DIR, 'kubernetes-test')
KUBECTL_PATH = os.path.join(KUBERNETES_TEST_PATH, 'kubectl')
KUBETEST_PATH = os.path.join(KUBERNETES_TEST_PATH, 'kubetest.sh')
if not (os.path.exists(KUBETEST_PATH) and os.path.exists(KUBECTL_PATH)):
    raise RuntimeError('kubetest.sh or kubectl not found. '
                       'Run `make kubernetes-test`')
skip_sidekicks_check = os.getenv('SKIP_SIDEKICKS_RESPONDING_CHECK')

log = logging.getLogger(__name__)


def k8s_setup(qbert, cluster, keystone_user, keystone_pass):
    kubeconfig = get_kubeconfig(qbert, cluster['name'], keystone_user, keystone_pass)
    # The kubeconfig server URL will use the external DNS name if available
    api_server = kubeconfig.cluster(cluster['name'])['server']
    log.info('##### API SERVER #####: {0}'.format(api_server))
    return kubeconfig, api_server

def test_k8s_api(qbert, cluster_uuids, keystone_user, keystone_pass, du_fqdn, keystone_token, expected_runtime="docker"):
    def action_closure(action_name, uuid):
        args = lambda: [qbert, uuid, keystone_user, keystone_pass, du_fqdn, keystone_token, expected_runtime]
        return Action(action_name, _test_k8s_api, args)
    task = Task()
    for uuid in cluster_uuids:
        task.add(action_closure('test-k8s-api-{}'.format(uuid), uuid))
    _start_task(task, 'test_k8s_api')


@retry(log=log, max_wait=600, interval=20)
def _test_k8s_api(qbert, cluster_uuid, keystone_user, keystone_pass, du_fqdn, keystone_token, expected_runtime):
    cluster = qbert.get_cluster_by_uuid(cluster_uuid)
    kubeconfig, api_server = k8s_setup(qbert, cluster, keystone_user, keystone_pass)
    log.info('Kubernetes API test using hostname (%s)', api_server)
    with kubeconfig.cluster_ca_file(cluster['name']) as ca_file_path:
        if not skip_sidekicks_check:
            _wait_sidekicks_responding(du_fqdn, qbert, cluster)
        kc_token = kubeconfig.user(keystone_user)['token']
        k8s = Kubernetes(api_server=api_server, verify=ca_file_path,
                         token=kc_token)

        _wait_for_k8s_apiserver_up(api_server)
        k8s.list_nodes()
        _verify_k8s_proxy_up(qbert.api_url, cluster['uuid'], kc_token,
                             keystone_token, qbert.tenant_id)
        with kubeconfig.as_file() as kubeconfig_file:
            if not _verify_k8s_proxy_direct_up(kubeconfig_file, cluster['name']):
                log.error('Kubernetes API _direct_ connection to cluster (%s) failed', cluster['name'])
                return False
        _wait_for_k8s_nodes_up(qbert, cluster, k8s)
        _wait_for_k8s_nodes_ready(k8s)
        _wait_for_k8s_master_pod(k8s)
        _verify_runtime_config(k8s, cluster['runtimeConfig'])
        _verify_nodes_patched_to_use_dynamic_kubelet_configmap(k8s)
        _verify_nodes_deployed_correct_container_runtime(k8s, expected_runtime)
        # If the cluster does NOT allow workloads on master run the following tests...
        if 'allowWorkloadsOnMaster' in cluster and not cluster['allowWorkloadsOnMaster']:
            _verify_master_nodes_tainted(k8s)
            _verify_tolerations_applied(k8s)
        _wait_dashboard_addon_ready(k8s)
    return True


@retry(log=log, max_wait=10, interval=2)
def _verify_nodes_deployed_correct_container_runtime(k8s, expected_runtime):
    log.info('Waiting for k8s nodes to be Ready')
    return k8s.verify_node_container_runtime(expected_runtime)


def _wait_sidekicks_responding(du_fqdn, qbert, cluster):
    for node in list(qbert.list_nodes_by_uuid().values()):
        if node['clusterUuid'] == cluster['uuid'] and node['isMaster']:
            _wait_sidekick_responding(du_fqdn, node['uuid'])


@retry(log=log, max_wait=300, interval=30)
def _wait_sidekick_responding(du_fqdn, host_id):
    with typical_du_fabric_settings(du_fqdn):
        cmd = ('curl -sS -X GET '
               '-H "Content-type: application/json" '
               'localhost:3011/v1/hosts/{0}')
        res = run(cmd.format(host_id), shell=True, pty=False)
        assert_true(res.succeeded)
        if 'Not Found' in res:
            log.info('Host {0} not found in sidekick server'
                     .format(host_id))
            return False
        sidekick = json.loads(res)
        if sidekick.get('hostid') != host_id:
            log.info('Expected {0} in sidekick server but found {1}'
                     .format(host_id, sidekick.get('hostid')))
            return False
        return True


@retry(log=log, max_wait=120, interval=10)
def _wait_for_k8s_master_pod(k8s):
    log.info('Waiting for k8s master pod to be listed')
    pods = k8s.get_all_pods(constants.SYSTEM_NAMESPACE)['items']
    return next((pod for pod in pods
                 if 'k8s-master' in pod['metadata']['name']), None)


@retry(log=log, max_wait=120, interval=10)
def _wait_for_k8s_nodes_ready(k8s):
    log.info('Waiting for k8s nodes to be Ready')
    k8s_nodes = k8s.list_nodes()
    for node_name, node_metadata in k8s_nodes.items():
        conditions = node_metadata['status']['conditions']
        if not any(c['type'] == 'Ready' and c['status'] == 'True'
                   for c in conditions):
            log.info('Node %s not yet in k8s Ready state', node_name)
            return False
    return True


@retry(log=log, max_wait=600, interval=10)
def _wait_for_k8s_nodes_up(qbert, cluster, k8s):
    log.info('Waiting for k8s nodes to be up')
    qbert_nodes = list(qbert.list_nodes_by_uuid().values())
    k8s_node_names = list(k8s.list_nodes())

    # For non-bareOS clusters, node names should match hostnames
    # changing the hostname to lower case as in case of azure cloud
    # provider, few nodes may have hostname with both upper and lower
    # cased alphabets e.g. k8s-worker-a6d6413f-d285-4ee4-a32d-045bda2ae8e900000A
    # K8s takes hostname and stores it in kubelet node metadata in lowercase
    # e.g. /api/v1/nodes/k8s-worker-a6d6413f-d285-4ee4-a32d-045bda2ae8e900000a
    # https://github.com/kubernetes/kubernetes/issues/71140#issuecomment-441703265
    qbert_hostnames = [node_metadata['name'].lower()
                          for node_metadata in qbert_nodes
                          if node_metadata['clusterName'] == cluster["name"]]
    # For bare OS clusters, node names should match primary IPs
    if cluster["cloudProviderType"] == "local":
        qbert_hostnames = [node_metadata['primaryIp']
                            for node_metadata in qbert_nodes
                            if node_metadata['clusterName'] == cluster["name"]]

    if len(qbert_hostnames) != len(k8s_node_names):
        return False

    k8s_node_names.sort()
    qbert_hostnames.sort()
    k8s_and_qbert_nodes = list(zip(k8s_node_names, qbert_hostnames))

    for k8s_node_name, qbert_node_name in k8s_and_qbert_nodes:
        # Due to k8s cloud provider controller code which chooses node names, it could be the case that
        # the qbert node name reported by hostagent includes more of an FQDN than the k8s node uses
        if k8s_node_name not in qbert_node_name:
            log.error("k8s_node_name {} not a substring of qbert_node_name {}".format(k8s_node_name, qbert_node_name))
            return False
    return True


@retry(log=log, max_wait=120, interval=10)
def _wait_dashboard_addon_ready(k8s):
    log.info('Waiting for the kubernetes-dashboard pods to be ready')
    pods = k8s.get_all_pods(constants.DASHBOARD_NAMESPACE)['items']

    if len(pods) == 0:
        log.info('No pods created yet in the %s namespace', constants.DASHBOARD_NAMESPACE)

        # Temporary workaround to ensure that this test does not fail for
        # the upgrade tests from 4.2 to 4.3.
        log.info('Checking if old kubernetes-dashboard exists in the %s '
                 'namespace', constants.SYSTEM_NAMESPACE)
        pods = [pod for pod in k8s.get_all_pods(constants.SYSTEM_NAMESPACE)['items']
                if pod['metadata']['name'].startswith('kubernetes-dashboard')]
        if len(pods) == 0:
            log.info('Dashboard not found in the %s namespace either', constants.SYSTEM_NAMESPACE)
            return False

    for pod in pods:
        if pod['status']['phase'] != "Running":
            log.info('Pod %s not yet in "Running" state', pod['metadata']['name'])
            return False
    return True

def _verify_k8s_proxy_up(api_url, cluster_uuid, token, keystone_token, tenant_id, version='v2'):
    api_proxy_server_url = '%s/%s/%s/clusters/%s/k8sapi' % (api_url, version, tenant_id, cluster_uuid)
    api_proxy_server_headers = {'X-Auth-Token': keystone_token}
    k8s_proxy = Kubernetes(api_server=api_proxy_server_url,
                           verify=False,
                           headers=api_proxy_server_headers,
                           token=token)
    k8s_proxy.list_nodes()

# Get the direct k8s connection
def _verify_k8s_proxy_direct_up(kubeconfig_file, cluster_name):
    # Save the kubernetes
    # External_name = cluster_name + "-pf9"
    try:
        from kubernetes import client, config
        config.load_kube_config(config_file=kubeconfig_file, context=cluster_name+"-pf9")
        v1 = client.CoreV1Api()
        ret = v1.list_node()
        for i in ret.items:
            log.info("Got items %s", i.metadata.name)
        return True
    except Exception as exc:
        log.exception(exc)
    return False

@retry(log=log, max_wait=120, interval=10)
def _wait_for_k8s_apiserver_up(apiserver):
    log.info('Waiting for k8s apiserver to be up')
    try:
        requests.get(apiserver, timeout=10, verify=False)
        return True
    except requests.exceptions.RequestException as exc:
        log.warn('Unable to connect k8s apiserver %s: %s', apiserver,
                 exc.message)
    return False


def _verify_runtime_config(k8s, runtime_config):
    if runtime_config == constants.DEFAULT_RUNTIME_CONFIG:
        log.info('Waiting to verify listing cluster roles fails')
        _verify_list_cluster_roles_fails(k8s)
    elif runtime_config == constants.ALL_APIS_RUNTIME_CONFIG:
        log.info('Waiting to verify listing cluster roles')
        _verify_list_cluster_roles(k8s)
    else:
        log.warn('Skipping runtime_config test due to unknown value: %s',
                 runtime_config)


@retry(log=log, max_wait=15, interval=5)
def _verify_list_cluster_roles(k8s):
    try:
        k8s.list_cluster_roles()
        return True
    except Exception as exc:
        log.exception('Exception verifying cluster roles list: %s', exc.message)
        return False


@retry(log=log, max_wait=15, interval=5)
def _verify_list_cluster_roles_fails(k8s):
    try:
        k8s.list_cluster_roles()
        log.warn('Unexpected response while listing cluster roles')
        return False
    except Exception as e:
        assert_true('404' in str(e))
        log.info('Got expected exception when listing cluster roles')
        return True


@retry(log=log, max_wait=10, interval=5)
def _verify_master_nodes_tainted(k8s):
    log.info('Verifying master nodes are tainted')
    pods = k8s.get_all_pods(constants.SYSTEM_NAMESPACE)['items']
    nodes = k8s.get_all_nodes()['items']

    for pod in pods:
        if 'k8s-master' in pod['metadata']['name']:
            node_name = pod['spec']['nodeName']
            for node in nodes:
                if node_name in node['metadata']['name']:
                    if 'taints' in node['spec']:
                        # This logic will not work when there is more than one taint!
                        for taint in node['spec']['taints']:
                            if 'node-role.kubernetes.io/master' not in taint['key']:
                                log.warn('%s does not have the correct taint', node_name)
                                return False
                    else:
                        log.warn('%s does not have any taints', node_name)
                        return False

    return True


@retry(log=log, max_wait=10, interval=5)
def _verify_tolerations_applied(k8s):
    log.info('Verifying tolerations are applied on critical pods')
    pods = k8s.get_all_pods(constants.SYSTEM_NAMESPACE)['items']

    # List of critical components that we want to verify have the toleration
    master_addons = ('metrics-server', 'dashboard', 'kube-dns')

    # It might be redundant to check every running pod, an alternative would be to check deployments
    for pod in pods:
        if any(name in pod['metadata']['name'] for name in master_addons):
            #Ensure 'tolerations' key exists otherwise the label 'node-role.kubenetes.io/master' gives a false positive
            if 'tolerations' in pod['spec']:
                if 'node-role.kubernetes.io/master' not in str(pod['spec']['tolerations']):
                    log.warn('%s does not have the appropriate tolerations', pod['metadata']['name'])
                    return False
            else:
                log.warn('%s does not have any tolerations', pod['metadata']['name'])
                return False

    return True

@retry(log=log, max_wait=60, interval=10)
def _verify_nodes_patched_to_use_dynamic_kubelet_configmap(k8s):
    log.info('Verifying nodes are successfully using ConfigMap for their kubelet config')
    k8s_nodes = k8s.get_all_nodes()['items']

    for node in k8s_nodes:
        if node['spec'].get('configSource') is None:
            log.warn('{}: Node is not using ConfigMap for source of kubelet config'.format(node['metadata']['name']))
            return False
    return True

def test_k8s_autoscaler(qbert, cluster_uuids, keystone_user, keystone_pass, cloud_provider_type=None):
    def action_closure(action_name, uuid, teardown):
        args = lambda: ([qbert, uuid, keystone_user, keystone_pass],
                        {'cloud_provider_type': cloud_provider_type, 'teardown': teardown})
        return Action(action_name, _test_k8s_autoscaler, args)
    task = Task()
    teardowns = [False for n in range(len(cluster_uuids))]
    teardowns[0] = True
    for i, uuid in enumerate(cluster_uuids):
        task.add(action_closure('test-k8s-autoscaler-{}'.format(uuid), uuid, teardowns[i]))
    _start_task(task, 'test_k8s_autoscaler', 20, 2400) # timeout after 2400 seconds

def _test_k8s_autoscaler(qbert, cluster_uuid, keystone_user, keystone_pass, cloud_provider_type=None, teardown=False):
    cluster = qbert.get_cluster_by_uuid(cluster_uuid)
    kubeconfig, api_server = k8s_setup(qbert, cluster, keystone_user, keystone_pass)
    with kubeconfig.as_file() as kubeconfig_path:
        _test_kubernetes_autoscaler(kubeconfig_path, api_server,
                                    cloud_provider_type, teardown)

def _test_kubernetes_autoscaler(kubeconfig_path, api_server, cloud_provider_type, teardown):
    log.info('Running kubernetes cluster autoscaler tests')
    kubetest_env = os.environ.copy()
    kubetest_env['KUBECTL'] = KUBECTL_PATH
    if cloud_provider_type:
        kubetest_env['CLOUD_PROVIDER_TYPE'] = cloud_provider_type

    kubeconfig_opt = ('--kubeconfig=%s --server=%s' %
                        (kubeconfig_path, api_server))
    kubetest_opts = (KUBETEST_PATH, 'nginx', kubeconfig_opt)
    cmd = '%s %s setup %s' % kubetest_opts
    return_code, output = run_command(cmd, kubetest_env)
    if return_code != 0:
        msg = '\n'.join([
            '{0} setup failed'. format('nginx'),
            'Command:', cmd,
            'Stdout+err:', output
        ])
        log.error(msg)
        raise Exception(msg)

        if teardown:
            cmd = '%s %s teardown %s' % kubetest_opts
            _, _ = run_command(cmd, kubetest_env)

def test_k8s_examples(qbert, cluster_uuids, keystone_user, keystone_pass, cloud_provider_type=None):
    def action_closure(action_name, uuid, teardown):
        args = lambda: ([qbert, uuid, keystone_user, keystone_pass],
                        {'cloud_provider_type': cloud_provider_type, 'teardown': teardown})
        return Action(action_name, _test_k8s_examples, args)
    task = Task()
    teardowns = [False for n in range(len(cluster_uuids))]
    teardowns[0] = True
    for i, uuid in enumerate(cluster_uuids):
        task.add(action_closure('test-k8s-examples-{}'.format(uuid), uuid, teardowns[i]))
    _start_task(task, 'test_k8s_examples', 20, 1200) # timeout after 1200 seconds


def _test_k8s_examples(qbert, cluster_uuid, keystone_user, keystone_pass, cloud_provider_type=None, teardown=False):
    cluster = qbert.get_cluster_by_uuid(cluster_uuid)
    kubeconfig, api_server = k8s_setup(qbert, cluster, keystone_user, keystone_pass)
    with kubeconfig.as_file() as kubeconfig_path:
        _test_kubernetes_examples(kubeconfig_path, api_server,
                                  cloud_provider_type,
                                  teardown)


def _test_kubernetes_examples(kubeconfig_path, api_server, cloud_provider_type, teardown):
    log.info('Running kubernetes example tests')
    kubetest_env = os.environ.copy()
    kubetest_env['KUBECTL'] = KUBECTL_PATH
    if cloud_provider_type:
        kubetest_env['CLOUD_PROVIDER_TYPE'] = cloud_provider_type

    kubeconfig_opt = ('--kubeconfig=%s --server=%s' %
                      (kubeconfig_path, api_server))

    examples = ['guestbook']
    for ex in examples:
        kubetest_opts = (KUBETEST_PATH, ex, kubeconfig_opt)
        cmd = '%s %s setup %s' % kubetest_opts
        return_code, output = run_command(cmd, kubetest_env)
        if return_code != 0:
            msg = '\n'.join([
                '{0} setup failed'.format(ex),
                'Command:', cmd,
                'Stdout+err:', output
            ])
            log.error(msg)
            raise Exception(msg)

        if teardown:
            cmd = '%s %s teardown %s' % kubetest_opts
            _, _ = run_command(cmd, kubetest_env)

def test_k8s_rbac(qbert, cluster_uuids, keystone_user, keystone_pass,
                  du_fqdn, keystone_token, cloud_provider_type=None):
    def action_closure(action_name, uuid):
        args = lambda: ([qbert, uuid, keystone_user, keystone_pass, du_fqdn, keystone_token],
                        {'cloud_provider_type': cloud_provider_type})
        return Action(action_name, _test_k8s_rbac, args)
    task = Task()
    for uuid in cluster_uuids:
        task.add(action_closure('test-k8s-rbac-{}'.format(uuid), uuid))
    _start_task(task, 'test_k8s_rbac')

@retry(log=log, max_wait=300, interval=30)
def _test_k8s_rbac(qbert, cluster_uuid, keystone_user, keystone_pass, du_fqdn, ks_token, cloud_provider_type=None):
    cluster = qbert.get_cluster_by_uuid(cluster_uuid)
    kubeconfig, api_server = k8s_setup(qbert, cluster, keystone_user, keystone_pass)
    test_kubernetes_rbac(kubeconfig, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, KUBECTL_PATH)
    return True

def test_network_policy(qbert, cluster_uuids, keystone_user, keystone_pass,
                  du_fqdn, keystone_token, cloud_provider_type=None):
    def action_closure(action_name, uuid):
        args = lambda: ([qbert, uuid, keystone_user, keystone_pass, du_fqdn, keystone_token],
                        {'cloud_provider_type': cloud_provider_type})
        return Action(action_name, _test_network_policy, args)
    task = Task()
    for uuid in cluster_uuids:
        task.add(action_closure('test_network_policy-{}'.format(uuid), uuid))
    _start_task(task, 'test_network_policy')

@retry(log=log, max_wait=600, interval=30)
def _test_network_policy(qbert, cluster_uuid, keystone_user, keystone_pass, du_fqdn, ks_token,
                    cloud_provider_type=None):
    cluster = qbert.get_cluster_by_uuid(cluster_uuid)
    if cluster['networkPlugin'] != 'calico':
        log.info('skipping calico cni policy tests since network backend is not calico')
        return True
    kubeconfig, api_server = k8s_setup(qbert, cluster, keystone_user, keystone_pass)
    test_k8s_network_policy(kubeconfig, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, KUBECTL_PATH, cluster['networkPlugin'])
    return True

def test_metallb(qbert, cluster_uuid, keystone_user, keystone_pass):
    def action_closure(action_name, uuid):
        args = lambda: ([qbert, uuid, keystone_user, keystone_pass])
        return Action(action_name, _test_metallb, args)
    task = Task()
    task.add(action_closure('test_network_policy-{}'.format(cluster_uuid), cluster_uuid))
    _start_task(task, 'test_metallb')

def _test_metallb(qbert, cluster_uuid, keystone_user, keystone_pass):
    cluster = qbert.get_cluster_by_uuid(cluster_uuid)
    kubeconfig, api_server = k8s_setup(qbert, cluster, keystone_user, keystone_pass)
    test_k8s_metallb(kubeconfig, qbert, cluster, api_server, KUBECTL_PATH)
    return True

def test_etcdbackup_for_cluster(qbert, du_fqdn, cluster_uuid):
    def action_closure(action_name, uuid):
        args = lambda: ([qbert, du_fqdn, cluster_uuid])
        return Action(action_name, _test_etcdbackup_for_cluster, args)
    task = Task()
    task.add(action_closure('test_etcdbackup_for_cluster-{}'.format(cluster_uuid), cluster_uuid))
    _start_task(task, 'test_etcdbackup_for_cluster')

def _test_etcdbackup_for_cluster(qbert, du_fqdn, cluster_uuid):
    test_etcdbackup(qbert, du_fqdn, cluster_uuid)

def _start_task(task, function_name, interval=20, timeout_timer=120*10):
    task.start()
    timeout = False
    try:
        task.wait_complete(interval, timeout_timer)
    except RuntimeError as e:
        log.error('Failed while waiting for {}: {}'.format(function_name, e))
        timeout = True
    errors = task.error()
    if errors or timeout:
        log.error('{} failed'.format(function_name))
        log.error('Errors so far:\n%s', pformat(task.error()))
        raise RuntimeError('{} failed'.format(function_name))
