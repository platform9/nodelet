import base64
import collections
import contextlib
import json
import logging
import tempfile
import yaml

from pf9lab.retry import retry


log = logging.getLogger(__name__)


class Kubeconfig(collections.namedtuple('Kubeconfig', 'kubeconfig_as_dict')):

    def cluster(self, name):
        for cluster in self.kubeconfig_as_dict['clusters']:
            if cluster['name'] == name:
                return cluster['cluster']
        raise ValueError('cluster %s not found in kubeconfig' % name)

    def user(self, name):
        for user in self.kubeconfig_as_dict['users']:
            if user['name'] == name:
                return user['user']
        raise ValueError('user %s not found in kubeconfig' % name)

    def set_password(self, username, password):
        for user in self.kubeconfig_as_dict['users']:
            if user['name'] == username:
                user['user']['password'] = password
                return
        raise ValueError('user %s not found in kubeconfig' % username)

    def set_token(self, username, token, force=False):
        for user in self.kubeconfig_as_dict['users']:
            #force simulates the case where admin can generate
            #kubeconfig for another user. Needed for RBAC tests
            if force:
                user['user']['token'] = token
            if user['name'] == username:
                user['user']['token'] = token
                return
        if not force:
            raise ValueError('user %s not found in kubeconfig' % username)

    @contextlib.contextmanager
    def cluster_ca_file(self, name):
        ca = self.cluster(name)['certificate-authority-data']
        ca_decoded = base64.b64decode(ca)
        with tempfile.NamedTemporaryFile() as ca_file:
            ca_file.write(ca_decoded)
            ca_file.flush()
            yield ca_file.name

    @contextlib.contextmanager
    def as_file(self):
        with tempfile.NamedTemporaryFile() as kubeconfig_file:
            kubeconfig_file.write(yaml.dump(self.kubeconfig_as_dict).encode())
            kubeconfig_file.flush()
            yield kubeconfig_file.name


def get_kubeconfig(qbert, cluster_name, keystone_user, passwd, force=False):

    # retry introduced as workaround for IAAS-5544 / PMK-155
    @retry(log=log, max_wait=2400, interval=10)
    def _get_kubeconfig_as_dict(cluster_name):
        log.info('Getting kubeconfig for cluster %s, user %s',
                 cluster_name, keystone_user)
        kubeconfig_yaml = qbert.get_kubeconfig(cluster_name)
        return yaml.safe_load(kubeconfig_yaml)

    log.info('Initializing kubeconfig for cluster %s', cluster_name)
    kc_dict = _get_kubeconfig_as_dict(cluster_name)
    kc = Kubeconfig(kc_dict)

    # Replace user/pass in kubeconfig with base64 encoded string
    user_data = {'username': keystone_user,
                 'password': passwd}
    kc_token = base64.b64encode(json.dumps(user_data).encode())
    kc.set_token(keystone_user, kc_token.decode(), force)
    return kc
