import json
import logging
import os
import sys

if sys.version_info[0] == 2:
    from backports.ssl_match_hostname import match_hostname
    from requests.packages.urllib3 import connection

    # See IAAS-6208 for details of why the SSL backport is needed
    connection.match_hostname = match_hostname

from requests import HTTPError, Session
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

log = logging.getLogger(__name__)
request_timeout = int(os.getenv('HTTP_REQUEST_TIMEOUT_IN_SECS', '180'))


def session_with_retries(host, max_retries=10):
    session = Session()
    http_statuses_to_retry = [
        502,  # Bad Gateway
        503,  # Service Unavailable
        504  # Gateway Timeout
    ]
    retries = Retry(total=max_retries, backoff_factor=1.0,
                    status_forcelist=http_statuses_to_retry)
    # HTTPAdapter's `max_retries` takes either an integer, or Retry object
    session.mount(host, HTTPAdapter(max_retries=retries))
    return session


def make_req(session, endpoint, method, body):
    resp = session.request(method, endpoint, json=body, verify=False,
                           timeout=request_timeout)
    log.debug('%s %s - %s', method, endpoint, resp.status_code)
    try:
        resp.raise_for_status()
        return resp
    except HTTPError as error:
        log.debug('HTTP Error. Full response text: %s', error.response.text)
        raise error
    finally:
        _log_req_and_resp(resp)


def _log_req_and_resp(resp):
    try:
        req_body = json.dumps(json.loads(resp.request.body), indent=4)
    except (ValueError, TypeError):
        req_body = resp.request.body
    try:
        resp_body = json.dumps(resp.json(), indent=4)
    except (ValueError, TypeError):
        resp_body = resp.text

    msg = '\n'.join(str(item) for item in ['HTTP Session log',
                                           '------- Request: -------',
                                           req_body,
                                           '------- Response: ------',
                                           resp_body])
    log.debug(msg)
