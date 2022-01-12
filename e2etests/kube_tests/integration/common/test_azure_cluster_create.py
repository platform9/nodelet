import logging
import os
import random
import time
import uuid

from proboscis.asserts import assert_equal, assert_true, fail

from pf9lab.retry import retry
from kube_tests.integration.common import constants, wait_for_cluster_taskstatus, wait_for_cluster_status

CLUSTER_LAUNCH_DELAY = 30
small_profile = {'numMasters': 1, 'numWorkers': 1}
large_profile = {'numMasters': 3, 'numWorkers': 3}
autoscaler_profile = {'numMasters': 1, 'numMinWorkers': 1, 'numMaxWorkers': 3, 'enableCAS': True}

log = logging.getLogger(__name__)

def test_azure_cluster_create(qbert, cp_uuid, azure_testbed_profile,
                            template_key, is_public, runtime_config, pf9_kube_role):
    suffix = _get_cluster_suffix()
    visibility = 'public' if is_public else 'private'

    all_cluster_uuids = dict()
    cluster_create_req = azure_testbed_profile.get_cluster_create_input()
    cluster_create_req['nodePoolUuid'] = _get_cp_nodepool_uuid(qbert, cp_uuid)
    cluster_create_req['kubeRoleVersion'] = pf9_kube_role
    azure_complete_uuid = _deploy_cluster(
        qbert, 'azure-complete-{}-{}'.format(visibility, suffix),
        cluster_create_req)
    all_cluster_uuids['azure_complete_uuid'] = azure_complete_uuid
    wait_for_cluster_taskstatus('converging', qbert, [azure_complete_uuid])

    # Create a cluster with network shared from the previous cluster
    # to test BYON configuration
    region_info = qbert.get_cloud_provider_region_info(cp_uuid,
                    cluster_create_req['location'])
    vnet_info = _get_vnet_info(region_info, azure_complete_uuid)
    cluster_create_req.update(vnet_info)
    # Switch to small profile for BYON configuration
    cluster_create_req.update(autoscaler_profile)
    cluster_create_req.pop('numWorkers')
    azure_network_provided_uuid = _deploy_cluster(
        qbert, 'azure-network-provided-{}-{}'.format(visibility, suffix),
        cluster_create_req)
    all_cluster_uuids['azure_network_provided_uuid'] = azure_network_provided_uuid

    wait_for_cluster_taskstatus('success', qbert, list(all_cluster_uuids.values()))
    return all_cluster_uuids

def _get_cluster_suffix():
    user = os.environ['USER']
    build_num = os.getenv('BUILD_NUMBER')
    if build_num:
        suffix = 'bld{0}'.format(build_num)
    else:
        suffix = str(uuid.uuid4())[-3:]
    return '{0}-{1}'.format(user, suffix)

def _get_vnet_info(region_info, cluster_uuid):
    vnet_name = 'virtual-network-' + cluster_uuid
    vnet = [vn for vn in region_info['virtualNetworks'] if vn['name'] == vnet_name][0]
    vnet_resource_group = vnet['resourceGroup']
    master_subnets = [s for s in vnet['properties']['subnets'] if 'master' in s['name']]
    worker_subnets = [s for s in vnet['properties']['subnets'] if 'worker' in s['name']]
    assert_true(len(master_subnets) > 0 and len(worker_subnets) > 0)
    vnet_master_subnet_name = master_subnets[0]['name']
    vnet_worker_subnet_name = worker_subnets[0]['name']
    return {'vnetName': vnet_name,
            'vnetResourceGroup': vnet_resource_group,
            'masterSubnetName': vnet_master_subnet_name,
            'workerSubnetName': vnet_worker_subnet_name}

def _deploy_cluster(qbert, name, request_body):
    request_body['name'] = name
    log.info('Creating cluster %s', name)
    return qbert.create_cluster(request_body, 'v4')

def _get_cp_nodepool_uuid(qbert, cp_uuid):
    return next(cp['nodePoolUuid']
                for cp in qbert.list_cloud_providers()
                if cp['uuid'] == cp_uuid)
