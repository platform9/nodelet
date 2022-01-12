# Copyright (c) Platform9 systems. All rights reserved

import logging
import os

from pf9lab.testbeds import Testbed
from pf9lab.testbeds.common import generate_short_du_name
from kube_tests.testbeds.utils import (wait_for_qbert_up,
        get_qbert_log, get_image_id, update_qbert_config)
from pf9lab.du.auth import login
from pf9lab.du import common as du_common
from pf9lab.du.provider import get_du_provider
from pf9lab.du.decco import DeccoDuProvider

du_provider = get_du_provider()

logging.basicConfig(level=logging.INFO)
LOG = logging.getLogger(__name__)


class CloudDuTestbed(Testbed):
    """
    Creates a testbed with an EC2/decco/pf9_cloud DU.

    In addition, uses the following environment variables:
    AMI_ID - Image id used to instantiate the image. Uses the image from
               the latest successful build as default.
    VSPHERE_PASSWORD
    AWS_ACCESS_KEY_DEV
    AWS_SECRET_KEY_DEV
    AWS_DEFAULT_REGION (not required, default us-west-1)
    """
    def __init__(self, du, template_key, is_private, runtime_config):
        self._du = du
        self.template_key = template_key
        self.is_private = is_private
        self.runtime_config = runtime_config

    @staticmethod
    def create(tag, template_key, is_private, runtime_config):
        """
        Create the testbed
        :param tag: short string to embed in names of DU
        :param template_key: string which usually is an OS, here are some used currently:
                "ubuntu", "centos", "centos7", "centos7-latest"
        :param is_private: boolean, if True, instances will be deployed on private subnets so that they aren't
                assigned public IPs. The instances will use NAT gateway for external traffic
        :param runtime_config: passed through to k8s API server's '--runtime-config' flag
                https://kubernetes.io/docs/tasks/administer-cluster/cluster-management/#turn-on-or-off-an-api-version-for-your-cluster
        """
        image_id = get_image_id(du_provider)
        """
        pf9-kube is not impacted by any changes in pf9-main repo.

        Hence there is no incentive to use local branch and ref. Use the
        ones provided in suite file.
        """
        branch = os.getenv('AMI_BRANCH', 'local')
        du_ref = os.getenv('DU_REF', 'local')
        LOG.info("image_id/manifest: %s", image_id)
        LOG.info("branch: %s", branch)

        # feature flags for DU deploy
        feature_flags = {
            'containervisor': True,
            'openstackEnabled': False,
        }

        # This flag was introduced v5.0 onwards
        if branch != 'platform9-v5.0':
            feature_flags['containervisor_only'] = True

        shortname = generate_short_du_name(tag)
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
                du = du_provider.create_du(shortname=shortname, manifest=image_id)
            else:
                instance_size = os.getenv('INSTANCE_TYPE') or du_provider.get_default_instance_size()
                LOG.info("instance_size: %s", instance_size)
                du = du_provider.create_du(shortname=shortname, image_id=image_id,
                                           instance_size=instance_size,
                                           features=feature_flags,
                                           ref=du_ref, release=branch)

        du['private_key'] = du_provider.get_du_private_keyfile()
        LOG.info("DU for Cloud Testbed is %s", du['fqdn'])
        LOG.debug(du)

        auth_info = login('https://%s' % du['fqdn'],
                          du['customize_env_vars']['ADMINUSER'],
                          existing_du_password or du['customize_env_vars']['ADMINPASS'],
                          'service')
        du['token'] = auth_info['access']['token']['id']
        du['tenant_id'] = auth_info['access']['token']['tenant']['id']
        update_qbert_config(du, 'fail_on_error', 'true', du_provider,
            section='sunpike')
        wait_for_qbert_up(du, du_provider)

        return CloudDuTestbed(du, template_key, is_private, runtime_config)

    def get_qbert_log(self):
        return get_qbert_log(self._du['ip'])

    @staticmethod
    def type_name():
        """ Helper method to define type name for new-style testbeds,
        found outside of pf9lab.testbeds """
        return __name__ + '.' + CloudDuTestbed.__name__

    @staticmethod
    def from_dict(desc):
        """ desc is a dict """
        type_name = CloudDuTestbed.type_name()
        if desc['type'] != type_name:
            raise ValueError(
                'attempt to build %s with %s' % (type_name, desc['type']))
        return CloudDuTestbed(desc['du'], desc['template_key'],
                            desc['is_private'], desc['runtime_config'])

    def to_dict(self):
        type_name = CloudDuTestbed.type_name()
        return { 'type': type_name, 'du': self._du,
                 'template_key': self.template_key,
                 'is_private': self.is_private,
                 'runtime_config': self.runtime_config }

    def destroy(self):
        LOG.info("Destroying the testbed")
        du_provider.teardown(self._du)

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

    def du_private_key(self):
        return self._du['private_key']
