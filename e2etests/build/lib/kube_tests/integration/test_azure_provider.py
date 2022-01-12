# Copyright (c) 2018 Platform9 systems. All rights reserved

import logging
import os
import unittest

from proboscis import test, before_class
from uuid import uuid4
from Crypto.PublicKey import RSA
from distutils.util import strtobool

from kube_tests.integration.common.test_kubernetes import test_k8s_api, test_k8s_examples, test_k8s_rbac, test_k8s_autoscaler
from kube_tests.integration.common.test_web_cli import test_web_cli
from kube_tests.integration.common.test_azure_provider_create import test_azure_provider_create
from kube_tests.integration.common.test_azure_provider_delete import test_azure_provider_delete
from kube_tests.integration.common.test_azure_cluster_create import test_azure_cluster_create
from kube_tests.integration.common.test_azure_cluster_update import test_azure_cluster_update
from kube_tests.integration.common.test_azure_cluster_delete import test_azure_cluster_delete
from kube_tests.integration.common import ensure_env_set, EnvTuple
from integration.test_util import BaseTestCase
from pf9lab.du.auth import login
from pf9lab.retry import retry
from pf9lab.testbeds.loader2 import load_testbed
from kube_tests.lib.qbert import Qbert
from kube_tests.testbeds.cloud_du_testbed import CloudDuTestbed

log = logging.getLogger(__name__)

AZURE_LOCATIONS = ['eastus', 'westus', 'westus2']

def _to_bool(v):
    return bool(strtobool(v))

