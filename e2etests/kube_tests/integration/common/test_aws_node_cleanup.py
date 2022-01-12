import logging

from proboscis.asserts import assert_equal, fail
import kube_tests.integration.common as utils

log = logging.getLogger(__name__)


def test_aws_node_cleanup(qbert, cluster_uuid, ec2):
    prev_cluster_size = utils.get_cluster_size(qbert, cluster_uuid)
    terminated_inst_name = _terminate_master_instance(cluster_uuid, ec2)
    utils.wait_until_node_absent(qbert, cluster_uuid, terminated_inst_name)
    utils.wait_until_cluster_size(qbert, cluster_uuid, prev_cluster_size)


def _terminate_master_instance(cluster_uuid, ec2):
    cluster_filter = {
        'Name': 'tag:ClusterUuid',
        'Values': [cluster_uuid]
    }
    master_inst = None
    for inst in ec2.instances.filter(Filters=[cluster_filter]):
        if _is_master_instance(inst):
            master_inst = inst
            break
    if master_inst is None:
        fail('Did not find master node for cluster %s' % cluster_uuid)
    # Cache the name, since it is cleared when the node is terminated
    master_inst_private_dns_name = master_inst.private_dns_name
    resp = master_inst.terminate()
    assert_equal(resp['ResponseMetadata']['HTTPStatusCode'], 200)
    return master_inst_private_dns_name


def _is_master_instance(inst):
    """
    :param inst boto3.ec2.Instance
    """
    asg_tag = next((tag for tag in inst.tags
                    if tag['Key'] == 'aws:autoscaling:groupName'), None)
    return bool(asg_tag and 'master' in asg_tag['Value'])






