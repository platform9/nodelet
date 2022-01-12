# Copyright (c) Platform9 systems. All rights reserved

import logging
import os

from fabric.api import run, sudo, settings
from pf9lab.hosts.authorize import typical_fabric_settings
from pf9lab.retry import retry
from pf9lab.testbeds.common import (authorize_hosts_with_config)
from pf9lab.utils import is_debian, typical_du_fabric_settings
from kube_tests.lib.qbert import Qbert
from pf9lab.hosts.host_credentials import HOST_KEY_FILE
from pf9lab.du.decco import DeccoDuProvider

LOG = logging.getLogger(__name__)
PF9_KUBE_EXTRA_OPTS_PATH = '/etc/pf9/kube_extra_opts.env'
QBERT_LOG_PATH = '/var/log/pf9/qbert/qbert.log'


def fatal_errors_fabric_settings(host, key_filename):
    """Non-zero status errors raise exception"""
    return settings(host_string=host, user='root',
                    key_filename=key_filename,
                    disable_known_hosts=True)


def fatal_errors_host_fabric_settings(host):
    return fatal_errors_fabric_settings(host, HOST_KEY_FILE)


# If qbert log has been rotated, will only return most recent file
def get_qbert_log(du_ip):
    try:
        with typical_du_fabric_settings(du_ip):
            ret = run('cat {0}'.format(QBERT_LOG_PATH),
                      shell=True, pty=False)
            if ret.failed:
                raise Exception('cat failed: %s' % ret)
            return ret
    except Exception as exc:
        LOG.error('Unable to read contents of qbert log: %s', exc)


def update_qbert_config(du, option, value, du_provider, section=''):
    if isinstance(du_provider, DeccoDuProvider):
        LOG.info("Not updating config on decco DU")
        return
    try:
        with typical_du_fabric_settings(du['ip']):
            # add sunpike.fail_on_error = true in qbert.json and write it to qbert.json.new
            # move qbert.json to qbert.json.old
            # move qbert.json.new to qbert.json
            # These commands are idempotent and calling it multiple times does not add same option multiple times.
            # The command string cannot be generated using usual `format` call because there are multiple control
            # characters - ", ', {}
            base_cmd = "jq ." + section + " += {\"" + option + "\"=" + value +"}"
            cmd = """{cmd} <<< cat /etc/pf9/qbert.json > /etc/pf9/qbert.json.new && \
                mv -f /etc/pf9/qbert.json /etc/pf9/qbert.json.old && \
                mv /etc/pf9/qbert.json.new /etc/pf9/qbert.json""".format(
                    cmd=base_cmd)
            LOG.info("Running command {} on DU".format(cmd))
            ret = run(cmd, shell=True)
    except Exception as exc:
        LOG.error("Could not update qbert config with option {}".format(
            option))

def wait_for_qbert_up(du, du_provider):

    @retry(log=LOG, max_wait=180, interval=10)
    def restart_qbert():
        qbertApiUrl = 'https://%s/qbert' % du_fqdn
        qbert = Qbert(du_fqdn, token, qbertApiUrl, tenant_id)
        # For multiversion scenario, we want resmgr up when the
        # cronjob in qbert starts. Sometimes it can happen that
        # resmgr is not running/ready when the cron job tries to
        # add role into resmgr, and connection is refused.
        #
        # Restarting qbert solves that issue
        qbert.restart_qbert()
        return True

    @retry(log=LOG, max_wait=180, interval=10)
    def wait_for_qbert_api_up():
        qbertApiUrl = 'https://%s/qbert' % du_fqdn
        qbert = Qbert(du_fqdn, token, qbertApiUrl, tenant_id)
        # Until the API is up, this call will raise an exception
        qbert.list_clusters()
        return True

    try:
        du_fqdn = du['fqdn']
        token = du['token']
        tenant_id = du['tenant_id']
        if not isinstance(du_provider, DeccoDuProvider):
            restart_qbert()
        wait_for_qbert_api_up()
    except RuntimeError as err:
        LOG.error(err)
        qbert_log = get_qbert_log(du['ip'])
        LOG.info('Contents of qbert log:\n%s', qbert_log)
        raise err


