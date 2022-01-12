# Copyright (c) 2016 Platform9 systems. All rights reserved

import glob
import uuid

from fabric.api import run, put
from fabric.contrib import files
from pf9lab.hosts.authorize import typical_fabric_settings
from pf9lab.testbeds.common import wait_for_service_running
from kube_tests.testbeds.teamcity_utils import download_rpm_from_last_release_build

pf9_kube_initd = "/etc/init.d/pf9-kube"
nodeletd_binary = "/opt/pf9/nodelet/nodeletd"

def uninstall_rpm_on_hosts(hosts, proxy):
    # Stops services and erases rpm
    for host in hosts:
        with typical_fabric_settings(host['ip']):
            run('rm -rf /rpms/*.rpm')
            if files.exists(pf9_kube_initd):
                res = run("%s status 2>&1 > /dev/null" % pf9_kube_initd)
                if res.succeeded:
                    assert run("%s stop 2>&1 > /dev/null" % pf9_kube_initd).succeeded
            elif run('%s phases status' % nodeletd_binary).succeeded:
                assert run('%s phases stop' % nodeletd_binary).succeeded
            # IAAS-6419: ensure proper node cleanup
            cmd = ' && '.join([
                'source /etc/pf9/kube.env',
                'cd /opt/pf9/pf9-kube',
                'source utils.sh',
                'source wait_until.sh',
                '! kubelet_running',
                '! docker_running'
            ])
            ret = run(cmd)
            if not ret.succeeded:
                print('failed to verify stopped state: ', ret)
                assert ret.succeeded

        run('yum -y erase /opt/pf9/pf9-kube')
        run('yum -y erase /opt/pf9/www/private')

    with typical_fabric_settings(proxy):
        run('rm -rf /rpms/*')
        run('service pf9-caproxy stop')
        run('yum -y erase /opt/pf9/qbert')


def install_prereqs_on_host(host_ip, skip_prereqs, proxy_host=False):
    if skip_prereqs:
        return
    with typical_fabric_settings(host_ip):
        # ntpdate requires ntpd to be stopped
        run('systemctl stop ntpd || true')
        ret = run('yum -y install ntpdate && ntpdate -s time.nist.gov')
        if not ret.succeeded:
            # ntpdate occasionally fails without an error message, not sure why
            print('warning: ntpdate failed:', ret)
        run('mkdir -p /rpms')
        run('useradd pf9')
        run('groupadd pf9group')
        if proxy_host:
            # Quick fix for qbert's nginx dependecy
            run('ln -s /usr/lib/systemd/system/sshd.service /usr/lib/systemd/system/nginx.service')


def install_rpm_on_hosts(hosts, proxy_host, kube_rpm_path=None, skip_prereqs=False):

    # Uses the rpm specified in the path if provided else downloads
    # from last release tagged version in teamcity
    if not kube_rpm_path:
        kube_rpm_path = download_rpm_from_last_release_build()

    for host in hosts:
        install_prereqs_on_host(host['ip'], skip_prereqs)
        with typical_fabric_settings(host['ip']):
            put(glob.glob(kube_rpm_path + '/pf9-kube*')[0], '/rpms')
            run('rpm -Uvh --nodeps /rpms/pf9-kube*')
            run('rpm -Uvh --nodeps /opt/pf9/www/private/pf9-kube*.rpm')

    install_prereqs_on_host(proxy_host, skip_prereqs, proxy_host=True)
    with typical_fabric_settings(proxy_host):
        put(glob.glob(kube_rpm_path + '/pf9-qbert*')[0], '/rpms')
        run('rpm -Uvh --nodeps /rpms/pf9-qbert*')


