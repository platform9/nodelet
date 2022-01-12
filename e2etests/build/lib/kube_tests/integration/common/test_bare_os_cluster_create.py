import logging
import random
import os
from fabric.api import run

from pf9lab.retry import retry
from kube_tests.lib.command_utils import wait_for_kube_service_running
from kube_tests.testbeds.aws_utils import create_dns_record
from kube_tests.integration.common import constants
import pf9lab.hosts.authorize as labrole

DEFAULT_INVALID_PRIMARY_IP = '0.0.0.0'
DEFAULT_POOL = 'defaultPool'
CLUSTER_FLANNEL_IFACE_LABEL = os.getenv('FLANNEL_IFACE_LABEL', 'eth0')
CLUSTER_FLANNEL_PUBLIC_IFACE_LABEL = os.getenv('FLANNEL_PUBLIC_IFACE_LABEL', 'eth0')
CLUSTER_MASTER_VIP_IFACE = os.getenv('CLUSTER_MASTER_VIP_IFACE', 'eth0')
PF9_SVC_NAME = 'pf9-kube'


log = logging.getLogger(__name__)


def test_bare_os_cluster_create(qbert, hosts, vip_port_ip, roleVersion, num_masters=3, attach_seperately=False,
                                enable_metallb=False, metallb_cidr=False, runtime="docker"):
    api_fqdn = _create_dns_record(hosts[0:num_masters])
    cluster_name = _get_or_deploy_cluster(qbert, api_fqdn, vip_port_ip, enable_metallb, metallb_cidr, roleVersion, runtime=runtime)
    node_names = [host['hostname'] for host in hosts]
    nodes = [{'nodeName': host['hostname'], 'isMaster': 0} for host in hosts]
    _wait_for_nodes_to_appear_in_qbert(qbert, node_names)
    _wait_for_nodes_to_have_valid_primary_ip(qbert, node_names)
    _wait_for_nodes_to_appear_ok(qbert, node_names)

    # mark the master nodes
    for x in range(num_masters):
        nodes[x]['isMaster'] = 1

    _ensure_nodes_attached(qbert, cluster_name, nodes, hosts, num_masters,
                           attach_separately=attach_seperately)

    return cluster_name


def _get_or_deploy_cluster(qbert, api_fqdn, master_vip_ipv4, enable_metallb, metallb_cidr, roleVersion, runtime="docker"):
    if os.getenv('QBERT_CLUSTER_NAME'):
        return os.getenv('QBERT_CLUSTER_NAME')

    cluster_name = 'test-bare-os-cluster-{0}'.format(random.randint(0, 10000))
    nodepool_uuid = qbert.list_nodepools()[DEFAULT_POOL]['uuid']
    qbert.create_cluster({
        'name': cluster_name,
        'nodePoolUuid': nodepool_uuid,
        'containersCidr': constants.CONTAINERS_CIDR,
        'servicesCidr': constants.SERVICES_CIDR,
        'keystoneEnabled': True,
        'appCatalogEnabled': os.getenv('USE_APP_CATALOG') == 'true',
        'debug': 'true',
        'flannelIfaceLabel': CLUSTER_FLANNEL_IFACE_LABEL,
        'flannelPublicIfaceLabel': CLUSTER_FLANNEL_PUBLIC_IFACE_LABEL,
        'externalDnsName': api_fqdn,
        'allowWorkloadsOnMaster': os.getenv('ALLOW_WORKLOADS_ON_MASTER', 'true') == 'true',
        'masterVipIpv4': master_vip_ipv4,
        'masterVipIface': CLUSTER_MASTER_VIP_IFACE,
        'enableMetallb': enable_metallb,
        'metallbCidr': metallb_cidr,
        'networkPlugin': 'flannel',
        'kubeRoleVersion': roleVersion,
        'containerRuntime': runtime
    }, 'v4')
    _wait_for_cluster_to_be_created(qbert, cluster_name)
    return cluster_name


def _create_dns_record(hosts):
    random_tag = random.randint(0, 9999)
    build_id = os.getenv('TEAMCITY_BUILD_ID', None)
    if not build_id:
        # Prefix nob to random num to indicate that it is not
        # associated with a build id
        build_id = 'nob{}'.format(random.randint(0, 999))
    else:
        # Prefix bld to build_num to indicate that it is an actual
        # build id association
        build_id = 'bld{}'.format(build_id)
    api_fqdn = 'api-{0}-{1}.k8s.platform9.horse'.format(build_id, random_tag)

    node_public_ips = [host['ip'] for host in hosts]
    create_dns_record(node_public_ips, api_fqdn)
    return api_fqdn


@retry(log=log, max_wait=30, interval=5)
def _wait_for_cluster_to_be_created(qbert, cluster_name):
    log.info('Waiting for cluster {0} to be created'.format(cluster_name))
    return cluster_name in qbert.list_clusters()


