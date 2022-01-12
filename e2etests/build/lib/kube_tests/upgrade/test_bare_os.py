
# Copyright (c) 2016 Platform9 systems. All rights reserved

import logging
import os
import time

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

        qbert_api_url = 'https://{0}/qbert'.format(self.du_fqdn)
        self.pf9_kube_role = '{0}-pmk.{1}'.format(os.getenv("KUBE_VERSION"), os.getenv("BUILD_NUMBER"))
        self.qbert = Qbert(self.du_fqdn, self.token, qbert_api_url, self.du_tenant_id, self.pf9_kube_role)
        self.container_runtime = os.getenv('CLUSTER_CREATE_CONTAINER_RUNTIME', default="docker")
        self.upgraded_runtime = os.getenv('CLUSTER_UPGRADE_CONTAINER_RUNTIME', default="docker")
        supported_roles = self.qbert.list_supported_roles()
        sem_ver = os.getenv("KUBE_VERSION").split('.')
        self.old_pf9_kube_role_patch = ''
        self.old_pf9_kube_role_minor = ''
        for role in supported_roles['roles']:
            # FIXME: When k8s major version is bumped,
            # e.g. 1.25 -> 2.1, we need to handle that
            # scenario as well.
            if (role['k8sMajorVersion'] == int(sem_ver[0]) and
                role['k8sMinorVersion'] == int(sem_ver[1])):

                self.old_pf9_kube_role_patch = role['roleVersion']

            if (role['k8sMajorVersion'] == int(sem_ver[0]) and
                role['k8sMinorVersion'] == int(sem_ver[1]) - 1):

                self.old_pf9_kube_role_minor = role['roleVersion']

    @test
    def test_create_cluster(self):
        kubeRoleVersions = []
        if (self.old_pf9_kube_role_patch != ''):
            kubeRoleVersions.append(self.old_pf9_kube_role_patch)

        if (self.old_pf9_kube_role_minor != ''):
            kubeRoleVersions.append(self.old_pf9_kube_role_minor)

        roleVersion = kubeRoleVersions[0]
        self.cluster_name = test_bare_os_cluster_create(self.qbert,
                                                        self._tb.hosts[:NUM_HOSTS],
                                                        self._tb.vip_port['ip'],
                                                        roleVersion,
                                                        num_masters = NUM_MASTERS)

        if (len(kubeRoleVersions) > 1):
            roleVersion = kubeRoleVersions[1]

        self.mlb_cluster_name = test_bare_os_cluster_create(self.qbert,
                                                        self._tb.hosts[NUM_HOSTS:],
                                                        self._tb.hosts[NUM_HOSTS]['ip'],
                                                        roleVersion,
                                                        num_masters = NUM_METALLB_MASTERS,
                                                        enable_metallb = True,
                                                        metallb_cidr = constants.METALLB_CIDR)

    @test(depends_on=[test_create_cluster])
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
    def test_pf9_kube_role_injection(self):
        self.qbert.inject_pf9_kube_version(self.qbert.pf9_kube_role)

    @test(depends_on=[test_pf9_kube_role_injection])
    @retry(log=log, max_wait=300, interval=5)
    def test_supported_roles(self):
        supported_roles = self.qbert.list_supported_roles()
        return next(item for item in supported_roles['roles'] if item['roleVersion'] == self.qbert.pf9_kube_role)

    @test(depends_on=[test_supported_roles])
    def test_cluster_upgrade(self):
        # Sleeping 60 seconds for sync loop to run and update
        # upgrade related fields for clusters.
        time.sleep(60)
        cluster = self.qbert.get_cluster(self.cluster_name)
        metallb_cluster = self.qbert.get_cluster(self.mlb_cluster_name)
        cuuids = [cluster['uuid'], metallb_cluster['uuid']]

        wait_for_cluster_taskstatus('success', self.qbert, cuuids)
        wait_for_cluster_status('ok', self.qbert, cuuids)

        if (cluster['canPatchUpgrade'] == 1 and
            cluster['patchUpgradeRoleVersion'] == self.pf9_kube_role):
            self.qbert.patch_upgrade_cluster(cluster['uuid'], runtime=self.upgraded_container_runtime)
        elif (cluster['canMinorUpgrade'] == 1 and
                 cluster['minorUpgradeRoleVersion'] == self.pf9_kube_role):
            self.qbert.minor_upgrade_cluster(cluster['uuid'], runtime=self.upgraded_container_runtime)

        if (metallb_cluster['canPatchUpgrade'] == 1 and
            metallb_cluster['patchUpgradeRoleVersion'] == self.pf9_kube_role):
            self.qbert.patch_upgrade_cluster(metallb_cluster['uuid'], runtime=self.upgraded_container_runtime)
        elif (metallb_cluster['canMinorUpgrade'] == 1 and
                 metallb_cluster['minorUpgradeRoleVersion'] == self.pf9_kube_role):
            self.qbert.minor_upgrade_cluster(metallb_cluster['uuid'], runtime=self.upgraded_container_runtime)

        wait_for_cluster_attr(self.qbert, cuuids, 'taskStatus', 'upgrading')
        wait_for_cluster_taskstatus('success', self.qbert, cuuids)
        wait_for_cluster_status('ok', self.qbert, cuuids)

    @test(depends_on=[test_cluster_upgrade])
    def test_cluster(self):
        cluster = self.qbert.get_cluster(self.cluster_name)
        self.test_k8s(cluster['uuid'])
        metallb_cluster = self.qbert.get_cluster(self.mlb_cluster_name)
        self.test_k8s(metallb_cluster['uuid'])
        self.test_metallb(metallb_cluster['uuid'])

    @test(depends_on=[test_cluster])
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

    def test_k8s(self, uuid):
        test_k8s_api(self.qbert, [uuid], self.user, self.passwd, self.du_fqdn, self.token,
            expected_runtime=self.upgraded_runtime)
        test_k8s_examples(self.qbert, [uuid], self.user, self.passwd)
        test_k8s_rbac(self.qbert, [uuid], self.user, self.passwd, self.du_fqdn, self.token)

    def test_metallb(self, uuid):
        test_metallb(self.qbert, uuid, self.user, self.passwd)
        test_bare_os_cluster_update(self.qbert, uuid)
        test_etcdbackup_for_cluster(self.qbert, self.du_fqdn, uuid)
        test_metallb(self.qbert, uuid, self.user, self.passwd)

    @test(depends_on=[test_workload_exists])
    def test_cluster_delete(self):
        test_bare_os_cluster_delete(self.qbert, self._tb.hosts[:NUM_HOSTS],
                                    self.cluster_name, num_masters = NUM_MASTERS)
        test_bare_os_cluster_delete(self.qbert, self._tb.hosts[NUM_HOSTS:],
                                   self.mlb_cluster_name, num_masters = NUM_METALLB_MASTERS)