def start_kube_services(hosts, proxy_host, start_signd_on_proxy=False,
                        async=False):

    with typical_fabric_settings(proxy_host):
        # configure pf9-caproxy to use a mock node/cluster validator
        run('systemctl stop ntpd || true')
        ret = run('ntpdate -s time.nist.gov')
        if not ret.succeeded:
            print('warning: ntpdate failed:', ret)
        run("echo -e 'CAPROXY_OPTIONS=-mockvalidator\\n' > /etc/pf9/caproxy.env")
        run("service pf9-caproxy start", pty=False)
        wait_for_service_running(proxy_host, "pf9-caproxy")
        # pf9-signd started on proxy host starting with 2.3.0. Prior to that
        # it is started automatically with pf9-kube start on master
        if start_signd_on_proxy:
            run('ln -s /usr/lib/systemd/system/sshd.service /usr/lib/systemd/system/mysqlfs.service')
            run("service pf9-signd start", pty=False)
            wait_for_service_running(proxy_host, "pf9-signd")

    for host in hosts:
        with typical_fabric_settings(host['ip']):
            cmd = "%s phases start" % nodeletd_binary
            if files.exists(pf9_kube_initd):
                cmd = "%s start" % pf9_kube_initd
            if async:
                # Start pf9-kube in the background to allow master nodes
                # to start in parallel, which is required for the etcd cluster
                # to initialize correctly in a multimaster configuration.
                # FIXME: how to handle intermittent docker bugs like the one
                #        mentioned below in the non-async case?
                assert run("%s &> /var/log/kubestart.log &" % cmd,
                           pty=False).succeeded

            elif not run(cmd, pty=False).succeeded:
                # Can fail due to intermittent docker bugs such as
                # https://github.com/docker/docker/issues/14048
                print('Warning: first service start failed, retrying...')
                assert run(cmd, pty=False).succeeded
    for host in hosts:
        wait_for_service_running(host['ip'], "pf9-kube")


def set_etcd_env(hosts):
    """
    Initialize etcd.env for each host.
    """
    def label(host):
        return host['private_ip']

    def peer_url(host):
        # FIXME Change protocol + port once IAAS-6850 is resolved
        return 'http://%s:2380' % host['private_ip']

    common_etcd_env = {
        'ETCD_INITIAL_CLUSTER_STATE': 'new',
        'ETCD_INITIAL_CLUSTER': 'ETCD_INITAL_CLUSTER=' \
            + ",".join(['%s=%s' % (label(host), peer_url(host)) for host in hosts]),
        'ETCD_PROXY': 'OFF'
    }
    for host in hosts:
        etcd_env = common_etcd_env.copy()
        etcd_env['ETCD_NAME'] = label(host)
        with typical_fabric_settings(host['ip']):
            with open("/tmp/etcd.env", 'w') as f:
                lines = ['%s=%s\n' % (key,val) for key,val in etcd_env.items()]
                f.writelines(lines)
            run('mkdir -p /etc/pf9/kube.d/etcd')
            put("/tmp/etcd.env", "/etc/pf9/kube.d/etcd/etcd.env")

def set_kube_env(hosts, proxy_ip, options=None, is_v3=False, master_ip=None,
                 num_masters=0):
    """
    Initialize kube.env for each host.
    In a multimaster configuration, set num_masters to the number of masters,
    otherwise set it to zero.
    """
    if not master_ip:
        master_ip = hosts[0]['private_ip']
    # Default options if options are not provided
    kube_env = {"MASTER_IP": master_ip,
                "FLANNEL_PUBLIC_IFACE_LABEL": "",
                "CLUSTER_ID": str(uuid.uuid4()),
                "CONTAINERS_CIDR": "172.31.0.0/20",
                "SERVICES_CIDR": "172.31.16.0/20",
                "ETCD_DATA_DIR": "/var/opt/pf9/kube/etcd/data",
                "FLANNEL_IFACE_LABEL": "",
                "DEBUG": "true",
                "DOCKER_ROOT": "/var/lib",
                "PRIVILEGED": "false",
                "AUTHZ_ENABLED": "true",
                "KEYSTONE_ENABLED": "false"}
    caproxy_url = "http://{0}:3931".format(proxy_ip)
    if is_v3:
        caproxy_url += "/v3"
    with open("/tmp/caproxy.env", 'w') as f:
        lines = ['export %s=%s\n' % ("CAPROXY_URL", caproxy_url) ]
        f.writelines(lines)
    with open("/tmp/signd.env", 'w') as f:
        lines = ['export %s=%s\n' % ("SIGND_PROXY_URL", caproxy_url)]
        f.writelines(lines)
    if options:
        for option in options:
            kube_env[option] = options[option]
    host_idx = 0
    for host in hosts:
        with typical_fabric_settings(host['ip']):
            if num_masters:
                kube_env['ROLE'] = 'master' if host_idx < num_masters else 'worker'
                host_idx += 1
            elif host['private_ip'] == kube_env["MASTER_IP"]:
                kube_env["ROLE"] = "master"
            else:
                kube_env["ROLE"] = "worker"
            with open("/tmp/kube.env", 'w') as f:
                lines = ['export %s=%s\n' % (key,val) for key,val in kube_env.items()]
                f.writelines(lines)
            put("/tmp/kube.env", "/etc/pf9/kube.env")
            put("/tmp/caproxy.env", "/etc/pf9/kube.d/caproxy.env")
            put("/tmp/signd.env", "/etc/pf9/kube.d/signd.env")

    return kube_env["MASTER_IP"]
