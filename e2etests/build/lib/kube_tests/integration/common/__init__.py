# Copyright (c) Platform9 systems. All rights reserved

import logging
import os
from requests.exceptions import RequestException
from pf9lab.retry import retry

log = logging.getLogger(__name__)

@retry(log=log, max_wait=1200, interval=20, tolerate_exceptions=False)
def wait_for_cluster_attr(qbert, cluster_uuids, attr_name, expected_val,
                          untolerable_vals=[]):
    """
    :param qbert: Object from Qbert.py
    :param cluster_uuids: list of string UUIDs
    :param attr_name: attribute name to check for expected value
    :param expected_val: expected value for the attribute
    :param untolerable_vals: array of values that shouldn't be tolerated, error out immediately
    :return: True if all clusters have expected attribute value, False otherwise
    """

    for cluster_uuid in cluster_uuids:
        try:
            cluster = qbert.get_cluster_by_uuid(cluster_uuid)
            attr_val = cluster[attr_name]
            log.info('cluster %s has %s %s; waiting for %s', cluster_uuid, attr_name, attr_val, expected_val)
            if attr_val in untolerable_vals:
                # Do not retry
                raise RuntimeError("cannot tolerate cluster attribute %s with value: %s", attr_name, attr_val)
            if attr_val != expected_val:
                return False
        except RequestException as exc:
            log.info("qbert api error: %s", exc.message)
            return False
    # looked at all uuids, all reported expected_status
    return True


@retry(log=log, max_wait=2400, interval=20, tolerate_exceptions=False)
def wait_for_cluster_taskstatus(expected_status, qbert, cluster_uuids):
    """
    :param expected_status: one of ['creating', 'converging', 'updating', 'success]
    :param qbert: Object from Qbert.py
    :param cluster_uuids: list of string UUIDs
    :return: True if all clusters have expected_status, False otherwise
    """
    for cluster_uuid in cluster_uuids:
        try:
            cluster = qbert.get_cluster_by_uuid(cluster_uuid)
            task_status = cluster['taskStatus']
            log.info('cluster %s has taskStatus %s; waiting for %s', cluster_uuid, task_status, expected_status)
            if task_status == 'error':
                # Do not retry
                raise RuntimeError("cluster task resulted in error: %s", cluster['taskError'])
            if task_status != expected_status:
                return False
        except RequestException as exc:
            log.info("qbert api error: %s", exc.message)
            return False
    # looked at all uuids, all reported expected_status
    return True


@retry(log=log, max_wait=2400, interval=1, tolerate_exceptions=False)
def wait_for_cluster_status(expected_status, qbert, cluster_uuids):
    for cluster_uuid in cluster_uuids:
        try:
            cluster = qbert.get_cluster_by_uuid(cluster_uuid)
            status = cluster['status']
            log.info('cluster %s has status %s', cluster_uuid, status)
            if status != expected_status:
                # This cluster is not yet in the state we expect it to be.
                # The method expects all clusters to be this state. Hence,
                # we need to retry. Return False
                return False
        except RequestException as exc:
            log.info("qbert api error: %s", exc.message)
            return False

    # Reached here only after all the clusters have the expected status.
    # Else, we would have returned in one of the earlier code paths
    return True


@retry(interval=30, log=log, tolerate_exceptions=False)
def wait_until_node_absent(qbert, cluster_uuid, node_name):
    nodes = qbert.list_nodes()
    return not next((node for node in nodes
                     if nodes[node]['clusterUuid'] == cluster_uuid and
                     node == node_name), None)


@retry(max_wait=2400, interval=30, log=log, tolerate_exceptions=False)
def wait_until_cluster_size(qbert, cluster_uuid, expected_cluster_size):
    actual_cluster_size = get_cluster_size(qbert, cluster_uuid)
    if actual_cluster_size == expected_cluster_size:
        return True
    log.info('Waiting for cluster size {0}. Current cluster size is {1}'
             .format(expected_cluster_size, actual_cluster_size))
    return False


def get_cluster_size(qbert, cluster_uuid):
    nodes = qbert.list_nodes()
    return sum(1 for node in nodes
               if nodes[node]['clusterUuid'] == cluster_uuid)


class MissingEnvError(Exception):
    def __init__(self, env_var_name):
        self.env_var_name = env_var_name

    def __str__(self):
        return self.env_var_name + ' environment variable is required but missing'


class InvalidEnvValueError(Exception):
    def __init__(self, env_var_name, env_var_value, possible_values):
        self.env_var_name = env_var_name
        self.env_var_value = env_var_value
        self.possible_values = possible_values

    def __str__(self):
        return "'{}' has invalid value '{}'. List of possible values: {}".format(
            self.env_var_name,
            self.env_var_value,
            self.possible_values
        )


class EnvTuple(object):
    def __init__(self, env_var_name, possible_values=None):
        self.env_var_name = env_var_name
        self.env_var_value = os.getenv(env_var_name)
        self.possible_values = possible_values

    def __str__(self):
        return "({}, {}, {})".format(self.env_var_name,
                                     self.env_var_value,
                                     self.possible_values)


def ensure_env_set(env_tuple):
    if env_tuple.env_var_value is None:
        raise MissingEnvError(env_tuple.env_var_name)
    if env_tuple.possible_values is not None \
            and type(env_tuple.possible_values) is list \
            and len(env_tuple.possible_values) > 0:
        if env_tuple.env_var_value not in env_tuple.possible_values:
            raise InvalidEnvValueError(env_tuple.env_var_name,
                                       env_tuple.env_var_value,
                                       env_tuple.possible_values)

