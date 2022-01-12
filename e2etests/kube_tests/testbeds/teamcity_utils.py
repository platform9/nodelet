# Copyright (c) Platform9 systems. All rights reserved

import requests
import logging
import urllib.request, urllib.parse, urllib.error
import tempfile
import xml.etree.ElementTree as ET
from subprocess import call

teamcity_url = "http://teamcity:8111/guestAuth"

LOG = logging.getLogger(__name__)

def download_kube_artifacts(build_id):
    url = "{0}/app/rest/builds/id:{1}/artifacts".format(teamcity_url, build_id)
    response = requests.get(url, verify=False)
    response.raise_for_status()
    assert 200 == response.status_code
    xml = ET.fromstring(response.content)
    artifact_path = tempfile.mkdtemp(suffix=build_id)
    for file in xml.findall("file"):
        file_name = file.attrib['name']
        if 'pf9-qbert' in file_name or 'pf9-kube' in file_name :
            file_url = "{0}/app/rest/builds/id:{1}/artifacts/content/{2}".\
                                    format(teamcity_url, build_id, file_name)
            urllib.request.urlretrieve(file_url, filename=file_name)
            call("mv ./*.rpm {0}".format(artifact_path), shell=True)
    return artifact_path

def get_latest_release_version():
    # defaults.json below is updated every release manually and
    # are the default options for a customer region deploy
    url = "https://mongo-prod.platform9.horse/etc/defaults.json"
    LOG.info("Querying snape for latest release version")
    response = requests.get(url, verify=False)
    response.raise_for_status()
    assert 200 == response.status_code
    latest_release = ""
    branch = ""
    for release in response.json()["releases"]:
        if (release > latest_release):
            latest_release = release
            branch = response.json()["releases"][release]["branch"]
    return latest_release, branch

def get_build_id(release_tag, branch):
    url = ("{0}/app/rest/builds/?locator=tags:{1},branch:{2},status:SUCCESS,"
           "project:Pf9project_Platform9ComponentsForKubernetes,pinned:true").\
                format(teamcity_url, release_tag, branch)
    LOG.info("Querying teamcity for latest release pinned build ID")
    response = requests.get(url, verify=False)
    response.raise_for_status()
    xml = ET.fromstring(response.content)
    try :
        build = next(xml.iterfind("build"))
        LOG.info("Artifacts will be downloaded from build id: %s", build);
    except:
        LOG.error("No teamcity build found for release " + release_tag)
        raise
    return build.attrib['id']

def download_rpm_from_last_release_build():
    latest_release_tag, branch = get_latest_release_version()
    LOG.info("Latest release is : %s and branch is: %s", latest_release_tag, branch)
    build_id = get_build_id(latest_release_tag, branch)
    artifacts_path = download_kube_artifacts(build_id)
    return artifacts_path
