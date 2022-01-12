# Copyright (c) 2016 Platform9 systems. All rights reserved

import json
import logging
import os

from kube_tests.lib import dict_utils
from kube_tests.lib import request_utils

log = logging.getLogger(__name__)


class Kubernetes(object):
    """
    Expose kubectl commands
    api_server: 'protocol://host:port/path'
    verify: 'path to ca cert', optional, see
      http://docs.python-requests.org/en/master/user/advanced/#ssl-cert-verification0
    client_cert: ('path to cert', 'path to key'), optional, see
      http://docs.python-requests.org/en/master/user/advanced/#client-side-certificates
    token: 'opaque string', optional, either a Keystone token or
      a 'credentials token' as defined by the Authentication Webhook
    headers: dict, optional, the request headers
    """
    def __init__(self, api_server, verify=False, client_cert=None,
                 token=None, headers=None):
        if not api_server:
            raise ValueError('need an API Server')
        if client_cert and token:
            raise ValueError('Must not use both token and client cert')
        if not (token or client_cert):
            raise ValueError('Must use token or client cert')

        self.api_url = api_server
        session = request_utils.session_with_retries(self.api_url)
        session.verify = verify
        if headers:
            session.headers = headers
        if 'content-type' not in session.headers:
            session.headers['content-type'] = 'application/json'
        if client_cert:
            session.clientCert = client_cert
        if token:
            session.headers['Authorization'] = 'Bearer %s' % token
        self.session = session

    def _make_req(self, endpoint, method='GET', body={}):
        return request_utils.make_req(self.session, self.api_url + endpoint,
                                      method, body)

    def get_version(self):
        log.info('Getting kubernetes version')
        endpoint = '/version'
        resp = self._make_req(endpoint)
        resp = self.session.get(self.api_url + endpoint)
        return resp.json()

    def get_all_pods(self, namespace):
        log.info('Getting all pods in namespace %s', namespace)
        endpoint = '/api/v1/namespaces/{0}/pods'.format(namespace)
        resp = self._make_req(endpoint)
        resp = self.session.get(self.api_url + endpoint)
        return resp.json()

    def get_all_nodes(self):
        log.info('Getting all nodes')
        resp = self._make_req('/api/v1/nodes')
        return resp.json()

    def list_nodes(self):
        # see /swagger-ui/#!/v1/listNamespacedNode
        log.info('Listing nodes')
        endpoint = '/api/v1/nodes'
        resp = self._make_req(endpoint)
        nodesList = resp.json()['items']
        return dict_utils.keyed_list_to_dict(nodesList, 'metadata.name')
    
    def verify_node_container_runtime(self, runtime):
        log.info("verifying the container runtime of all nodes matches %s", runtime)
        log.info('Listing nodes')
        endpoint = '/api/v1/nodes'
        resp = self._make_req(endpoint)
        all_nodes = resp.json()['items']
        # https://platform9.atlassian.net/browse/PMK-4111
        # Consider adding more checks to make sure correct container runtime is configured directly on the nodes.
        for node in all_nodes:
            if 'status' in node and 'nodeInfo' in node['status'] and 'containerRuntimeVersion' in node['status']['nodeInfo']:
                node_container_runtime_version = node['status']['nodeInfo']['containerRuntimeVersion']
                node_container_runtime = node_container_runtime_version.split(':')[0]
                if node_container_runtime != runtime:
                    return False
        return True

    def create_deployment(self, namespace, template):
        log.info('Creating deployment')
        endpoint = ('/apis/apps/v1/namespaces/{0}/deployments'.format(namespace))
        method = 'POST'
        resp = self._make_req(endpoint, method, body=template)
        return resp

    def get_deployment(self, namespace, name):
        log.info('Getting deployment')
        endpoint = ('/apis/apps/v1/namespaces/{0}/deployments/{1}'
                    .format(namespace, name))

        resp = self._make_req(endpoint)
        return resp

    def delete_deployment(self, namespace, name):
        log.info('Deleting deployment')
        endpoint = ('/apis/apps/v1/namespaces/{0}/deployments/{1}'.format(namespace, name))
        method = 'DELETE'
        resp = self._make_req(endpoint, method)
        return resp

    def create_replicationcontroller(self, namespace, template):
        # see /swagger-ui/#!/v1/createNamespacedReplicationController
        log.info('Creating replicationcontroller')
        endpoint = ('/api/v1/namespaces/{0}/replicationcontrollers'
                    .format(namespace))
        method = 'POST'
        resp = self._make_req(endpoint, method, body=template)
        return resp.json()

    def get_replicationcontroller(self, namespace, name):
        endpoint = '/api/v1/namespaces/{0}/replicationcontrollers/{1}'.format(
            namespace, name)
        resp = self._make_req(endpoint)
        return resp.json()

    def delete_replicationcontroller(self, namespace, name):
        # see /swagger-ui/#!/v1/deleteNamespacedReplicationController
        log.info('Deleting replicationcontroller')
        endpoint = ('/api/v1/namespaces/{0}/replicationcontrollers/{1}'
                    .format(namespace, name))
        method = 'DELETE'
        resp = self._make_req(endpoint, method)
        return resp.json()

    def scale_replicationcontroller(self, namespace, name, replicas):
        log.info('Scaling replicationcontroller to %d', replicas)
        endpoint = '/api/v1/namespaces/{0}/replicationcontrollers/{1}'.format(
            namespace, name)
        resp = self._make_req(endpoint)
        body = resp.json()
        body['spec']['replicas'] = replicas
        self._make_req(endpoint, 'PUT', body)

    def list_replicationcontrollers(self, namespace):
        # see /swagger-ui/#!/v1/listNamespacedReplicationController
        log.info('Listing replicationcontrollers')
        endpoint = ('/api/v1/namespaces/{0}/replicationcontrollers'
                    .format(namespace))
        method = 'GET'
        resp = self._make_req(endpoint, method)
        replicationcontrollersList = resp.json()['items']
        return dict_utils.keyed_list_to_dict(replicationcontrollersList,
                                             'metadata.name')

    def create_service(self, namespace, template):
        # see /swagger-ui/#!/v1/createNamespacedService
        log.info('Creating service')
        endpoint = '/api/v1/namespaces/{0}/services'.format(namespace)
        method = 'POST'
        resp = self._make_req(endpoint, method, body=template)
        return resp.json()

    def delete_service(self, namespace, name):
        # see /swagger-ui/#!/v1/deleteNamespacedService
        log.info('Deleting service')
        endpoint = ('/api/v1/namespaces/{0}/services/{1}'
                    .format(namespace, name))
        method = 'DELETE'
        resp = self._make_req(endpoint, method)
        return resp.json()

    def list_services(self, namespace):
        # see /swagger-ui/#!/v1/listNamespacedService
        log.info('Listing services')
        endpoint = '/api/v1/namespaces/{0}/services'.format(namespace)
        resp = self._make_req(endpoint)
        servicesList = resp.json()['items']
        return dict_utils.keyed_list_to_dict(servicesList, 'metadata.name')

    def list_cluster_roles(self):
        log.info('Listing cluster roles')
        endpoint = '/apis/rbac.authorization.k8s.io/v1alpha1/clusterroles'
        method = 'GET'
        resp = self._make_req(endpoint, method)
        cluster_roles_list = resp.json()['items']
        return cluster_roles_list

    def proxy_req_to_service(self, namespace, service, path, method, body={}):
        endpoint = ('/api/v1/proxy/namespaces/{0}/services/{1}/{2}'
                    .format(namespace, service, path))
        resp = self._make_req(endpoint, method, body)
        return resp.json()

    def get_proxy_client(self, namespace, service):
        return lambda path, method='GET', body={}: self.proxy_req_to_service(
            namespace, service, path, method, body)


if __name__ == '__main__':
    import sys
    logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)

    api_server = os.getenv('APISERVER')
    externalDnsName = os.getenv('EXTERNALDNSNAME')
    token = os.getenv('TOKEN')
    client_cert = os.getenv('CLIENTCERT')
    verify = os.getenv('VERIFY')
    rawHeaders = os.getenv('HEADERS')

    headers = None
    if rawHeaders:
        headers = json.loads(rawHeaders)

    k8s = Kubernetes(api_server=api_server,
                     verify=verify,
                     client_cert=client_cert,
                     token=token,
                     headers=headers)
    print(k8s.list_services('default'))
