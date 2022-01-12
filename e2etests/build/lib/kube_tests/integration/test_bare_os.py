
# Copyright (c) 2016 Platform9 systems. All rights reserved

import logging
import os

from proboscis import test, before_class, after_class

from kube_tests.integration.common.test_bare_os_cluster_create import test_bare_os_cluster_create
from kube_tests.integration.common.test_bare_os_cluster_update import test_bare_os_cluster_update
from kube_tests.integration.common.test_bare_os_cluster_delete import test_bare_os_cluster_delete
from kube_tests.integration.common.test_kubernetes import test_k8s_api, test_k8s_examples, test_k8s_rbac, test_metallb, run_command
from kube_tests.integration.common.test_kubernetes import test_etcdbackup_for_cluster
from integration.test_util import BaseTestCase
from pf9lab.du.auth import login
from pf9lab.retry import retry
from pf9lab.testbeds.loader2 import load_testbed
from kube_tests.lib.qbert import Qbert
from kube_tests.testbeds.raw_kube_testbed import RawKubeTestbed
from kube_tests.lib.kubernetes import Kubernetes
from kube_tests.lib.kubeconfig import get_kubeconfig
import kube_tests.integration.common.test_workload as workload
from kube_tests.integration.common import constants, wait_for_cluster_attr, wait_for_cluster_taskstatus, wait_for_cluster_status

GET_SANS_CMD = '''\\
echo -n | \\
openssl s_client -connect {0}:443 2>/dev/null | \\
sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' | \\
openssl x509 -text -noout | \\
grep 'Subject Alternative Name' -A 1'''

# Test config
NUM_MASTERS = 3
NUM_WORKERS = 2
NUM_HOSTS = NUM_MASTERS + NUM_WORKERS
NUM_METALLB_MASTERS = 1
NUM_METALLB_WORKERS = 1
NUM_METALLB_HOSTS = NUM_METALLB_MASTERS + NUM_METALLB_WORKERS

log = logging.getLogger(__name__)


