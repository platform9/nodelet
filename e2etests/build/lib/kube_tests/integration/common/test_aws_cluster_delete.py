import logging
import os
import time
import fabric.api as fabric
from pf9lab.utils import typical_du_fabric_settings

import requests

from proboscis.asserts import assert_equal
from pf9lab.retry import retry

CLUSTER_DESTROY_DELAY = 180
KUBERNETES_RESOURCE_TAG = 'KubernetesCluster'
KUBERNETES_RESOURCE_PREFIX = 'kubernetes.io/cluster/'
MYSQL_FS_PATH = '/mnt/mysqlfs/qbert/cloud/aws/'


log = logging.getLogger(__name__)


def test_aws_cluster_delete(qbert, du_ip, cluster_uuids, subnet_ids,
                            cp_uuid, region):
    if os.getenv('AWS_CLUSTER_DONT_DELETE'):
        return
    aws_subnets, aws_private_subnets = subnet_ids

    delete_uuids = []

    qbert.delete_cluster_by_uuid(cluster_uuids['aws_uuid'])
    delete_uuids.append(cluster_uuids['aws_uuid'])

    if qbert.subnet_shareable:
        qbert.delete_cluster_by_uuid(cluster_uuids['aws_uuid_shared'])
        delete_uuids.append(cluster_uuids['aws_uuid_shared'])

    time.sleep(CLUSTER_DESTROY_DELAY)

    _wait_for_cluster_delete(qbert, delete_uuids)

    log.info("SKIPPING TF ARTIFACT DELETION VALIDATION UNTIL PMK-1799 IS FIXED!")
    # _validate_tf_artifacts_deletion(du_ip, cluster_uuids['aws_uuid'])

    _wait_subnets_untagged(aws_subnets + aws_private_subnets)

    for subnet in aws_subnets + aws_private_subnets:
        _delete_subnet(subnet)

    qbert.delete_cluster_by_uuid(cluster_uuids['aws_complete_uuid'])
    _wait_for_cluster_delete(qbert, cluster_uuids['aws_complete_uuid'])

    _wait_for_vpc_deletion(qbert, list(cluster_uuids.values()), cp_uuid, region)

@retry(log=log, max_wait=600, interval=20)
def _wait_for_cluster_delete(qbert, cluster_uuids):
    clusters = list(qbert.list_clusters().values())
    for cluster in clusters:
        if cluster['uuid'] in cluster_uuids:
            return False
    return True

@retry(log=log, max_wait=300, interval=20)
def _wait_subnet_untagged(subnet):
    subnet.reload()
    tags = subnet.tags
    if tags is None:
        return True
    # make sure both kinds of tags are removed. As old and new clusters can share existing subnets
    return all( (t['Key'] != KUBERNETES_RESOURCE_TAG) and (not t['Key'].startswith(KUBERNETES_RESOURCE_PREFIX))
               for t in tags)


def _wait_subnets_untagged(subnets):
    """
    :param subnets boto3.ec2.subnets
    """
    for subnet in subnets:
        _wait_subnet_untagged(subnet)


@retry(log=log, max_wait=900, interval=20)
def _wait_for_vpc_deletion(qbert, cluster_uuids, cp_uuid, region):
    log.info('Waiting for vpcs backing clusters: {0} to be deleted'
             .format(cluster_uuids))
    region_info = qbert.get_cloud_provider_region_info(cp_uuid, region)
    for vpc in region_info['vpcs']:
        for tag in vpc['Tags']:
            key = tag['Key']
            val = tag['Value']
            if key == 'ClusterUuid' and val in cluster_uuids:
                log.info('Cluster: {0} has vpc: {1} backing it'
                         .format(tag['Value'], vpc['VpcId']))
                return False
    return True

@retry(log=log, max_wait=300, interval=20)
def _delete_subnet(subnet):
    subnet.delete()
    return True

@retry(log=log, max_wait=30, interval=5)
def _validate_tf_artifacts_deletion(du_ip, cluster_uuid):
    artifact_regex = MYSQL_FS_PATH + cluster_uuid + '*'

    cmd = 'ls ' +  artifact_regex
    with typical_du_fabric_settings(du_ip):
        log.info("Executing DU command %s " % cmd)
        ret = fabric.run(cmd)
        log.info(ret)
        if ret.succeeded > 0:
            raise Exception('Terraform artifacts not cleaned.')
    return True


