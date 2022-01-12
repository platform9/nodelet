import logging
import os

from proboscis.asserts import assert_equal

from kube_tests.integration.common import wait_for_cluster_taskstatus, wait_for_cluster_status, get_cluster_size, wait_until_cluster_size

log = logging.getLogger(__name__)


def test_aws_cluster_update(qbert, uuid):
    cluster = qbert.get_cluster_by_uuid(uuid)
    if 'renamed' in cluster['name']:
        log.warn('Cluster %s has already been renamed. Skipping cluster update test', cluster['name'])
        return
    new_cluster_size = get_cluster_size(qbert, uuid) + 1
    new_cluster_name = '{0}-renamed'.format(cluster['name'])
    new_num_workers = cluster['numWorkers'] + 1

    body = {'name': new_cluster_name, 'numWorkers': new_num_workers}
    if 'USE_SPOT_INSTANCES' in os.environ:
        new_num_spot_workers = int(cluster['cloudProperties']['numSpotWorkers']) + 1
        body['numSpotWorkers'] = new_num_spot_workers
        new_cluster_size += 1

    qbert.update_cluster(uuid, body)
    wait_for_cluster_update_complete(qbert, uuid, new_cluster_size)

    cluster = qbert.get_cluster_by_uuid(uuid)
    assert_equal(cluster['name'], new_cluster_name)
    assert_equal(cluster['numWorkers'], new_num_workers)

    if 'USE_SPOT_INSTANCES' in os.environ:
        assert_equal(cluster['cloudProperties']['numSpotWorkers'],
                     str(new_num_spot_workers))


# IAAS-8046 Workaround
# The cluster update operation transitions the cluster state to ('success',
# 'ok') as soon as `terraform apply` returns, but if nodes are added, the
# cluster state transitions to ('success', 'pending') before transitioning to
# ('success', 'ok') once the new nodes converge. This will wait for all of the
# state transitions to complete.
def wait_for_cluster_update_complete(qbert, uuid, new_cluster_size):
    wait_for_cluster_taskstatus('success', qbert, [uuid])
    wait_for_cluster_status('ok', qbert, [uuid])
    wait_until_cluster_size(qbert, uuid, new_cluster_size)
    wait_for_cluster_status('ok', qbert, [uuid])