@retry(log=log, max_wait=240, interval=20)
def _wait_for_nodes_to_appear_in_qbert(qbert, node_names):
    log.info('Waiting for nodes to appear in qbert, nodes: {0}'.format(node_names))
    return set(node_names) <= set(qbert.list_nodes())


@retry(log=log, max_wait=240, interval=20)
def _wait_for_nodes_to_have_valid_primary_ip(qbert, node_names):
    log.info('Waiting for nodes to have valid primaryIp, nodes: {0}'.format(node_names))
    nodes = qbert.list_nodes()
    for node_name in node_names:
        if node_name not in nodes:
            log.warning('Node %s not yet known to qbert', node_name)
            return False
        if nodes[node_name]['primaryIp'] is None or nodes[node_name]['primaryIp'] == DEFAULT_INVALID_PRIMARY_IP or nodes[node_name]['primaryIp'] == "null":
            log.info('Node %s does not have a valid primaryIp. Current value: %s',
                     node_name, nodes[node_name]['primaryIp'])
            return False
    return True

@retry(log=log, max_wait=240, interval=20)
def _wait_for_nodes_to_appear_ok(qbert, node_names):
    log.info('Waiting for nodes to have status ok, nodes: {0}'.format(node_names))
    nodes = qbert.list_nodes()
    for node_name in node_names:
        if node_name not in nodes:
            log.warning('Node %s not yet known to qbert', node_name)
            return False
        if nodes[node_name]['status'] != "ok":
            log.info('Node %s does not have ok status. Current status: %s',
                     node_name, nodes[node_name]['status'])
            return False
    return True

def _ensure_nodes_attached(qbert, cluster_name, nodes, hosts, num_masters,
                           attach_separately):
    if attach_separately:
        masters = nodes[:num_masters]
        workers = nodes[num_masters:]
        log.info('Attempting to attach masters {0} to cluster'.format(masters))
        _ensure_nodes_attached_to_cluster(qbert, cluster_name, masters, hosts)
        log.info('Attempting to attach workers {0} to cluster'.format(workers))
        _ensure_nodes_attached_to_cluster(qbert, cluster_name, workers, hosts)
    else:
        log.info('Attaching all nodes {0} at the same time'.format(nodes))
        _ensure_nodes_attached_to_cluster(qbert, cluster_name,
                                          nodes, hosts)


@retry(log=log, max_wait=1200, interval=30, tolerate_exceptions=True)
def _wait_for_successful_attach(qbert, nodes, cluster_name):
    """Retry is needed so since attach_nodes() can fail if there's an
    existing master, and it is not in 'ok' state"""
    qbert.attach_nodes(nodes, cluster_name, 'v3')
    return True


def _ensure_nodes_attached_to_cluster(qbert, cluster_name, nodes, hosts):

    log.info('Ensuring nodes {0} attached to cluster {1}'
             .format(nodes, cluster_name))
    _wait_for_successful_attach(qbert, nodes, cluster_name)
    node_names = [node_item['nodeName'] for node_item in nodes]
    _wait_for_nodes_to_be_attached(qbert, cluster_name, node_names)
    try:
        _wait_for_pf9_kube_to_start(qbert, node_names, hosts)
    except Exception as exc:
        for node_name in node_names:
            kubelog = qbert.get_kubelog(node_name)
            log.debug(
                '------------ start of kube.log for node %s ------------',
                node_name)
            log.debug(kubelog)
            log.debug(
                '------------   end of kube.log for node %s ------------',
                node_name)
        raise exc


@retry(log=log, max_wait=120, interval=5, tolerate_exceptions=False)
def _wait_for_nodes_to_be_attached(qbert, cluster_name, node_names):
    log.info('Waiting for nodes {0} to be attached'.format(node_names))
    nodes = qbert.list_nodes()
    for node_name in node_names:
        if node_name not in nodes:
            log.info('Node %s not yet authorized.', node_name)
            return False
        if nodes[node_name]['clusterName'] != cluster_name:
            log.info('Node %s not yet attached to cluster %s',
                     node_name, cluster_name)
            return False
    return True


@retry(log=log, max_wait=1200, interval=30, tolerate_exceptions=False)
def _wait_for_pf9_kube_to_start(qbert, node_names, hosts):
    log.info('Waiting for service pf9-kube to run on nodes {0}'.format(node_names))
    nodes = qbert.list_nodes()
    for node_name in node_names:
        if node_name not in nodes:
            log.warning('Node %s not yet known to qbert', node_name)
            return False
        node_ip = nodes[node_name]['primaryIp']
        try:
            node_pubic_ip = _get_host_public_ip(node_ip, hosts)
        except StopIteration:
            log.warning('ip %s not yet known to qbert', node_ip)
            return False
        wait_for_kube_service_running(node_pubic_ip)
    return True


def _get_host_public_ip(private_ip, hosts):
    return next(h['ip'] for h in hosts
                if h['private_ip'] == private_ip)
