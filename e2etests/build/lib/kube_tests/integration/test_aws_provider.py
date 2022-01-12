# Copyright (c) 2016 Platform9 systems. All rights reserved

import logging
import os
import unittest

import boto3
from proboscis import test, before_class

from kube_tests.integration.common.test_aws_cluster_create import test_aws_cluster_create
from kube_tests.integration.common.test_aws_cluster_delete import test_aws_cluster_delete
from kube_tests.integration.common.test_aws_cluster_update import test_aws_cluster_update
from kube_tests.integration.common.test_aws_node_cleanup import test_aws_node_cleanup
from kube_tests.integration.common.test_aws_provider_create import test_aws_provider_create
from kube_tests.integration.common.test_aws_provider_update import test_aws_provider_update
from kube_tests.integration.common.test_aws_provider_delete import test_aws_provider_delete
from kube_tests.integration.common.test_kubernetes import test_k8s_api, test_k8s_examples, test_k8s_rbac, test_k8s_autoscaler, test_network_policy
from kube_tests.integration.common import constants, wait_for_cluster_attr, wait_for_cluster_taskstatus, wait_for_cluster_status
from integration.test_util import BaseTestCase
from pf9lab.du.auth import login
from pf9lab.retry import retry
from pf9lab.testbeds.loader2 import load_testbed
from kube_tests.lib.qbert import Qbert
from kube_tests.testbeds.cloud_du_testbed import CloudDuTestbed
from kube_tests.lib.kubernetes import Kubernetes
from kube_tests.lib.kubeconfig import get_kubeconfig
import kube_tests.integration.common.test_workload as workload

log = logging.getLogger(__name__)


