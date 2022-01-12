import logging
import os
import time

from proboscis.asserts import assert_equal

from kube_tests.integration.common import wait_for_cluster_taskstatus, wait_for_cluster_status
from kube_tests.integration.common import constants

log = logging.getLogger(__name__)
SYNC_DELAY = 30 #secs

def test_bare_os_cluster_update(qbert, uuid):
    cluster = qbert.get_cluster_by_uuid(uuid)
    new_metallb_cidr = constants.METALLB_CIDR_UPDATE

    body = {'metallbCidr': new_metallb_cidr}
    qbert.update_cluster(uuid, body)
    wait_for_cluster_update_complete(qbert, uuid)

def wait_for_cluster_update_complete(qbert, uuid):
    wait_for_cluster_taskstatus('success', qbert, [uuid])
    # The cluster may enter 'pending' state on next sync.
    # If update operation is too fast it may directly enter
    # 'ok' state. Just sleep for SYNC_DELAY and then wait
    # for cluster state to become 'ok'.
    time.sleep(SYNC_DELAY)
    wait_for_cluster_status('ok', qbert, [uuid])