@test(groups=['integration'])
class TestQbert(BaseTestCase):

    @before_class
    def setUp(self):
        log.info('[QBERT-TEST-SETUP] In qbert test setUp')
        testbed_file = os.getenv('TESTBED')
        self.assertTrue(testbed_file)
        self._tb = load_testbed(testbed_file)
        self.assertTrue(isinstance(self._tb, RawKubeTestbed))

        self.du_fqdn = self._tb.du_fqdn()
        self.user = self._tb.du_user()
        self.passwd = self._tb.du_pass()
        self.du_tenant_id = self._tb.du_tenant_id()

        auth_info = login('https://%s' % self.du_fqdn,
                          self.user,
                          self.passwd,
                          'service')
        self.token = auth_info['access']['token']['id']
        self.container_runtime = os.getenv('CLUSTER_CREATE_CONTAINER_RUNTIME', default="docker")
        qbert_api_url = 'https://{0}/qbert'.format(self.du_fqdn)
        self.pf9_kube_role = '{0}-pmk.{1}'.format(os.getenv("KUBE_VERSION"), os.getenv("BUILD_NUMBER"))
        self.qbert = Qbert(self.du_fqdn, self.token, qbert_api_url, self.du_tenant_id, self.pf9_kube_role)

    @test
    def test_pf9_kube_role_injection(self):
        self.qbert.inject_pf9_kube_version(self.qbert.pf9_kube_role)

    @test(depends_on=[test_pf9_kube_role_injection])
    @retry(log=log, max_wait=300, interval=5)
    def test_supported_roles(self):
        supported_roles = self.qbert.list_supported_roles()
        return next(item for item in supported_roles['roles'] if item['roleVersion'] == self.qbert.pf9_kube_role)

    @test(depends_on=[test_supported_roles])
    def test_create_cluster(self):
        self.cluster_name = test_bare_os_cluster_create(self.qbert,
                                                        self._tb.hosts[:NUM_HOSTS],
                                                        self._tb.vip_port['ip'],
                                                        self.qbert.pf9_kube_role,
                                                        num_masters = NUM_MASTERS,
                                                        runtime=self.container_runtime)
        self.mlb_cluster_name = test_bare_os_cluster_create(self.qbert,
                                                        self._tb.hosts[NUM_HOSTS:],
                                                        self._tb.hosts[NUM_HOSTS]['ip'],
                                                        self.qbert.pf9_kube_role,
                                                        num_masters = NUM_METALLB_MASTERS,
                                                        enable_metallb = True,
                                                        metallb_cidr = constants.METALLB_CIDR,
                                                        runtime=self.container_runtime)

    @test(depends_on=[test_create_cluster])
    def test_cluster(self):
        cluster = self.qbert.get_cluster(self.cluster_name)

        # Verify master IP address in the the server SANs
        cmd = GET_SANS_CMD.format(cluster['externalDnsName'])
        return_code, output = run_command(cmd)
        self.assertEqual(return_code, 0)
        master_ip_san = 'IP Address:{0}'.format(cluster['masterIp'])
        self.assertTrue(master_ip_san in output)
        self.test_k8s(cluster['uuid'])

        # MetalLB tests
        cluster = self.qbert.get_cluster(self.mlb_cluster_name)
        self.test_metallb(cluster['uuid'])

    @test(depends_on=[test_cluster])
    def test_add_workload(self):
        clusters = self.qbert.list_clusters()
        for cluster in clusters:
            kubeconfig = get_kubeconfig(self.qbert,
                                        cluster,
                                        self.user,
                                        self.passwd)
            api_server = kubeconfig.cluster(cluster)['server']
            with kubeconfig.cluster_ca_file(cluster) as ca_file_path:
                kc_token = kubeconfig.user(self.user)['token']
                k8s = Kubernetes(api_server=api_server, verify=ca_file_path,
                                  token=kc_token)
                workload.test_verify_workload_does_not_exist(k8s)
                workload.test_add_workload(k8s)
                workload.test_verify_workload_exists(k8s)

    @test(depends_on=[test_add_workload])
    def test_minor_upgrade(self):
        do_minor_upgrade = False
        cluster = self.qbert.get_cluster(self.cluster_name)
        metallb_cluster = self.qbert.get_cluster(self.mlb_cluster_name)
        cuuids = [cluster['uuid'], metallb_cluster['uuid']]

        # Since both clusters are deployed using same pf9-kube role version
        # both of them would be eligible for minor upgrade if canMinorUpgrade
        # is true for any one of these clusters
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

            self.test_k8s(cluster['uuid'])
            self.test_metallb(metallb_cluster['uuid'])

    def test_k8s(self, uuid):
        test_k8s_api(self.qbert, [uuid], self.user, self.passwd, self.du_fqdn, self.token,
            expected_runtime=self.container_runtime)
        test_k8s_examples(self.qbert, [uuid], self.user, self.passwd)
        test_k8s_rbac(self.qbert, [uuid], self.user, self.passwd, self.du_fqdn, self.token)

    def test_metallb(self, uuid):
        test_metallb(self.qbert, uuid, self.user, self.passwd)
        test_bare_os_cluster_update(self.qbert, uuid)
        test_etcdbackup_for_cluster(self.qbert, self.du_fqdn, uuid)
        test_metallb(self.qbert, uuid, self.user, self.passwd)

    @test(depends_on=[test_minor_upgrade])
    def test_workload_exists(self):
        clusters = self.qbert.list_clusters()
        for cluster in clusters:
            kubeconfig = get_kubeconfig(self.qbert,
                                        cluster,
                                        self.user,
                                        self.passwd)
            api_server = kubeconfig.cluster(cluster)['server']
            with kubeconfig.cluster_ca_file(cluster) as ca_file_path:
                kc_token = kubeconfig.user(self.user)['token']
                k8s = Kubernetes(api_server=api_server, verify=ca_file_path,
                                  token=kc_token)
                workload.test_verify_workload_exists(k8s)
                workload.test_delete_workload(k8s)
                workload.test_verify_workload_does_not_exist(k8s)

    @test(depends_on=[test_workload_exists])
    def test_cluster_delete(self):
        test_bare_os_cluster_delete(self.qbert, self._tb.hosts[:NUM_HOSTS],
                                    self.cluster_name, num_masters = NUM_MASTERS)
        test_bare_os_cluster_delete(self.qbert, self._tb.hosts[NUM_HOSTS:],
                                   self.mlb_cluster_name, num_masters = NUM_METALLB_MASTERS)
