# Copyright (c) Platform9 systems. All rights reserved

import logging
import os
import time
from random import randint

from pf9lab.du.auth import login
from pf9lab.du import common as du_common
from pf9lab.du.provider import get_du_provider
from pf9lab.du.decco import DeccoDuProvider
from pf9lab.hosts.provider import get_host_provider
from pf9lab.nova import set_powerops_validator
from pf9lab.testbeds import Testbed
from pf9lab.testbeds.common import generate_short_du_name
from . import utils

host_provider = get_host_provider()
du_provider = get_du_provider()

logging.basicConfig(level=logging.INFO)
LOG = logging.getLogger(__name__)

NUM_HOSTS_DEFAULT = 7

class RawKubeTestbed(Testbed):
    """
    Creates a testbed with a DU and 5 hosts.

    Hosts have hostagent and pf9-kube packages.
    They are authorized but uninitialized, meaning their service state
    is set to the default value of False.

    In addition, uses the following environment variables:
    AMI_ID - Image id used to instantiate the image. Uses the image from
               the latest successful build as default.
    DU_ENV - Determine whether the DU are provisioned on
                  AWS or dogfood
    HOST_ENV - Determine whether the hosts are provisioned on
                  vsphere or dogfood
    VSPHERE_PASSWORD
    AWS_ACCESS_KEY_DEV
    AWS_SECRET_KEY_DEV
    AWS_DEFAULT_REGION (not required, default us-west-1)

    vip_port is the Virtual IP port with following dictionary format:
    {'ip': u'10.105.20.53', 'id': u'71369588-307b-4305-8947-21f066c52984', 'name': u'907059'}
    """

    SUPPORTS_MONOCULAR = True

    def __init__(self, du, hosts, vip_port, has_docker_volume_group=False):
        self._du = du
        self.hosts = hosts
        self.vip_port = vip_port
        self._has_docker_volume_group = has_docker_volume_group
        set_powerops_validator(self.validate_power_state)

    @staticmethod
    def create(tag, template_key):
        """
        Create the testbed
        :param tag: short string to embed in names of DU and hosts
        :param template_key: key of the hypervisor template entry in
            template_mappings.json. For any new hypervisor OS being
            introduced, the entry needs to be added to template_mappings.json
        """
        image_id = utils.get_image_id(du_provider)
        # Must use 'local' to use update-pf9-rmps Ansible role
        """
        pf9-kube is not impacted by any changes in pf9-main repo.

        Hence there is no incentive to use local branch and ref. Use the
        ones provided in suite file.
        """
        branch = os.getenv('AMI_BRANCH', 'local')
        du_ref = os.getenv('DU_REF', 'local')
        LOG.info('image_id/manifest: %s', image_id)
        LOG.info('branch: %s', branch)
        utils.log_env_vars()

        # feature flags for DU deploy
        feature_flags = {
            'containervisor': True,
            'openstackEnabled': False,
        }

        # This flag was introduced v5.0 onwards
        if branch != 'platform9-v5.0':
            feature_flags['containervisor_only'] = True

        # Create hosts
        flavor_name = os.getenv('HOST_FLAVOR_NAME')
        if not flavor_name:
            raise Exception("HOST_FLAVOR_NAME not set.")
        num_hosts = int(os.getenv('NUM_HOSTS', NUM_HOSTS_DEFAULT))

        # TODO: change name of pf9-main provider function to be "get_provider_network"
        network_name = host_provider.get_multi_master_provider_network()
        LOG.info('host_flavour_name: %s, num_hosts: %s, network_name: %s', flavor_name, num_hosts, network_name)

        hosts, has_docker_vg = utils.create_hosts(host_provider, num_hosts,
                                                  tag, template_key,
                                                  flavor_name, network_name)
        LOG.info('Hosts for RawKubeTestbed are %s', [host['name'] for host in hosts])

        LOG.info("Creating port for VIP & associating it to hosts")
        vip_port_name = None
        teamcity_build_id = os.getenv('TEAMCITY_BUILD_ID')
        if teamcity_build_id is None:
            vip_port_name = os.getenv("USER") + "_local_" + str(randint(0000, 999999))
        else:
            vip_port_name = "teamcity_" + teamcity_build_id

        vip_port = utils.create_and_assign_vip_to_hosts(host_provider, hosts, network_name, vip_port_name)
        LOG.info("Created vip_port = %s", vip_port)

        LOG.debug(hosts)

        # Check if existing DU is to be used
        existing_du_fqdn = os.getenv('EXISTING_DU_FQDN', None)
        existing_du_password = None
        if existing_du_fqdn:
            existing_du_fqdn = existing_du_fqdn.lower()
            LOG.info("Found EXISTING_DU_FQDN to be set")
            LOG.info("Using %s as the DU", existing_du_fqdn)
            du = du_common.build_du_description(existing_du_fqdn)
            existing_du_password = du_provider.get_existing_du_password()
        else:
            # Create DU
            if isinstance(du_provider, DeccoDuProvider):
                du = du_provider.create_du(
                        shortname=generate_short_du_name(tag),
                        manifest=image_id)
            else:
                instance_size = (os.getenv('INSTANCE_TYPE') or
                                 du_provider.get_default_instance_size())
                LOG.info('instance_size: %s', instance_size)
                du = du_provider.create_du(
                    shortname=generate_short_du_name(tag),
                    image_id=image_id,
                    instance_size=instance_size,
                    features=feature_flags,
                    ref=du_ref,
                    release=branch)
        du['private_key'] = du_provider.get_du_private_keyfile()
        auth_info = login('https://%s' % du['fqdn'],
                          du['customize_env_vars']['ADMINUSER'],
                          existing_du_password or du['customize_env_vars']['ADMINPASS'],
                          'service')
        du['token'] = auth_info['access']['token']['id']
        du['tenant_id'] = auth_info['access']['token']['tenant']['id']
        LOG.info('DU for RawKubeTestbed is %s', du['fqdn'])
        LOG.debug(du)

        utils.update_qbert_config(du, 'fail_on_error', 'true', du_provider,
            section='sunpike')
        utils.wait_for_qbert_up(du, du_provider)
        utils.ensure_kube_role_installed(du, hosts, template_key)
        LOG.info('Sleeping for 180 seconds for hosts info to be reflected correctly in DB')
        time.sleep(180)

        return RawKubeTestbed(du, hosts, vip_port, has_docker_volume_group=has_docker_vg)

    @staticmethod
    def type_name():
        """ Helper method to define type name for new-style testbeds,
        found outside of pf9lab.testbeds """
        return __name__ + '.' + RawKubeTestbed.__name__

    @staticmethod
    def from_dict(desc):
        """ desc is a dict """
        type_name = RawKubeTestbed.type_name()
        if desc['type'] != type_name:
            raise ValueError('attempt to build %s with %s' %
                             (type_name, desc['type']))
        hosts = desc['hosts']
        return RawKubeTestbed(desc['du'], hosts, desc['vip_port'],
                              desc.get('has_docker_volume_group'))

    def to_dict(self):
        type_name = RawKubeTestbed.type_name()
        return {'type': type_name,
                'du': self._du,
                'has_docker_volume_group': self._has_docker_volume_group,
                'hosts': self.hosts,
                'vip_port': self.vip_port}

    def destroy(self):
        LOG.info('Destroying the testbed')
        host_provider.destroy_testbed_from_objs(self.hosts)
        host_provider.delete_port(self.vip_port['id'])
        du_provider.teardown(self._du)

    def has_docker_volume_group(self):
        return self._has_docker_volume_group

    def du_ip(self):
        return self._du['ip']

    def du_fqdn(self):
        return self._du['fqdn']

    def du_token(self):
        return self._du['token']

    def du_tenant_id(self):
        return self._du['tenant_id']

    def du_user(self):
        return self._du['customize_env_vars']['ADMINUSER']

    def du_pass(self):
        return du_provider.get_existing_du_password() or self._du['customize_env_vars']['ADMINPASS']

    def du_cust_shortname(self):
        return self._du['customize_env_vars']['CUSTOMER_SHORTNAME']

    def du_cust_fullname(self):
        return self._du['customize_env_vars']['CUSTOMER_FULLNAME']

    def validate_power_state(self, instance, power_state):
        return True

    def get_host_public_ip(self, private_ip):
        host_ip = [h['ip'] for h in self.hosts
                   if h['private_ip'] == private_ip][0]
        return host_ip

    def get_qbert_log(self):
        return utils.get_qbert_log(self._du['ip'])

    def get_vip_port(self):
        return self.vip_port