class AzureTestbedProfile:
    def __init__(self):
        self.id = uuid4()
        self.public_key = self._generate_key_pair(self.id)

    def _generate_key_pair(self, uuid):
        from stat import S_IREAD
        public_key = None
        key = RSA.generate(2048)
        # pathing needed to allow the private key for azure tests to be published as an artifact on teamcity
        test_suite_results_dir = os.getenv("SUITE_RESULTS_DIR", "../build/testing/qbert_azure_ubuntu18")
        path_to_private_key = os.path.join(test_suite_results_dir, 'pmk_azure_tests_{}'.format(uuid))
        with open(path_to_private_key, 'wb') as content_file:
            os.chmod(path_to_private_key, S_IREAD) # mode 600
            content_file.write(key.exportKey('PEM'))
        public_key = key.publickey().exportKey('OpenSSH')
        return public_key

    def get_cloud_provider_create_input(self):
        return {
            'clientId'         : os.environ['AZURE_TESTBED_CLOUD_PROVIDER_CLIENT_ID'],
            'clientSecret'     : os.environ['AZURE_TESTBED_CLOUD_PROVIDER_CLIENT_SECRET'],
            'name'             : 'pmk-azure-{}'.format(self.id),
            'subscriptionId'   : os.environ['AZURE_TESTBED_CLOUD_PROVIDER_SUBSCRIPTION_ID'],
            'tenantId'         : os.environ['AZURE_TESTBED_CLOUD_PROVIDER_TENANT_ID'],
            'type'             : 'azure'
        }

    def get_cluster_create_input(self):
        # ssh_key_placeholder = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC1Ettja5AJ3sjGTeMcvsgHdgdyAGgG8V0bqRF9Dq1SlyYfRvuo2PlXfY+Ft/KmdIxCv38wZ4KhyNLqHBhz965oC7AYLz1U5w4LgdtVG/BrtxM8B3rsYn8bHItjx3cbba+8dy3mpoEhyMdMHlD0SqjSw8fJ4dNJspka5i7NiRMrM7eUBRhQZexTi/lPvIKfJDNjlT7XJEZ2TgpRZn30uWVeNddMiVWLjlBnufHESzz07JY8XPbvaBA0kTIJO8G6F3XQk8R7T5IAAE+V2QckDjF1XkHtqly3QKXyDQi6uTPU+JbXbWI5Z6LGmAh+Ar2iQ0eh7eReIzdUwpPu3iDNy5Mv9hLRZ0tqe+TEdkS6+pn8swTBGfAXpk2E8HxfMhijkj+idw/sbXxTG0TcN+qJdkdgJGl9r3V0VeZnfJQwok5MxtM9EYggemZM8DLjjYWijrlFP0fA8Eq4vqunqpoWLXJDzDZvrc8zvJN1xnfjgpgJw3UqRhdYXLNV2BYkh6wfGDLHUq4bjsnjA27hQB1vdPLwVJqFpM0knzTR3OxZ4N8P/Jvlr/Kk6qPeD1QryzY/0yWk71abenEOmzUMB4y1g/5tASuhVvWDd9udY8Rgtifymv4iCDkz9RVFOboLoRFNXyjAXjH4EuQ4OsXujs2+UnOrjQUFQEj2iupjhT37iyVzuw== josh@platform9.com"
        return {
            'appCatalogEnabled': _to_bool(os.getenv('AZURE_TESTBED_CLUSTER_APP_CATALOG_ENABLED', 'True')),
            'assignPublicIps': _to_bool(os.getenv('AZURE_TESTBED_CLUSTER_ASSIGN_PUBLIC_IPS', 'True')),
            'containersCidr': '10.20.0.0/16',  # Leave these hardcoded or pull from constants.py
            'debug': os.getenv('AZURE_TESTBED_CLUSTER_DEBUG', 'true'),
            'internalLb': _to_bool(os.getenv('AZURE_TESTBED_CLUSTER_INTERNAL_LB', 'True')),
            'location': os.environ['AZURE_TESTBED_CLUSTER_LOCATION'],
            'masterSku': os.environ['AZURE_TESTBED_CLUSTER_MASTER_SKU'],
            'nodePoolUuid': None,  # Set from output of GET cloudProviders -> type: azure
            'numMasters': os.environ['AZURE_TESTBED_CLUSTER_NUM_MASTERS'],   # Set via env var
            'numWorkers': os.environ['AZURE_TESTBED_CLUSTER_NUM_WORKERS'],   # Set via env var
            'operatingSystem': os.environ['AZURE_TESTBED_CLUSTER_OPERATING_SYSTEM'],
            'servicesCidr': '10.40.0.0/16',  # Leave these hardcoded or pull from constants.py
            'sshKey': os.getenv('AZURE_TESTBED_CLUSTER_SSH_KEY', self.public_key),  # TODO: GENERATE KEY-PAIR or something
            'workerSku': os.environ['AZURE_TESTBED_CLUSTER_WORKER_SKU'],
            'networkPlugin': 'flannel',
            'allowWorkloadsOnMaster': _to_bool(os.environ['ALLOW_WORKLOADS_ON_MASTER'])
            #'zones': [1, 2, 3],  # TODO: make this configurable?
        }


