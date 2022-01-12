# Copyright (c) Platform9 systems. All rights reserved

import logging
import os
import yaml

from pf9lab.du.auth import login
from pf9lab.du.provider import get_du_provider
from pf9lab.hosts.provider import get_host_provider
from pf9lab.testbeds import Testbed
from pf9lab.testbeds.common import generate_short_du_name
from pf9lab.nova import set_powerops_validator
import pf9lab.pf9deploy_wrapper as pf9deploy
from pf9lab.testbeds import testbed_utils

from kube_tests.testbeds.utils import (wait_for_qbert_up, create_hosts,
    ensure_kube_role_installed, log_env_vars, get_image_id,
    update_qbert_config)

host_provider = get_host_provider()
du_provider = get_du_provider()

logging.basicConfig(level=logging.INFO)
LOG = logging.getLogger(__name__)


class UpgradeableKubeTestbed(Testbed):
    """
    Deploys a DU with pf9deploy and 3 hosts on dogfood.
    Allows DU to be upgraded with pf9deploy.
    """
    def __init__(self, du, hosts, has_docker_volume_group=False):
        self._du = du
        self.hosts = hosts
        self._has_docker_volume_group = has_docker_volume_group
        set_powerops_validator(self.validate_power_state)

    @staticmethod
    def create(tag, template_key):
        """
        Create the testbed
        :param tag: short string to embed in names of DU and hosts
        :param template_key: key of the hypervisor template entry in template_mappings.json
                             For any new hypervisor OS being introduced, the entry needs to
                             be added to template_mappings.json
        """
        def create_du():
            pf9deploy.create_customer(shortname, fqdn)
            inserted_du = pf9deploy.insert_du_aws(shortname, fqdn, image_id, region,
                    features={'containervisor': True}, ref=branch,
                    release=branch, db_host=db_host)
            testbed_utils.raise_if_none(inserted_du, LOG, "Failed to insert du record",
                    "Insert DU - done")
            du_record = pf9deploy.generate_certs(fqdn)
            testbed_utils.raise_if_none(du_record, LOG, "Failed to generate certificates",
                    "Generate certificates - done")

            if db_host == 'rds':
                db_created = pf9deploy.create_db(fqdn, shortname)
                testbed_utils.raise_if_not_true(db_created, LOG, "Failed to create RDS database",
                        "Create RDS database - done")

            provisioned_du = pf9deploy.provision_du(fqdn)
            testbed_utils.raise_if_none(provisioned_du, LOG, "Failed to provision DU",
                    "Provision DU - done")

            dns_setup = pf9deploy.setup_dns_aws(fqdn)
            testbed_utils.raise_if_not_true(dns_setup, LOG, "Failed to setup dns", "Setup dns - done")

        def configure_du():
            du_is_configured = pf9deploy.configure_du(fqdn, region, branch)
            testbed_utils.raise_if_not_true(du_is_configured, LOG, "Failed to configure (deploy) DU",
                    "Configure (deploy) DU - done")

        def build_du_description():
            host_vars = pf9deploy.get_host_vars(fqdn)
            du_config = yaml.safe_load(host_vars)
            query_result = pf9deploy.query_du(fqdn)
            du_config['ip'] = query_result['ip_addr']
            du_config['fqdn'] = du_config['customize_env_vars']['DU_FQDN']
            du_config['shortname'] = shortname
            auth_info = login('https://%s' % du_config['fqdn'],
                              du_config['customize_env_vars']['ADMINUSER'],
                              du_config['customize_env_vars']['ADMINPASS'],
                              'service')
            du_config['token'] = auth_info['access']['token']['id']
            du_config['tenant_id'] = auth_info['access']['token']['tenant']['id']
            LOG.info("DU is %s", du_config['fqdn'])
            return du_config

        flavor_name = os.getenv('HOST_FLAVOR_NAME', '1cpu.2gb.40gb')
        num_hosts = int(os.getenv('NUM_HOSTS', 3))
        use_rds = os.getenv('USE_RDS', False)
        db_host = 'rds' if use_rds else 'localhost'
        image_id, branch, instance_type = du_provider.get_last_release_image_details()
        region = 'master'
        if os.getenv('REUSE_DU'):
            shortname = os.getenv('REUSE_DU')
        else:
            shortname = generate_short_du_name(tag)
        fqdn = shortname + ".platform9.net"

        LOG.info("IMAGE ID for last release install = %s", image_id)
        LOG.info("INSTANCE TYPE for last release install = %s", instance_type)
        LOG.info("BRANCH for last release install = %s", branch)
        log_env_vars()

        hosts, has_docker_vg = create_hosts(host_provider, num_hosts,
            tag, template_key, flavor_name)
        if not os.getenv('REUSE_DU'):
            create_du()
            configure_du()
        du = build_du_description()
        update_qbert_config(du, 'fail_on_error', 'true', du_provider,
            section='sunpike')
        wait_for_qbert_up(du, du_provider)
        ensure_kube_role_installed(du, hosts, template_key)

        return UpgradeableKubeTestbed(du, hosts, has_docker_volume_group=has_docker_vg)

    def upgrade(self, du_fqdn):
        # Upgrade DU
        LOG.info("Upgrading the testbed with du fqdn = %s", du_fqdn)

        # check if AMI_ID is set and use
        upgrade_ami_id = os.getenv('AMI_ID')
        if upgrade_ami_id:
            upgrade_instance_type = os.getenv('INSTANCE_TYPE')
            if not upgrade_instance_type:
                upgrade_instance_type = du_provider.get_instance_type_by_ami_id(upgrade_ami_id)
                if not upgrade_instance_type:
                    raise RuntimeError("Failed to find instance type for AMI ID %s " % upgrade_ami_id)
        else:
            upgrade_ami_id, upgrade_instance_type = get_image_id(du_provider)
            if not upgrade_ami_id:
                raise RuntimeError("Failed to find ami for instance type %s" % upgrade_instance_type)

        upgrade_branch = os.getenv('UPGRADE_BRANCH', 'atherton')
        release = os.getenv('RELEASE', upgrade_branch)

        LOG.info("Using AMI_ID = %s, INSTANCE_TYPE = %s and UPGRADE_BRANCH = %s for upgrade",
                upgrade_ami_id, upgrade_instance_type, upgrade_branch)

        upgrade_done = pf9deploy.upgrade_du(du_fqdn, upgrade_ami_id, release, upgrade_branch)
        testbed_utils.raise_if_not_true(upgrade_done, LOG, "Failed to upgrade DU", "Upgrade- done")

        # Wait for the qbert to be up which involves restarting
        # the qbert service which will then inject the role in resmgr
        # service.
        wait_for_qbert_up(self._du, du_provider)

    @staticmethod
    def type_name():
        """ Helper method to define type name for new-style testbeds,
        found outside of pf9lab.testbeds """
        return __name__ + '.' + UpgradeableKubeTestbed.__name__

    @staticmethod
    def from_dict(desc):
        """ desc is a dict """
        type_name = UpgradeableKubeTestbed.type_name()
        if desc['type'] != type_name:
            raise ValueError(
                'attempt to build %s with %s' % (type_name, desc['type']) )
        hosts = desc['hosts']
        return UpgradeableKubeTestbed(desc['du'], hosts,
                              desc.get('has_docker_volume_group'))

    def to_dict(self):
        type_name = UpgradeableKubeTestbed.type_name()
        return {'type': type_name,
                'du': self._du,
                'has_docker_volume_group': self._has_docker_volume_group,
                'hosts': self.hosts}

    def destroy(self):
        host_provider.destroy_testbed_from_objs(self.hosts)
        try:
            pf9deploy.wipe_customer(self.du_cust_shortname())
        except Exception:
            LOG.warn("Failed to destroy DUs or delete customer %s", self.du_cust_shortname())

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
        return self._du['customize_env_vars']['ADMINPASS']

    def validate_power_state(self, instance, power_state):
        return True

    def du_cust_shortname(self):
        return self._du['shortname']
