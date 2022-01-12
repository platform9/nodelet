import requests
import logging
import json

from ipaddress import *
from os import path
from time import sleep
from kube_tests.lib.command_utils import run_command
from pf9lab.retry import retry

NUM_SVC = 10

log = logging.getLogger(__name__)

class TestMetalLB(object):

    def __init__(self, kc, qbert, cluster, api_server, kubectl):
        self.admin_kc = kc
        self.api_server = api_server
        self.kubectl = kubectl
        self.qbert = qbert
        self.cluster = cluster
        self.pool_size = _calc_pool_size(cluster['metallbCidr'])

    @retry(log=log, max_wait=600, interval=20)
    def run_k8s_command(self, k8_cmd, kc):
        with kc.as_file() as kc_path:
            cmd = ('%s --kubeconfig=%s --server=%s %s' %
                      (self.kubectl, kc_path, self.api_server, k8_cmd))
            log.info("Running command " + cmd)
            rc, output = run_command(cmd)
        return rc, output

    @retry(log=log, max_wait=600, interval=20)
    def ensure_apiserver_running(self):
        # Do `kubectl get pods` with retry until successful
        rc_ping, out = self.run_k8s_command("get pods", self.admin_kc)
        log.info('Result Code - %s', rc_ping)
        log.info('Output - %s', out)
        if rc_ping != 0:
            raise Exception("Failed to communicate with k8s api server. Error: %s" % (out))
        return True

    def create_services(self):
        #Create multiple services
        for service_name in _gen_service_name():
            rc_create, out = self.run_k8s_command("create service loadbalancer %s --tcp=5678:8080" %(service_name), self.admin_kc)
            log.info('Result Code - %s', rc_create)
            log.info('Output - %s', out)
            if rc_create != 0:
                raise Exception("Could not create service %s, the error is: %s" % (service_name, out))

    @retry(log=log, max_wait=100, interval=20)
    def check_services(self):
        ips_pending = 0
        ips_used = 0
        for service_name in _gen_service_name():
            rc_get, out = self.run_k8s_command("get service %s -o=custom-columns=EXTERNAL-IP:.status.loadBalancer.ingress[0].ip"
                                                %(service_name), self.admin_kc)
            log.info('Result Code - %s', rc_get)
            log.info('Output - %s', out)
            if rc_get != 0:
                raise Exception("Could not list service %s, the error is: %s" % (service_name, out))
            ip = out.split('\n')[1]
            if ip != '<none>':
                if not _ip_in_pool(ip, self.cluster['metallbCidr']):
                    raise Exception("Service IP %s is outside of MetalLB pool %s" % (ip, metallb_cidr))
                ips_used += 1
            else:
                ips_pending += 1
        log.info('Pool size: %d, IPs used: %d, IPs pending: %d' %(self.pool_size, ips_used, ips_pending))
        if ips_pending > 0 and ips_used != self.pool_size:
            raise Exception("MetalLB range is not getting used fully.")
        return True

    def delete_services(self):
        #Create multiple services
        for service_name in _gen_service_name():
            rc_delete, out = self.run_k8s_command("delete service  %s" %(service_name), self.admin_kc)
            log.info('Result Code - %s', rc_delete)
            log.info('Output - %s', out)
            if rc_delete != 0:
                raise Exception("Could not delete service %s, the error is: %s" % (service_name, out))

def _calc_pool_size(pool):
    pool_size = 0
    # Get multiple ranges from the pool
    for r in pool.split(','):
        # Get lower and upper bounds of range
        r = r.split('-')
        lower_bound = r[0].strip()
        upper_bound = r[1].strip()
        next_ip = None
        i = 0
        while (next_ip != IPv4Address(str(upper_bound))):
            next_ip = IPv4Address(str(lower_bound)) + i
            i += 1
            pool_size += 1
    return pool_size

def _gen_service_name():
    for i in range (NUM_SVC):
        service_name = "svc-%d" %(i)
        yield service_name

def _ip_in_pool(ip, pool):
    # Get multiple ranges from the pool
    for r in pool.split(','):
        # Get lower and upper bounds of range
        r = r.split('-')
        lower_bound = r[0].strip()
        upper_bound = r[1].strip()
        if _ip_in_range(ip, lower_bound, upper_bound):
            return True
    return False


def _ip_in_range(ip, lower_bound, upper_bound):
    return IPv4Address(str(lower_bound)) <=  IPv4Address(str(ip)) and \
           IPv4Address(str(upper_bound)) >=  IPv4Address(str(ip))

def test_k8s_metallb(kc, qbert, cluster, api_server, kubectl):
    log.info("Starting MetalLB test")
    test_metallb = TestMetalLB(kc, qbert, cluster, api_server, kubectl)
    test_metallb.ensure_apiserver_running()
    log.info("Creating services")
    test_metallb.create_services()
    log.info("Checking services")
    test_metallb.check_services()
    log.info("Deleting services")
    test_metallb.delete_services()
    log.info('MetalLB test successful')
