# Copyright (c) Platform9 systems. All rights reserved

import logging
import os

from kube_tests.lib import dict_utils
from kube_tests.lib import request_utils

log = logging.getLogger(__name__)


class Resmgr(object):
    def __init__(self, token, api_url):
        if not (token and api_url):
            raise ValueError('need a keystone token and API url')
        if api_url[-1] == '/':
            raise ValueError('API url must not have trailing slash')
        self.api_url = api_url
        session = request_utils.session_with_retries(self.api_url)
        session.headers = {'X-Auth-Token': token,
                           'Content-Type': 'application/json'}
        self.session = session

    def _make_req(self, endpoint, method='GET', body={}):
        return request_utils.make_req(self.session, self.api_url + endpoint,
                                      method, body)

    def get_all_hosts(self):
        log.info('Getting all hosts from resmgr')
        endpoint = '/hosts'
        resp = self._make_req(endpoint)
        return resp.json()

    def get_role(self, role_name):
        log.info('Getting info about the role %s from resmgr', role_name)
        endpoint = '/roles/{0}'.format(role_name)
        resp = self._make_req(endpoint)
        return resp.json()