@test(groups=['integration'])
class TestAzureCloudProvider(BaseTestCase):
    """
        1. Tests cloud provider CRUD
        2. Tests cluster auto deploy for Azure cloud provider
        3. Tests basic Azure resource cleanup
    """

    def __init__(self):
        # TODO: Remove some of these envs as being required once the cluster create calls
        # use output from cloud provider region details as input
        required_env_vars = [
            EnvTuple('AZURE_TESTBED_CLUSTER_MASTER_SKU'),
            EnvTuple('AZURE_TESTBED_CLUSTER_WORKER_SKU'),
            EnvTuple('AZURE_TESTBED_CLUSTER_NUM_MASTERS'),
            EnvTuple('AZURE_TESTBED_CLUSTER_NUM_WORKERS'),
            EnvTuple('AZURE_TESTBED_CLUSTER_LOCATION', possible_values=AZURE_LOCATIONS),
        ]
        for env in required_env_vars:
            ensure_env_set(env)

    @before_class
    def setUp(self):
        log.info('In azure provider test setUp')

        testbed_file = os.getenv('TESTBED')
        self.assertTrue(testbed_file)
        self._tb = load_testbed(testbed_file)

        # Creates a DU on cloud.platform9.net AFAIK :)
        self.assertTrue(isinstance(self._tb, CloudDuTestbed))
        self.du_ip = self._tb.du_fqdn()
        self.du_tenant_id = self._tb.du_tenant_id()

        auth_info = login('https://%s' % self.du_ip,
                          self._tb.du_user(),
                          self._tb.du_pass(),
                          'service')
        self.token = auth_info['access']['token']['id']
        qbert_api_url = 'https://{0}/qbert'.format(self.du_ip)
        pf9_kube_role = '{0}-pmk.{1}'.format(os.getenv("KUBE_VERSION"), os.getenv("BUILD_NUMBER"))
        self.qbert = Qbert(self.du_ip, self.token, qbert_api_url, self.du_tenant_id, pf9_kube_role)

        log.info("Loading testbed profile")
        self.azure_testbed_profile = AzureTestbedProfile()

    @test
    def test_cloud_provider_create(self):
        self.cp_uuid = test_azure_provider_create(self.qbert, self.azure_testbed_profile)

    @test(depends_on=[test_cloud_provider_create])
    def test_pf9_kube_role_injection(self):
        self.qbert.inject_pf9_kube_version(self.qbert.pf9_kube_role)

    @test(depends_on=[test_pf9_kube_role_injection])
    @retry(log=log, max_wait=300, interval=5)
    def test_supported_roles(self):
        supported_roles = self.qbert.list_supported_roles()
        return next(item for item in supported_roles['roles'] if item['roleVersion'] == self.qbert.pf9_kube_role)

    @test(depends_on=[test_supported_roles])
    def test_cluster_create(self):
        self.cluster_uuids = test_azure_cluster_create(
            self.qbert, self.cp_uuid, self.azure_testbed_profile,
            self._tb.template_key, self.azure_testbed_profile.get_cluster_create_input()['assignPublicIps'],
             self._tb.runtime_config, self.qbert.pf9_kube_role)

    @test(depends_on=[test_cluster_create])
    def test_cluster_update(self):
        test_azure_cluster_update(self.qbert, self.cluster_uuids['azure_complete_uuid'])

    @test(depends_on=[test_cluster_update])
    def _test_kubernetes(self):
        keystone_user = self._tb.du_user()
        keystone_pass = self._tb.du_pass()
        test_k8s_api(self.qbert, list(self.cluster_uuids.values()),
                     keystone_user, keystone_pass,
                     self.du_ip, self.token)
        test_k8s_examples(self.qbert, list(self.cluster_uuids.values()),
                          keystone_user, keystone_pass,
                          cloud_provider_type='azure')
        test_k8s_rbac(self.qbert, list(self.cluster_uuids.values()),
                      keystone_user, keystone_pass,
                      self.du_ip, self.token,
                      cloud_provider_type='azure')
        # Disable cluster autoscaler till PMK-3172 is resolved
        #test_k8s_autoscaler(self.qbert, [self.cluster_uuids['azure_network_provided_uuid']],
        #                  keystone_user, keystone_pass,
        #                  cloud_provider_type='azure')

    @test(depends_on=[_test_kubernetes])
    def _test_web_cli(self):
        for cluster_uuid in list(self.cluster_uuids.values()):
            test_web_cli(self.qbert, cluster_uuid, self.du_ip)
    @test(depends_on=[_test_web_cli])
    def test_cloud_provider_update(self):
        return True
        # TODO: Add once https://platform9.atlassian.net/browse/PMK-1180 is DONE
        # test_azure_provider_update(
        #     self.qbert,
        #     self.region,
        #     self.cp_uuid)

    @test(depends_on=[test_cloud_provider_update])
    def test_cluster_delete(self):
        test_azure_cluster_delete(self.qbert, self.cluster_uuids)

    @test(depends_on=[test_cluster_delete])
    def test_cloud_provider_delete(self):
        test_azure_provider_delete(self.qbert, self.cp_uuid)



if __name__ == '__main__':
    unittest.main()
