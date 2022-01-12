from fabric.api import run
from fabric.contrib import files
from pf9lab.retry import retry

import pf9lab.hosts.authorize as labrole
from subprocess import run as subproc_run, STDOUT, PIPE # run from fabric.api should not conflict with run from subprocess
import logging


log = logging.getLogger(__name__)
nodelet_config = "/etc/pf9/nodelet/config_sunpike.yaml"
containerd_cfg_lines = 'RUNTIME: "containerd"'

#Executes a command with optional env
#Returns return code and output
def run_command(cmd, env=None):
    p = subproc_run(cmd,env=env,stderr=STDOUT,stdout=PIPE,universal_newlines=True,shell=True)
    log.debug("### Command '%s' ### Beginning of output", cmd)
    log.debug(p.stdout)
    log.debug("### Command '%s' ### End of output, returncode %s", cmd, p.returncode)
    return p.returncode, p.stdout


def _is_kube_service_running(host_ip):
    pf9_kube_initd = "/etc/init.d/pf9-kube"
    nodeletd_binary = "/opt/pf9/nodelet/nodeletd"
    with labrole.typical_fabric_settings(host_ip):
        if not files.exists(pf9_kube_initd):
            res = run('%s phases status 2>&1 > /dev/null' % nodeletd_binary)
            return res.return_code == 0
        res = run("%s status 2>&1 > /dev/null" % pf9_kube_initd)
        return res.return_code == 0


@retry(log=log, max_wait=1200, interval=10)
def wait_for_kube_service_running(host_ip):
    """
    Wait for the service status to return 0 (success, running). Raises
    on timeout.
    :param host_ip:
    :param service_name:
    """
    log.info('Waiting for service pf9-kube to start on host %s' % host_ip)
    return _is_kube_service_running(host_ip)


@retry(log=log, max_wait=1200, interval=10)
def wait_for_kube_service_stopped(host_ip):
    log.info('Waiting for service pf9-kube to stop on host %s' % host_ip)
    return not _is_kube_service_running(host_ip)


def _validate_container_runtime_docker(host_ip):
    with labrole.typical_fabric_settings(host_ip):
        # 1.20 config file will not have any configuration setting 
        # RUNTIME variable hence check if RUNTIME="containerd" is not set
        if not files.contains(nodelet_config, containerd_cfg_lines):
            # Further verify if docker is running or not.
            res = run("systemctl status docker > /dev/null 2>&1")
            return res.return_code == 0
    return False

def _validate_container_runtime_containerd(host_ip):
    with labrole.typical_fabric_settings(host_ip):
        if files.contains(nodelet_config, containerd_cfg_lines):
            # Further verify if docker is running or not.
            res = run("systemctl status docker > /dev/null 2>&1")
            if res.return_code == 0:
                log.error("docker is running on host %s", host_ip)
                return False
    return False

@retry(log=log, max_wait=60, interval=10)
def validate_container_runtime(host_ip, runtime="docker"):
    if runtime == "docker":
        return _validate_container_runtime_docker(host_ip)
    elif runtime == "containerd":
        return _validate_container_runtime_containerd(host_ip)
    return False
