import subprocess
import requests
import logging
import json
import uuid
import tempfile
import yaml
from pf9lab.keystonev3 import create_group
from pf9lab.keystone import create_user
from pf9lab.keystone import tenant_name_to_id
from pf9lab.keystone import role_name_to_id
from pf9lab.keystone import add_role_to_user

from kube_tests.lib.kubeconfig import get_kubeconfig
from kube_tests.lib.qbert import Qbert
from kube_tests.lib.command_utils import run_command
from pf9lab.du.auth import login

log = logging.getLogger(__name__)

#Main class for RBAC related tests
class Rbac(object):

    def __init__(self, kc, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, kubectl):
        self.admin_kc = kc
        self.du_fqdn = du_fqdn
        self.ks_token = ks_token
        self.api_server = api_server
        self.cloud_provider_type = cloud_provider_type
        self.kubectl = kubectl
        self.headers= {'Content-Type': 'application/json', 'X-Auth-Token' : self.ks_token}
        self.ks_url = ''.join(['https://',self.du_fqdn, '/keystone/v3/'])
        self.qbert = qbert
        self.cluster = cluster

    #Sets up a new user in keystone
    #Used as unprivileged user in subsequent tests
    def setup_ks_user(self):
        r = create_user(self.du_fqdn, self.ks_token, "testuser" + str(uuid.uuid4()), "testuser")
        self.ks_user = r['user']

    #Sets up a new group in keystone, adds above user to this group
    #Used to test assigning permissions to a group
    def setup_ks_group(self):
        r = create_group(self.du_fqdn, self.ks_token, "testgroup" + str(uuid.uuid4()))
        self.ks_group = r['group']

        #Add user to the group
        #Calling keystone directly because this is a special usecase where we want to
        #add a user to a group without giving the user any role on any project
        url =  ('%s/groups/%s/users/%s' % (self.ks_url, self.ks_group['id'], self.ks_user['id']))
        requests.put(url, data=None, verify=False, headers=self.headers)

    #Create role binding base yaml as dict
    def setup_base_yaml(self):
        self.role_binding = dict(
            kind = 'ClusterRoleBinding',
            apiVersion = 'rbac.authorization.k8s.io/v1',
            metadata = dict(
                name = 'role-binding' + str(uuid.uuid4())
            ),
            subjects = [
                dict(
                    kind = 'User',
                    name = 'testuser',
                )
            ],
            roleRef = dict(
                kind = 'ClusterRole',
                name = 'cluster-admin',
                apiGroup = 'rbac.authorization.k8s.io'
            )
        )

    #Utility method to call k8s commands
    #given a kubeconfig
    def run_k8s_command(self, k8_cmd, kc):
        with kc.as_file() as kc_path:
            cmd = ('%s --kubeconfig=%s --server=%s  %s ' %
                      (self.kubectl, kc_path, self.api_server, k8_cmd))
            log.info("Running command " + cmd)
            rc, output = run_command(cmd)
        return rc, output

    #Tests if default role bindings are available
    def test_rbac_is_enabled(self):
        rc, output = self.run_k8s_command("get clusterroles", self.admin_kc)
        if rc != 0 or "apiproxy-role" not in output:
                raise Exception("Default role bindings not created")

    #Tests if user with admin role can
    #access the entire cluster
    def test_admin_has_access(self):
        rc, output = self.run_k8s_command("get pods --all-namespaces", self.admin_kc)
        if rc != 0 or "forbidden" in output.lower() or "kube-system" not in output.lower():
                raise Exception("Admin user was not able to get all pods", self.admin_kc)

    #Generates kube config for the new keystone user
    def setup_get_user_kc(self):
        self.user_kc = get_kubeconfig(self.qbert, self.cluster['name'], self.ks_user['name'], 'testuser', force=True)

    #Tests access of user who has no
    #role in the service project
    def test_user_with_no_role(self):
        rc, output = self.run_k8s_command("get pods", self.user_kc)
        if rc == 0:
            raise Exception("User with no role was able to get pods ")

    #Tests access of user who only has _member_ role
    #in service project but no RBAC bindings
    def test_user_with_only_member_role(self):
        #Assign member role to user for the project
        self.ks_project = tenant_name_to_id(self.du_fqdn, self.ks_token, "service")
        self.ks_role = role_name_to_id(self.du_fqdn, self.ks_token, "_member_")
        add_role_to_user(self.du_fqdn, self.ks_token, self.ks_project, self.ks_user['id'], self.ks_role)
        #Get pods should still fail, but this time with forbidden error
        rc, output = self.run_k8s_command("get pods", self.user_kc)
        if rc == 0 or "forbidden" not in output.lower():
            raise Exception("User with no rbac permissions was able to get pods ")
        #However should still be able to get namespaces
        rc, output = self.run_k8s_command("get namespaces", self.user_kc)
        if rc != 0 or "kube-system" not in output.lower():
            raise Exception("User not able to list namespaces")

    #Grants permission to the user
    #kind: either User/Group
    #name:  name of user or group the permission is assigned
    def test_grant_rbac_and_check_access(self, kind, name):
        with tempfile.NamedTemporaryFile() as kubeconfig_file:
            self.role_binding['subjects'][0]['kind'] = kind
            self.role_binding['subjects'][0]['name'] = name.encode("utf-8")
            kubeconfig_file.write(yaml.dump(self.role_binding).encode())
            kubeconfig_file.flush()
            rc_create, output_create = self.run_k8s_command("apply -f " + kubeconfig_file.name, self.admin_kc)
            rc, output = self.run_k8s_command("get pods --all-namespaces", self.user_kc)
            rc_delete, output_delete = self.run_k8s_command("delete -f " + kubeconfig_file.name, self.admin_kc)
            if rc != 0 or "kube-system" not in output.lower():
                raise Exception("User " + self.user_kc + " not able to list namespaces after granting permissions to " + kind + " " + name)

    #Grants access to user using username
    def test_grant_rbac_to_username(self):
        self.test_grant_rbac_and_check_access('User', self.ks_user['name'])

    #Grants access to user using the group user belongs to
    def test_grant_rbac_to_groupname(self):
        self.test_grant_rbac_and_check_access('Group', self.ks_group['name'])

    #Grants access to user using group ssu_users
    #All users with _member_ role belong to this group
    def test_grant_rbac_to_ssu_users(self):
        self.test_grant_rbac_and_check_access('Group', 'ssu_users')


#Called as part of test_kubernetes.
def test_kubernetes_rbac(kc, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, kubectl):
    log.info("Starting RBAC tests")
    rbac = Rbac(kc, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, kubectl)
    rbac.setup_ks_user()
    rbac.setup_ks_group()
    log.info("Created keystone user and group for RBAC tests")
    rbac.setup_get_user_kc()
    log.info("Got kube config for the newly created user")
    rbac.setup_base_yaml()
    rbac.test_rbac_is_enabled()
    log.info("Successfully tested that RBAC is enabled")
    rbac.test_admin_has_access()
    log.info("Successfully tested that user with admin role has access")
    rbac.test_user_with_no_role()
    log.info("Successfully tested that user with no role in project has no access")
    rbac.test_user_with_only_member_role()
    log.info("Successfully tested user with only _member_ role in the project")
    rbac.test_grant_rbac_to_username()
    rbac.test_grant_rbac_to_groupname()
    rbac.test_grant_rbac_to_ssu_users()
    log.info("Successfully tested assigning access to user through username, groupname and ssu_user groupname")

