import requests
import logging
import json
import tempfile
import yaml

from os import path
from time import sleep
from kube_tests.lib.command_utils import run_command
from pf9lab.retry import retry

log = logging.getLogger(__name__)

class NetworkPolicy(object):

    def __init__(self, kc, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, kubectl, network_plugin):
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
        self.network_plugin = network_plugin
        self.yaml_dir_path = path.abspath(path.join(path.dirname(__file__), "network_policy_yamls"))
        self.daemonset_file = path.abspath(path.join(self.yaml_dir_path, "daemonset.yaml"))
        self.allow_file = path.abspath(path.join(self.yaml_dir_path, "allow.yaml"))
        self.deny_file = path.abspath(path.join(self.yaml_dir_path, "deny.yaml"))
        self.read_daemonset_yaml()

    def run_k8s_command(self, k8_cmd, kc):
        with kc.as_file() as kc_path:
            cmd = ('%s --kubeconfig=%s --server=%s %s' %
                      (self.kubectl, kc_path, self.api_server, k8_cmd))
            log.info("Running command " + cmd)
            rc, output = run_command(cmd)
        return rc, output

    def read_daemonset_yaml(self):
        with open(self.daemonset_file, 'r') as stream:
            try:
                self.daemonset = yaml.safe_load(stream)
            except yaml.YAMLError as e:
                raise Exception("Daemonset spec not present: ", e)
        log.info(self.daemonset)

    def apply_policy(self, policy_file):
        #Create a NetworkPolicy
        rc_create, out = self.run_k8s_command("apply -f " + policy_file, self.admin_kc)
        log.info('Result Code - %s', rc_create)
        log.info('Output - %s', out)
        if rc_create != 0:
            raise Exception("Could not create network policy, the error is: ", out)

    def create_daemonset(self):
        #Create a Daemonset
        rc_create, out = self.run_k8s_command("apply -f " + self.daemonset_file, self.admin_kc)
        log.info('Result Code - %s', rc_create)
        log.info('Output - %s', out)
        if rc_create != 0:
                raise Exception("Could not create daemonset, the error is: ", out)

    def apt_update_pod(self, pod_name):
        #Update sources in pod to avoid install failure
        rc_update, output_update = self.run_k8s_command("exec -it " + pod_name + " -- /usr/bin/apt update", self.admin_kc)
        log.info("rc_apt_update - %s", rc_update)
        log.info("out_apt_update - %s", output_update)
        return rc_update, output_update

    def apt_install_ping(self, pod_name):
        #Install ping utilities to the pod
        rc_update, output_update = self.run_k8s_command("exec -it " + pod_name + " -- /usr/bin/apt install -y iputils-ping", self.admin_kc)
        log.info("rc_ping_install - %s", rc_update)
        log.info("out_ping_install - %s", output_update)
        return rc_update, output_update

    @retry(log=log, max_wait=80, interval=20)
    def check_ping_fail(self, pod1_name, pod2_ip):
        log.info("pod1 name - %s", pod1_name)
        log.info("pod2 ip - %s", pod2_ip)

        #Run ping operation to verify network connection
        rc_ping, output_ping = self.run_k8s_command("exec -it " + pod1_name + " -- /bin/ping -c 4 -q " + pod2_ip, self.admin_kc)
        log.info("rc_ping - %s", rc_ping)
        log.info("out_ping - %s", output_ping)
        if rc_ping == 0:
            raise RuntimeError("Didn't get the expected result code while checking the ping failure, retrying..")
        return rc_ping, output_ping

    @retry(log=log, max_wait=80, interval=20)
    def check_ping_success(self, pod1_name, pod2_ip):
        log.info("pod1 name - %s", pod1_name)
        log.info("pod2 ip - %s", pod2_ip)

        #Run ping operation to verify network connection
        rc_ping, output_ping = self.run_k8s_command("exec -it " + pod1_name + " -- /bin/ping -c 4 -q " + pod2_ip, self.admin_kc)
        log.info("rc_ping - %s", rc_ping)
        log.info("out_ping - %s", output_ping)
        if rc_ping != 0:
            raise RuntimeError("Didn't get the expected result code while checking the ping success, retrying..")
        return rc_ping, output_ping

    @retry(log=log, max_wait=200, interval=20)
    def _get_pod_ip(self, pod_name):
        rc, out = self.run_k8s_command("get pods " + pod_name + " -o=custom-columns=NAME:.status.phase | tail -n +2", self.admin_kc)
        if 'Running' not in out:
            raise RuntimeError("Pod has no IP yet, retrying")

        rc, out = self.run_k8s_command("get pods " + pod_name + " -o=custom-columns=NAME:.status.podIP | tail -n +2", self.admin_kc)
        log.info("rc - %s", rc)
        log.info("out - %s", out)
        if out is None or out == "":
            raise RuntimeError("Pod has no IP yet, retrying")
        #Ignoring the new line character sent by Kubernetes
        ip = out.split('\n')[0]
        log.info("pod ip is - %s", ip)
        if not ip or 'error' in ip:
            raise RuntimeError("Pod has no IP yet, retrying")
        return rc, ip

    def get_pod_ip(self, pod_name):
        _, ip = self._get_pod_ip(pod_name)
        return ip

    def _select_pods_by_labels(self, labels):
        rc, out = self.run_k8s_command("get pods -l " + "name=" + labels["name"] + " -o=custom-columns=NAME:.metadata.name | tail -n +2", self.admin_kc)
        log.info("rc - %s", rc)
        log.info("out - %s", out)
        pods = out.split('\n')
        return rc, pods

    def select_pods_by_labels(self, labels):
        _, pods = self._select_pods_by_labels(labels)
        return pods

    def setup_pod(self, pod_name):
        self.apt_update_pod(pod_name)
        rc,_ = self.apt_install_ping(pod_name)
        if rc != 0:
            raise Exception("Could not install ping utils to the pod, check network connection")

    def validate_mtu(self, pod_name, expected_mtu):
        rc, cat_out = self.run_k8s_command("exec -it " + pod_name + " -- cat /sys/class/net/eth0/mtu", self.admin_kc)
        if rc != 0 or cat_out == '':
            raise RuntimeError("Failed to read MTU from the pod.")
        # Ignore the other text and just read the last word which is MTU size
        mtu_out = cat_out.split()[-1]
        if not mtu_out.isdigit() or int(mtu_out) != expected_mtu:
            raise RuntimeError("MTU test failed. Expected %d, got %s." % (expected_mtu, mtu_out))
        else:
            log.info("MTU validation passed.")
        return rc, mtu_out

    def test_networking(self):
        pods = self.select_pods_by_labels(self.daemonset['spec']['template']['metadata']['labels'])
        log.info(pods)

        #Run only if there is more than one pod on a calico environment
        if len(pods) < 2 or self.network_plugin != 'calico':
            raise Exception("Nothing to test")

        #Fetching ips of the first 2 pods by their names
        pod1_ip = self.get_pod_ip(pods[0])
        pod2_ip = self.get_pod_ip(pods[1])

        for pod in pods[:2]:
            #Ignoring the empty string returned by Kubernetes
            if len(pod) > 0:
                self.setup_pod(pod)

        #As a part of initial verification of the setup
        #Veryfing ping from pod1 to pod2 and back
        rc1, _ = self.check_ping_success(pods[0], pod2_ip)
        rc2, _ = self.check_ping_success(pods[1], pod1_ip)

        if rc1 != 0 or rc2 != 0:
            raise Exception("Network test in pods failed, check cni settings.")

        log.info("Initial network verification passed.")

        #Applying deny all policy which should stop the communication between pods
        self.apply_policy(self.deny_file)
        rc1, _ = self.check_ping_fail(pods[0], pod2_ip)
        rc2, _ = self.check_ping_fail(pods[1], pod1_ip)

        if rc1 != 0 and rc2 != 0:
            log.info("Got expected network error. Deny policy test passed")
        else:
            raise Exception("Deny policy did not work!.")

        #Applying allow all policy which should allow the communication between pods
        self.apply_policy(self.allow_file)
        rc1, _ = self.check_ping_success(pods[0], pod2_ip)
        rc2, _ = self.check_ping_success(pods[1], pod1_ip)

        if rc1 != 0 or rc2 != 0:
            raise Exception("Allow policy did not work!")
        else:
            log.info("Allow policy test passed")

        # Validate MTU size
        if self.cluster['networkPlugin'] != 'calico':
            log.info('skipping calico MTU test since network backend is not calico')
        else:
            expected_mtu = int(self.cluster['mtuSize'])
            rc1, _ = self.validate_mtu(pods[0], expected_mtu)
            if rc1 != 0:
                raise Exception("MTU validation failed!")

        return "Network policy tests passed"

def test_k8s_network_policy(kc, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, kubectl,
            network_plugin):
    log.info("Starting network policy tests")
    network_policy = NetworkPolicy(kc, qbert, cluster, du_fqdn, ks_token, api_server, cloud_provider_type, kubectl, network_plugin)
    log.info("Creating Daemonset")
    network_policy.create_daemonset()
    result = network_policy.test_networking()
    log.info(result)
