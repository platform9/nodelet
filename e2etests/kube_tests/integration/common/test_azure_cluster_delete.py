import logging
import os
import time

import requests

from proboscis.asserts import assert_equal
from pf9lab.retry import retry

CLUSTER_DESTROY_DELAY = 180

log = logging.getLogger(__name__)

def test_azure_cluster_delete(qbert, cluster_uuids):
    if os.getenv('AZURE_CLUSTER_DONT_DELETE'):
        return

    # Delete BYON cluster first as the network from complete cluster is being
    # used by it. Wait till the cluster is deleted before deleting the other.
    qbert.delete_cluster_by_uuid(cluster_uuids['azure_network_provided_uuid'])
    time.sleep(CLUSTER_DESTROY_DELAY)
    _wait_for_cluster_delete(qbert, cluster_uuids['azure_network_provided_uuid'])

    qbert.delete_cluster_by_uuid(cluster_uuids['azure_complete_uuid'])
    time.sleep(CLUSTER_DESTROY_DELAY)
    _wait_for_cluster_delete(qbert, cluster_uuids['azure_complete_uuid'])

@retry(log=log, max_wait=1200, interval=60)
def _wait_for_cluster_delete(qbert, cluster_uuid):
    clusters = list(qbert.list_clusters().values())
    for cluster in clusters:
        if cluster['uuid'] == cluster_uuid:
            return False
    return True
