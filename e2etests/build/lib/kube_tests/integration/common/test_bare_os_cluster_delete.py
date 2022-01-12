import os
import logging
import time

from fabric.api import run
from proboscis.asserts import assert_equal

from pf9lab.retry import retry
from kube_tests.testbeds.utils import fatal_errors_host_fabric_settings
from kube_tests.lib.command_utils import wait_for_kube_service_stopped
from .test_bare_os_cluster_create import _get_host_public_ip, PF9_SVC_NAME

log = logging.getLogger(__name__)

NODE_DETACH_DELAY_SEC = 10 # seconds


def test_bare_os_cluster_delete(qbert, hosts, cluster_name, num_masters=3, detach_separately=False):
    if os.getenv('BARE_OS_CLUSTER_DONT_DELETE'):
        log.info('Skipping cluster {0} deletion'.format(cluster_name))
        return

    node_names = [host['hostname'] for host in hosts]
    ensure_nodes_detached_from_cluster(qbert, cluster_name,
                                       node_names, hosts, num_masters,
                                       detach_separately=detach_separately)
    _ensure_cluster_deleted(qbert, cluster_name)
    wait_all_docker_daemons_stopped(hosts)


def _ensure_cluster_deleted(qbert, name):
    log.info('Deleting cluster {0}'.format(name))
    qbert.delete_cluster(name)
    _wait_for_cluster_to_be_deleted(qbert, name)


@retry(log=log, max_wait=30, interval=5)
def _wait_for_cluster_to_be_deleted(qbert, name):
    log.info('Waiting for cluster {0} to be deleted'.format(name))
    return name not in qbert.list_clusters()


@retry(log=log, max_wait=360, tolerate_exceptions=True)
def wait_all_docker_daemons_stopped(hosts):
    _verify_docker_daemon_count(hosts, 0)
    return True


def _verify_docker_daemon_count(hosts, expected_number):
    # FIXME(daniel) accept node_names parameter
    for host in hosts:
        with fatal_errors_host_fabric_settings(host['ip']):
            run('touch /root/.cloud-warnings.skip')
            ret = run('ps -ef | grep docker | grep -- "--graph '
                      '/var/lib/docker" | grep -v grep | wc -l')
            assert_equal(int(ret), expected_number, 'docker daemon cnt')

def ensure_nodes_detached_from_cluster(qbert, cluster_name, node_names, hosts, num_masters, detach_separately):
    if detach_separately:
        workers = node_names[num_masters:]
        masters = node_names[:num_masters]
        log.info('Attempting to detach workers {0} to cluster'.format(workers))
        _ensure_nodes_detached_from_cluster_internal(
            qbert, cluster_name, workers, hosts)
        log.info('Attempting to detach master {0} to cluster'.format(masters))
        _ensure_nodes_detached_from_cluster_internal(
            qbert, cluster_name, masters, hosts)
    else:
        log.info('Attempting to detach all nodes {0} to cluster'.format(node_names))
        _ensure_nodes_detached_from_cluster_internal(
            qbert, cluster_name, node_names, hosts)


def _detach_nodes_from_cluster(qbert, cluster_name, node_names):
    for node_name in reversed(node_names):
        log.info('detaching %s from cluster %s', node_name, cluster_name)
        qbert.detach_node(node_name, cluster_name, 'v3')
        time.sleep(NODE_DETACH_DELAY_SEC)


@retry(log=log, max_wait=30, interval=5)
def _wait_for_nodes_to_be_detached(qbert, cluster_name, node_names):
    log.info('Waiting for nodes {0} to be detached'.format(node_names))
    nodes = qbert.list_nodes()
    for node_name in node_names:
        if node_name not in nodes:
            raise RuntimeError('Node %s has been removed from qbert.' %
                               node_name)
        if nodes[node_name]['clusterName'] == cluster_name:
            log.info('Node %s not yet detached from cluster %s',
                     node_name, cluster_name)
            return False
    return True


@retry(log=log, max_wait=600, interval=30, tolerate_exceptions=False)
def _wait_for_pf9_kube_to_stop(qbert, node_names, hosts):
    log.info('Waiting for service pf9-kube to stop on nodes {0}'.format(node_names))
    nodes = qbert.list_nodes()
    for node_name in node_names:
        if node_name not in nodes:
            raise RuntimeError('Node %s has been removed from qbert.' %
                               node_name)
        node_ip = nodes[node_name]['primaryIp']
        node_pubic_ip = _get_host_public_ip(node_ip, hosts)
        wait_for_kube_service_stopped(node_pubic_ip)
    return True


def _ensure_nodes_detached_from_cluster_internal(qbert, cluster_name, node_names, hosts):
    log.info('Ensuring nodes {0} detached from cluster'.format(node_names))
    _detach_nodes_from_cluster(qbert, cluster_name, node_names)
    _wait_for_nodes_to_be_detached(qbert, cluster_name, node_names)
    _wait_for_pf9_kube_to_stop(qbert, node_names, hosts)