@test(groups=['integration'])
class TestAwsCloudProvider(BaseTestCase):
    """
        1. Tests cloud provider CRUD
        2. Tests cluster auto deploy for aws cloud provider
        3. Tests basic aws resource cleanup
    """

    @before_class
    def setUp(self):
        log.info('In aws provider test setUp')

        testbed_file = os.getenv('TESTBED')
        self.assertTrue(testbed_file)
        self._tb = load_testbed(testbed_file)
        self.assertTrue(isinstance(self._tb, CloudDuTestbed))
        self.du_ip = self._tb.du_fqdn()
        self.du_tenant_id = self._tb.du_tenant_id()

        auth_info = login('https://%s' % self.du_ip,
                          self._tb.du_user(),
                          self._tb.du_pass(),
                          'service')
        self.token = auth_info['access']['token']['id']
        qbert_api_url = 'https://{0}/qbert'.format(self.du_ip)
        self.pf9_kube_role = '{0}-pmk.{1}'.format(os.getenv("KUBE_VERSION"), os.getenv("BUILD_NUMBER"))
        self.qbert = Qbert(self.du_ip, self.token, qbert_api_url, self.du_tenant_id, self.pf9_kube_role)
        self.region = os.environ.get('AWS_REGION', 'us-west-1')
        self.container_runtime = os.getenv('CLUSTER_CREATE_CONTAINER_RUNTIME', default="docker")
        boto_creds = {
            'aws_access_key_id': os.getenv('HYBRID_ACCESS_KEY'),
            'aws_secret_access_key': os.getenv('HYBRID_ACCESS_SECRET'),
            'region_name': self.region
        }
        self.ec2 = boto3.resource('ec2', **boto_creds)
        # The low level client may be needed for operations
        # not supported in the resource model
        # See https://github.com/boto/boto3/issues/424
        self._ec2 = boto3.client('ec2', **boto_creds)
        self.mtu_size = 1200

    @test
    def test_cloud_provider_create(self):
        self.cp_uuid = test_aws_provider_create(self.qbert, self.region)

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
        self.cluster_uuids, self.subnets = test_aws_cluster_create(
            self.qbert, self.cp_uuid, self.region, self.ec2, self._ec2,
            self._tb.template_key, self._tb.is_private,
            self._tb.runtime_config, [self.qbert.pf9_kube_role], mtu_size=self.mtu_size,
            runtime=self.container_runtime)

    @test(depends_on=[test_cluster_create])
    def test_cluster_update(self):
        test_aws_cluster_update(self.qbert, self.cluster_uuids['aws_complete_uuid'])

    @test(depends_on=[test_cluster_update])
    def _test_kubernetes(self):
        keystone_user = self._tb.du_user()
        keystone_pass = self._tb.du_pass()
        test_k8s_api(self.qbert, list(self.cluster_uuids.values()),
                     keystone_user, keystone_pass,
                     self.du_ip, self.token,
                     expected_runtime=self.container_runtime)
        test_k8s_examples(self.qbert, list(self.cluster_uuids.values()),
                          keystone_user, keystone_pass,
                          cloud_provider_type='aws')
        test_k8s_rbac(self.qbert, list(self.cluster_uuids.values()),
                      keystone_user, keystone_pass,
                      self.du_ip, self.token,
                      cloud_provider_type='aws')
        test_network_policy(self.qbert, list(self.cluster_uuids.values()),
                            keystone_user, keystone_pass,
                            self.du_ip, self.token,
                            cloud_provider_type='aws')
        test_k8s_autoscaler(self.qbert, [self.cluster_uuids['aws_uuid_shared']],
                          keystone_user, keystone_pass,
                          cloud_provider_type='aws')

    @test(depends_on=[_test_kubernetes])
    def test_node_cleanup(self):
        test_aws_node_cleanup(self.qbert, self.cluster_uuids['aws_complete_uuid'], self.ec2)

    @test(depends_on=[test_node_cleanup])
    def test_cloud_provider_update(self):
        # updating the keys with same keys
        test_aws_provider_update(self.qbert, self.region, self.cp_uuid)

    @test(depends_on=[test_cloud_provider_update])
    def test_add_workload(self):
        clusters = self.qbert.list_clusters()
        for cluster in clusters:
            kubeconfig = get_kubeconfig(self.qbert,
                                        cluster,
                                        self._tb.du_user(),
                                        self._tb.du_pass())
            api_server = kubeconfig.cluster(cluster)['server']
            with kubeconfig.cluster_ca_file(cluster) as ca_file_path:
                kc_token = kubeconfig.user(self._tb.du_user())['token']
                k8s = Kubernetes(api_server=api_server, verify=ca_file_path,
                                  token=kc_token)
                workload.test_verify_workload_does_not_exist(k8s)
                workload.test_add_workload(k8s)
                workload.test_verify_workload_exists(k8s)

    @test(depends_on=[test_add_workload])
    def test_minor_upgrade(self):
        do_minor_upgrade = False
        cuuids = list(self.cluster_uuids.values())

        # Since all the clusters are deployed using same pf9-kube role version
        # all of them would be eligible for minor upgrade if canMinorUpgrade
        # is true for any one of these clusters
        cluster = self.qbert.get_cluster_by_uuid(cuuids[0])
        if cluster['canMinorUpgrade'] == 1:
            do_minor_upgrade = True

        wait_for_cluster_taskstatus('success', self.qbert, cuuids)
        wait_for_cluster_status('ok', self.qbert, cuuids)

        if do_minor_upgrade:
            for uuid in cuuids:
                self.qbert.minor_upgrade_cluster(uuid)

            wait_for_cluster_attr(self.qbert, cuuids, 'taskStatus', 'upgrading')
            wait_for_cluster_taskstatus('success', self.qbert, cuuids)
            wait_for_cluster_status('ok', self.qbert, cuuids)
            self._test_kubernetes_post_upgrade()

    def _test_kubernetes_post_upgrade(self):
        keystone_user = self._tb.du_user()
        keystone_pass = self._tb.du_pass()
        test_k8s_api(self.qbert, list(self.cluster_uuids.values()),
                     keystone_user, keystone_pass,
                     self.du_ip, self.token,
                     expected_runtime=self.container_runtime)
        test_k8s_examples(self.qbert, list(self.cluster_uuids.values()),
                          keystone_user, keystone_pass,
                          cloud_provider_type='aws')
        test_k8s_rbac(self.qbert, list(self.cluster_uuids.values()),
                      keystone_user, keystone_pass,
                      self.du_ip, self.token,
                      cloud_provider_type='aws')

    @test(depends_on=[test_minor_upgrade])
    def test_workload_exists(self):
        clusters = self.qbert.list_clusters()
        for cluster in clusters:
            kubeconfig = get_kubeconfig(self.qbert,
                                        cluster,
                                        self._tb.du_user(),
                                        self._tb.du_pass())
            api_server = kubeconfig.cluster(cluster)['server']
            with kubeconfig.cluster_ca_file(cluster) as ca_file_path:
                kc_token = kubeconfig.user(self._tb.du_user())['token']
                k8s = Kubernetes(api_server=api_server, verify=ca_file_path,
                                  token=kc_token)
                workload.test_verify_workload_exists(k8s)
                workload.test_delete_workload(k8s)
                workload.test_verify_workload_does_not_exist(k8s)


    @test(depends_on=[test_workload_exists])
    def test_cluster_delete(self):
        test_aws_cluster_delete(
            self.qbert, self.du_ip, self.cluster_uuids, self.subnets,
            self.cp_uuid, self.region)

    @test(depends_on=[test_cluster_delete])
    def test_cloud_provider_delete(self):
        test_aws_provider_delete(self.qbert, self.cp_uuid)

if __name__ == '__main__':
    unittest.main()
