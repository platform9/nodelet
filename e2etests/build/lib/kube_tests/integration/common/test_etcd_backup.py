import logging
from fabric.api import run
from pf9lab.hosts.authorize import typical_fabric_settings
from pf9lab.retry import retry
from pf9lab.utils import typical_du_fabric_settings, run_command_until_success
from proboscis.asserts import assert_equal, assert_true
from requests import HTTPError
import os
import time


log = logging.getLogger(__name__)


class etcd_backup_validator(object):
    def __init__(self, du_fqdn, qbert):
        self.du_fqdn = du_fqdn
        self.qbert = qbert


    def test_etcd_backup_on_cluster_error(self, uuid, backup_path=None):
        if not backup_path:
            backup_path = '/var/opt/pf9/etcd-backup'

        # Validate qbert rejects requests of invalid backup interval
        try:
            # By default qbert expects the interval to be greater than 30 mins
            # In the tests we make config changes for it to be greater than 1 min
            # Use 1 min interval here to ensure it will be idempotent on multiple runs
            log.info("Attempting etcd backup setting with 1 min backup interval")
            self.update_cluster(uuid, 1, backup_path)
        except Exception as e:
            assert_true(isinstance(e, HTTPError), 'expected Exception HTTPError, caught {}'.format(e))
            assert_equal(e.response.status_code, 400, 'update cluster call should have returned error code 400')


    def test_etcd_backup_on_cluster(self, uuid, backup_path=None):
        if not backup_path:
            backup_path = '/var/opt/pf9/etcd-backup'

        # Update qbert with new backup interval
        edited_conf = {"\"minBackupIntervalMins\": .*": "\"minBackupIntervalMins\": 1"}
        log.info("Updating qbert config with min etcd backup interval of 1 min")
        self.change_qbert_conf(edited_conf)
        # Validate qbert request with valid backup interval works
        self.update_cluster(uuid, 2, backup_path)
        self.wait_for_cluster_etcd_backup_complete(uuid, backup_path)


    def change_qbert_conf(self, params_to_edit):
        du_commands = ["cp /etc/pf9/qbert.json /etc/pf9/qbert.json.bkup"]

        for k, v in list(params_to_edit.items()):
            du_commands.append("sed 's/{}/{}/g' -i /etc/pf9/qbert.json".format(k, v))

        du_commands.append("systemctl restart pf9-qbert")

        log.info('Updating DU qbert config')
        with typical_du_fabric_settings(self.du_fqdn):
            for cmd in du_commands:
                run_command_until_success(cmd)

        # sleep 5 seconds to allow qbert service to settle down after restart
        time.sleep(5)


    def update_cluster(self, uuid, backup_interval, backup_path):
        # Add etcd backup details
        etcd_backup_dict = {'isEtcdBackupEnabled': 1,
                            'intervalInMins': backup_interval,
                            'storageType': 'local',
                            'storageProperties': {
                                'localPath': backup_path
                                }
                            }

        body = {'etcdBackup': etcd_backup_dict}
        log.info("Enable etcd backup on the cluster with body: {}".format(body))
        self.qbert.update_cluster(uuid, body)


    @retry(log=log, max_wait=360, interval=5)
    def wait_for_cluster_etcd_backup_complete(self, uuid, etcd_backup_location):
        log.info("wait_for_cluster_etcd_backup_complete")
        cluster = self.qbert.get_cluster_by_uuid(uuid)
        nodes = self.qbert.list_nodes()
        master_nodes = []
        # fetch master nodes in the cluster
        for node in list(nodes.values()):
            if node['clusterUuid'] == cluster['uuid'] and node['isMaster'] == 1:
                master_nodes.append(node)
        log.info("master_nodes list: {}".format(master_nodes))
        success = False
        for master_node in master_nodes:
            master_node_public_ip = master_node['primaryIp']
            with typical_fabric_settings(master_node_public_ip):
                res = run('ls {}'.format(etcd_backup_location))
                log.info('res of ls command on : {} is {}'.format(master_node_public_ip, res.__dict__))
                if res.succeeded:
                    lines = str(res).split()
                    if lines and len(lines) > 0:
                        log.info("Etcd backup files found in {} list: {}".format(
                            master_node_public_ip, lines))
                        success = True
                        break

        return success


def test_etcdbackup(qbert, du_fqdn, cluster_uuid):
    if os.getenv('SKIP_ETCD_BACKUP_TEST'):
        return

    validator = etcd_backup_validator(du_fqdn, qbert)
    validator.test_etcd_backup_on_cluster_error(cluster_uuid)
    validator.test_etcd_backup_on_cluster(cluster_uuid)