def get_image_id(du_provider):
    if isinstance(du_provider, DeccoDuProvider):
        image_id = du_provider.get_ddu_manifest()
    else:
        image_id = du_provider.get_du_image()
    if not image_id:
        raise ValueError('DU provider did not find an image_id/manifest')
    return image_id


def log_env_vars():
    # Do NOT log all env vars; log only qbert specific ones
    LOG.info('Relevant environment variables:')
    for key in ['QBERT_API_VERSION']:
        LOG.info('%s: %s', key, os.getenv(key))


def create_hosts(host_provider, num_hosts, tag, template_key, flavor_name, network_name=None):
    hosts = host_provider.make_testbed(num_hosts,
                                       tag,
                                       template_key,
                                       flavor_name=flavor_name,
                                       networks=[network_name],
                                       assign_floating_ip=False)
    disable_swap(hosts)
    has_docker_vg = enable_docker_volume_group(hosts)
    host_provider.remove_non_serializable_entries(hosts)
    return hosts, has_docker_vg


def create_and_assign_vip_to_hosts(host_provider, hosts, network_name, vip_port_name):
    # Create a new port for VIP
    vip_port = host_provider.create_port(network_name, vip_port_name)

    # Assign VIP to hosts
    host_provider.add_allowed_ip_address_to_hosts(vip_port['ip'], hosts)
    return vip_port


# Default behavior of Kubelet from 1.8.4 is to abort if swap is enabled on the node
def disable_swap(hosts):
    # Disable swap for all tags
    for host in hosts:
        with typical_fabric_settings(host['ip']):
            sudo('swapoff -a')
            sudo("sudo sed -i '/ swap / s/^\(.*\)$/#\1/g' /etc/fstab")


def ensure_kube_role_installed(du, hosts, template_key):
    config = dict()
    for i in range(1, len(hosts) + 1):
        host = 'host%i' % i
        config[host] = {'roles': ['pf9-kube']}

    debian = is_debian(template_key)
    _ = authorize_hosts_with_config(du, hosts, config, debian)


def enable_docker_volume_group(hosts):
    has_at_least_one_docker_vg = False
    if os.environ.get('DISABLE_DOCKER_VOLUME_GROUP'):
        return False
    for host in hosts:
        with typical_fabric_settings(host['ip']):
            # Verify whether this is a RedHat/CentOS derivative
            if run('stat /etc/redhat-release').failed:
                continue
            # Skip if host already has docker-vg volume group
            if run('vgdisplay docker-vg').succeeded:
                continue

            # Create /dev/vda4 partition out of free space
            fdisk_cmd = '\n'.join(
                ['n', 'p', '4', '', '', 't', '4', '8e', 'p', 'w', ''])
            cmd = 'echo -e " %s " | fdisk /dev/vda' % fdisk_cmd
            res = sudo(cmd)
            assert res.return_code == 1
            # partx allows new partitions to be discovered without rebooting
            res = sudo('partx -v -a /dev/vda')
            assert res.return_code == 1

        with fatal_errors_host_fabric_settings(host['ip']):
            # Create docker-vg volume group from /dev/vda3 and /dev/vda4
            sudo('pvcreate /dev/vda3 && pvcreate /dev/vda4 && '
                 'vgcreate docker-vg /dev/vda3 /dev/vda4')
            # Create logical volumes (one for data, another for metadata)
            sudo('lvcreate --wipesignatures y -n thinpool "docker-vg" -l 95%VG && lvcreate --wipesignatures y -n thinpoolmeta "docker-vg" -l 1%VG')
            # Convert data volume to a thin volume, using metadata volume for thin volume metadata
            sudo('lvconvert -y --zero n -c 512K --thinpool "docker-vg/thinpool" --poolmetadata "docker-vg/thinpoolmeta"')
            # Ensure both volumes are extended as necessary
            # 1. Create a profile
            sudo('echo -e "activation {\nthin_pool_autoextend_threshold=80\nthin_pool_autoextend_percent=20\n}" > "/etc/lvm/profile/docker-vg-thinpool.profile"')
            # 2. Link profile to data volume
            sudo('lvchange --metadataprofile "docker-vg-thinpool" "docker-vg/thinpool"')
            # 3. Enable monitoring of data volume size, so that extension is triggered automatically
            sudo('lvs -o+seg_monitor')

        has_at_least_one_docker_vg = True
    return has_at_least_one_docker_vg
