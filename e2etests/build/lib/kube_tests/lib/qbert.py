# Copyright (c) Platform9 systems. All rights reserved

import logging
import os
import re

from kube_tests.lib import dict_utils
from kube_tests.lib import request_utils

from pf9lab.utils import run_command_until_success, typical_du_fabric_settings

log = logging.getLogger(__name__)

class Qbert(object):
    def __init__(self, du_fqdn, token, api_url, tenant_id, pf9_kube_role=''):
        if not (token and api_url):
            raise ValueError('need a keystone token and API url')
        if api_url[-1] == '/':
            raise ValueError('API url must not have trailing slash')
        if not tenant_id:
            raise ValueError('need tenant id')
        if not du_fqdn:
            raise ValueError('need DU FQDN')
        self.api_url = api_url
        self.du_fqdn = du_fqdn
        self.pf9_kube_role = pf9_kube_role
        self.tenant_id = tenant_id
        session = request_utils.session_with_retries(self.api_url)
        session.headers = {'X-Auth-Token': token,
                           'Content-Type': 'application/json'}
        self.session = session
        self.subnet_shareable = False

    def _make_req(self, endpoint, method='GET', body={}):
        return request_utils.make_req(self.session, self.api_url + endpoint,
                                      method, body)

    def get_cloud_provider_regions(self, uuid, version='v2'):
        log.info('Getting cloud provider region info for: %s', uuid)
        endpoint = '/{0}/{1}/cloudProviders/{2}'.format(version, self.tenant_id, uuid)
        resp = self._make_req(endpoint)
        return resp.json()

    def get_cloud_provider_region_info(self, uuid, region, version='v2'):
        log.info('Getting cloud provider region info for: %s', uuid)
        endpoint = '/{0}/{1}/cloudProviders/{2}/region/{3}'.format(version, self.tenant_id, uuid, region)
        resp = self._make_req(endpoint)
        return resp.json()

    def delete_cloud_provider(self, uuid, version='v2'):
        endpoint = '/{0}/{1}/cloudProviders/{2}'.format(version, self.tenant_id, uuid)
        method = 'DELETE'
        self._make_req(endpoint, method)

    def create_cloud_provider(self, request_body, version='v2'):
        endpoint = '/{0}/{1}/cloudProviders'.format(version, self.tenant_id)
        method = 'POST'
        resp = self._make_req(endpoint, method, request_body)
        uuid = resp.json()['uuid']
        return uuid

    def update_cloud_provider(self, uuid, request_body, version='v2'):
        """
        :param uuid: of the cloud provider
        :param request_body: JSON with cloud provider-specific fields to update
               'name' and 'type' are required fields regardless of cloud provider
        :return:
        """
        endpoint = '/{0}/{1}/cloudProviders/{2}'.format(version, self.tenant_id, uuid)
        method = 'PUT'
        self._make_req(endpoint, method, request_body)

    def list_cloud_providers(self, version='v2'):
        log.info('Listing Cloud Providers')
        endpoint = '/{0}/{1}/cloudProviders'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return resp.json()

    def list_cloud_provider_types(self, version='v2'):
        log.info('Listing Cloud Provider Types')
        endpoint = '/{0}/{1}/cloudProvider/types'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return resp.json()

    def list_nodepools(self, version='v2'):
        log.info('Listing node pools')
        endpoint = '/{0}/{1}/nodePools'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return dict_utils.keyed_list_to_dict(resp.json(), 'name')

    def list_nodes(self, version='v2'):
        log.info('Listing nodes')
        endpoint = '/{0}/{1}/nodes'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return dict_utils.keyed_list_to_dict(resp.json(), 'name')

    def list_nodes_by_uuid(self, version='v2'):
        log.info('Listing nodes')
        endpoint = '/{0}/{1}/nodes'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return dict_utils.keyed_list_to_dict(resp.json(), 'uuid')

    def list_clusters(self, version='v2'):
        log.info('Listing clusters')
        endpoint = '/{0}/{1}/clusters'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return dict_utils.keyed_list_to_dict(resp.json(), 'name')

    def list_clusters_by_uuid(self, version='v2'):
        log.info('Listing clusters')
        endpoint = '/{0}/{1}/clusters'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return dict_utils.keyed_list_to_dict(resp.json(), 'uuid')

    def update_cluster(self, uuid, body, version='v2'):
        log.info('Updating cluster: %s', uuid)
        endpoint = '/{0}/{1}/clusters/{2}'.format(version, self.tenant_id, uuid)
        method = 'PUT'
        self._make_req(endpoint, method, body)

    def create_cluster(self, body, version='v2'):
        log.info('Creating cluster %s', body['name'])
        endpoint = '/{0}/{1}/clusters'.format(version, self.tenant_id)
        method = 'POST'
        resp = self._make_req(endpoint, method, body)
        uuid = resp.json()['uuid']
        return uuid

    def get_cluster_by_uuid(self, uuid, version='v2'):
        log.info('Get cluster')
        endpoint = '/{0}/{1}/clusters/{2}'.format(version, self.tenant_id, uuid)
        resp = self._make_req(endpoint)
        return resp.json()

    def delete_cluster_by_uuid(self, uuid, version='v2'):
        log.info('Deleting cluster %s', uuid)
        endpoint = '/{0}/{1}/clusters/{2}'.format(version, self.tenant_id, uuid)
        method = 'DELETE'
        self._make_req(endpoint, method)

    def delete_cluster(self, name, version='v2'):
        log.info('Deleting cluster %s', name)
        clusteruuid = self.list_clusters()[name]['uuid']
        endpoint = '/{0}/{1}/clusters/{2}'.format(version, self.tenant_id, clusteruuid)
        method = 'DELETE'
        self._make_req(endpoint, method)

    def attach_nodes(self, nodes_list, cluster_name, version='v2'):
        log.info('Attaching nodes %s to cluster %s', nodes_list, cluster_name)
        nodes = self.list_nodes()
        node_uuids = [{'uuid': nodes[node_item['nodeName']]['uuid'], 'isMaster': node_item['isMaster']} for node_item in nodes_list]
        cluster_uuid = self.list_clusters()[cluster_name]['uuid']
        endpoint = '/{0}/{1}/clusters/{2}/attach'.format(version, self.tenant_id, cluster_uuid)
        method = 'POST'
        body = node_uuids
        self._make_req(endpoint, method, body)

    def detach_node(self, nodeName, clusterName, version='v2'):
        log.info('Detaching node %s from cluster %s', nodeName, clusterName)
        nodeUuid = [{'uuid': self.list_nodes()[nodeName]['uuid']}]
        cluster_uuid = self.list_clusters()[clusterName]['uuid']
        endpoint = '/{0}/{1}/clusters/{2}/detach'.format(version, self.tenant_id, cluster_uuid)
        method = 'POST'
        body = nodeUuid
        self._make_req(endpoint, method, body)

    def attach_nodes_v2_upgrade(self, node_names, cluster_name, version='v2'):
        log.info('Attaching nodes %s to cluster %s', node_names, cluster_name)
        nodes = self.list_nodes()
        node_uuids = [nodes[node_name]['uuid'] for node_name in node_names]
        cluster_uuid = self.list_clusters()[cluster_name]['uuid']
        endpoint = '/{0}/{1}/clusters/{2}/attach'.format(version, self.tenant_id, cluster_uuid)
        method = 'POST'
        body = node_uuids
        self._make_req(endpoint, method, body)

    def detach_node_v2_upgrade(self, nodeName, clusterName, version='v2'):
        log.info('Detaching node %s from cluster %s', nodeName, clusterName)
        nodeUuid = self.list_nodes()[nodeName]['uuid']
        endpoint = '/{0}/{1}/nodes/{2}'.format(version, self.tenant_id, nodeUuid)
        method = 'PUT'
        body = {'clusterUuid': None}
        self._make_req(endpoint, method, body)

    def list_supported_roles(self, version='v4'):
        log.info('Listing supported roles')
        endpoint = '/{0}/{1}/clusters/supportedRoleVersions'.format(version, self.tenant_id)
        resp = self._make_req(endpoint)
        return resp.json()

    def get_cluster(self, name, version='v2'):
        log.info('Getting cluster %s', name)
        clusters = self.list_clusters()
        log.info('list_clusters output: %s', clusters)
        clusteruuid = clusters[name]['uuid']
        endpoint = '/{0}/{1}/clusters/{2}'.format(version, self.tenant_id, clusteruuid)
        resp = self._make_req(endpoint)
        return resp.json()

    def get_masterIp(self, clusterName, version='v2'):
        log.info('Getting masterIp for cluster %s', clusterName)
        return self.get_cluster(clusterName, version)['masterIp']

    def get_kubeconfig(self, clusterName, version='v2'):
        log.info('Getting kubeconfig for cluster %s', clusterName)
        clusterUuid = self.list_clusters()[clusterName]['uuid']
        return self.get_kubeconfig_using_uuid(clusterUuid, version)

    def get_kubeconfig_using_uuid(self, clusterUuid, version='v2'):
        endpoint = '/{0}/{1}/kubeconfig/{2}'.format(version, self.tenant_id, clusterUuid)
        resp = self._make_req(endpoint)
        return resp.text

    def get_kubelog(self, nodeName, version='v2'):
        log.info('Requesting kube.log from node %s', nodeName)
        nodeUuid = self.list_nodes()[nodeName]['uuid']
        endpoint = '/{0}/{1}/logs/{2}'.format(version, self.tenant_id, nodeUuid)
        resp = self._make_req(endpoint)
        return resp.text

    def trigger_omniupgrade(self, version='v2'):
        log.info('Triggering omniupgrade')
        endpoint = '/{0}/{1}/omniupgrade'.format(version, self.tenant_id)
        method = 'POST'
        return self._make_req(endpoint, method)

    def upgrade_cluster(self, uuid, version='v2', runtime="docker"):
        log.info('Upgrading cluster %s', uuid)
        endpoint = '/{0}/{1}/clusters/{2}/upgrade'.format(version, self.tenant_id, uuid)
        method = 'POST'
        body = {
            'batchUpgradePercent': 100,
            'containerRuntime': runtime
        }
        return self._make_req(endpoint, method, body)

    def minor_upgrade_cluster(self, uuid, version='v4', runtime="docker"):
        log.info('Minor upgrading cluster %s', uuid)
        endpoint = '/{0}/{1}/clusters/{2}/upgrade?type=minor'.format(version, self.tenant_id, uuid)
        method = 'POST'
        body = {
            'batchUpgradePercent': 100,
            'containerRuntime': runtime
        }
        return self._make_req(endpoint, method, body)

    def patch_upgrade_cluster(self, uuid, version='v4', runtime="docker"):
        log.info('Minor upgrading cluster %s', uuid)
        endpoint = '/{0}/{1}/clusters/{2}/upgrade?type=patch'.format(version, self.tenant_id, uuid)
        method = 'POST'
        body = {
            'batchUpgradePercent': 100,
            'containerRuntime': runtime
        }
        return self._make_req(endpoint, method, body)

    def inject_pf9_kube_version(self, kube_version):
        if not kube_version:
            raise ValueError('need pf9-kube role version')

        du_commands = ["mkdir -p /opt/pf9/qbert/supportedRoleVersions",
                       'touch /opt/pf9/qbert/supportedRoleVersions/{0}'.format(kube_version),
                       "systemctl restart pf9-qbert"
                      ]
        with typical_du_fabric_settings(self.du_fqdn):
            for cmd in du_commands:
                run_command_until_success(cmd)

    def restart_qbert(self):
        log.info('Restarting qbert server')
        du_commands = ["systemctl restart pf9-qbert"]
        with typical_du_fabric_settings(self.du_fqdn):
            for cmd in du_commands:
                run_command_until_success(cmd)

if __name__ == '__main__':
    import sys
    ch = logging.StreamHandler(sys.stdout)
    log.addHandler(ch)
    log.setLevel(logging.DEBUG)
    token = os.getenv('TOKEN')
    du_fqdn = os.getenv('DU_FQDN')
    api_url = os.getenv('QBERT_API_URL')
    q = Qbert(du_fqdn=du_fqdn , token=token, api_url=api_url, tenant_id='')
    create_cluster_params = {
        'name': 'qbert-py-test',
        'nodePool??': 'defaultPool',
        'containersCidr': '10.70.0.0/16',
        'servicesCidr': '10.90.0.0/16'
    }
    q.create_cluster(create_cluster_params)
    q.delete_cluster('test')
