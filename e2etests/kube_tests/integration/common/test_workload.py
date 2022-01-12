# Copyright (c) Platform9 systems. All rights reserved

import logging
from os import path
from requests.models import HTTPError

import yaml
from kubernetes import client, config
from pf9lab.retry import retry
from kube_tests.lib.kubeconfig import get_kubeconfig
from kube_tests.lib.kubernetes import Kubernetes

log = logging.getLogger(__name__)

def test_add_workload(k8sClient):
    with open(path.join(path.dirname(__file__), "nginx-deployment.yaml")) as f:
        dep = yaml.safe_load(f)
        resp = k8sClient.create_deployment(template=dep, namespace="default")
        print("Deployment created. status='%s'" % str(resp.status_code))


@retry(log=log, max_wait=120, interval=5)
def test_verify_workload_exists(k8sClient):
    """
    :return: bool True if the workload exists, False if it doesn't
    """
    resp = k8sClient.get_deployment(namespace="default", name="nginx-deployment")
    deployment = resp.json()
    if str(deployment['metadata']['name']) == "nginx-deployment":
        return True
    return False


@retry(log=log, max_wait=120, interval=5)
def test_verify_workload_does_not_exist(k8sClient):
    """
    :return: bool True if the workload does not exist, False if it does
    """
    try:
        resp = k8sClient.get_deployment(namespace="default", name="nginx-deployment")
    except HTTPError as error: 
        if (error.response.status_code == 404):
            return True

        raise error
    return False


def test_delete_workload(k8sClient):
    resp = k8sClient.delete_deployment(namespace="default", name="nginx-deployment")
    print(resp.status_code)
