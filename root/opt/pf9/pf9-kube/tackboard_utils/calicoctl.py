#!/opt/pf9/hostagent/bin/python

import getopt
import json
import logging
import os
import requests
import subprocess
import shlex
import sys
import base64

TACKBOARD_URL = "http://localhost:9080/tackboard/"
# Assumes kubeconfig path is fixed! Change if kubeconfig is moved
KUBECONFIG_PATH = "/etc/pf9/kube.d/kubeconfigs/admin.yaml"
CALICOCTL_CMDS = ['get', 'create', 'apply', 'replace', 'delete', 'patch',
                  'label']
CALICO_API_VERSION = "projectcalico.org/v3"
CALICOCTL_BIN = "/opt/pf9/pf9-kube/bin/calicoctl"

env = os.environ

log = logging.getLogger(__name__)

def remove_cluster_info(cfg):
    # We don't care about these values - but they throw an error if present
    # when applying, and are returned from a get when fetching all resources
    cfg["metadata"].pop("uid", None)
    cfg["metadata"].pop("resourceVersion", None)
    cfg["metadata"].pop("creationTimestamp", None)

def validate_resource_cfg(cfg):
    ret = ""

    if not isinstance(cfg, dict):
        return "Resource config is not a valid JSON dictionary. "
    if "kind" not in cfg:
        ret = "Missing resource type in kind field. "
    if "metadata" not in cfg or "name" not in cfg["metadata"]:
        ret += "Missing resource name in metadata section. "
    if "spec" not in cfg:
        ret += "Missing spec section. "
    return ret

def sanitize_resource_cfg(cfg):
    ret = validate_resource_cfg(cfg)
    if ret:
        return ret

    # For convenience, don't require API to send API version, or so things
    # don't break if user wants to use new API. Fill in if not specified
    if "apiVersion" not in cfg:
        cfg.update({ "apiVersion": CALICO_API_VERSION })

    remove_cluster_info(cfg)
    return False

def set_kubeconfig_env():
    global env
    env["DATASTORE_TYPE"] = "kubernetes"
    env["KUBECONFIG"] = KUBECONFIG_PATH

def handle_get(args):
    ''' Returns the config of one or more Calico resources

    Arguments: Takes in a list of resource type followed by zero or more
               resource names to get
    Returns: A tuple indicating True/False on error and
             message string with config or error message
    '''

    if len(args[1:]) < 1:
        log.error("Usage: get <TYPE> [ <NAMES...> ]")
        log.error("Requires a resource type followed by zero or more names")
        sys.exit(1)

    resource = args[1]
    names = args[2:]
    cmd_str = "%s get %s %s -o json --export" \
        % (CALICOCTL_BIN, resource, " ".join(names))

    cmd_str_list = shlex.split(cmd_str)
    proc = subprocess.Popen(cmd_str_list, stdout=subprocess.PIPE,
                            stderr=subprocess.STDOUT, env=env,
                            universal_newlines=True)
    stdout, stderr = proc.communicate()
    if proc.returncode:
        return (True, stdout)

    # The response is pretty-printed. Remove all newlines, indentation,
    # and whitespace separators to compress the response as it can be large
    # Also, --export which removes cluster specific fields does not work
    # when fetching all nodes. Remove this info, and because it throws an
    # error if present when applying/replacig config later
    try:
        json_cfg = json.loads(stdout)
    except (ValueError, TypeError) as err:
        # We redirected stderr to stdout, so if not JSON it's an error string
        response = stdout
        return (True, response)

    if "items" in json_cfg:
        for item in json_cfg["items"]:
            remove_cluster_info(item)
    else:
        remove_cluster_info(json_cfg)

    return (False, json_cfg)

def handle_create_update(args, cfg):
    ''' Creates, Replaces, or Apply (create if not exist, else replace)

    Arguments:
    args - A single list argument indicating calicoctl action to take
    cfg - Resource config to appply. Must be JSON python dict, not a string

    Returns: A tuple indicating True/False on error, and error message
    '''

    # create, replace, apply take in resource type and name in config to apply
    # Request can take in one resource, or list of resources to apply at once
    if "items" in cfg:
        for item in cfg["items"]:
            error = sanitize_resource_cfg(item)
            if error:
                return (True, error)
    else:
        error = sanitize_resource_cfg(cfg)
        if error:
            return (True, error)

    action = args[0]
    cmd_str = "%s %s -f -" % (CALICOCTL_BIN, action)
    cmd_str_list = shlex.split(cmd_str)
    input_stdin = json.dumps(cfg)
    log.info("Invoking %s\n\nWith input:\n\n%s", cmd_str, input_stdin)

    proc = subprocess.Popen(cmd_str_list, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, stdin=subprocess.PIPE,
                            env=env)
    stdout, stderr = proc.communicate(input=input_stdin)

    if proc.returncode or stderr:
        return (True, stderr)

    return (False, stdout)

def handle_delete(args):
    if len(args[1:]) < 1:
        return (True, "Usage: get <TYPE> [ <NAMES...> ]")

    resource = args[1]
    names = args[2:]

    cmd_str = "%s delete %s %s --skip-not-exists" \
        % (CALICOCTL_BIN, resource, " ".join(names))
    cmd_str_list = shlex.split(cmd_str)

    proc = subprocess.Popen(cmd_str_list, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE)
    stdout, stderr = proc.communicate()
    if proc.returncode or stderr:
        return(True, stderr)
    return (False, stdout)

def handle_patch(args, patch_cfg):
    ''' Patches a Calico resource

    The patch command is used to update a specific key in the resource's spec
    It applies a strategic merge patch. This is useful versus apply, which
    requires the entire resource config. Note that this should not be used to
    change metadata labels - use the label command for that

    Args: A list containging the [<RESOURCE> <NAME>]
    patch_cfg: JSON dict of the key-value pairs to update

    '''

    if len(args[1:]) < 2:
        return (True, "Usage: patch <TYPE> <NAMES ...>")

    if len(patch_cfg) == 0:
        return (True, "Patch missing JSON config to update the resource")

    resource = args[1]
    names = args[2:]

    # calicoctl patch handles only 1 resource name per patch operation
    # Our API need not be so restrictive. Run a patch cmd for each name
    for name in names:
        cmd_str = '%s patch %s %s --patch \'%s\'' \
            % (CALICOCTL_BIN, resource, name, patch_cfg)
        cmd_str_list = shlex.split(cmd_str)

        proc = subprocess.Popen(cmd_str_list, stdout=subprocess.PIPE,
                                stderr=subprocess.PIPE, env=env)
        stdout, stderr = proc.communicate()
        if proc.returncode or stderr:
            return (True, stderr)

    return (False, stdout)

def handle_label(args, label_cfg, overwrite=False, remove=False):
    ''' Adds a metadata label to Calico resource

    This is convenience operation that could also be achieved using patch or
    apply. Rather than having to specify a JSON input, it takes in a key=value
    pair as an argument on the command line. It also supports flags opts to
    overwrite or remove a particular label. Default is to error if key exists

    Arguments: A list containing: [<resource type>, <resource name>, key=value]
    '''
    if len(args[1:]) <  2:
        return (True, 'Usage: label <TYPE> <NAME> -d \'{"key":"value")\'')

    if not isinstance(label_cfg, dict):
        return (True, 'Label config must be in format { "<key>" : "<value>" })')

    resource = args[1]
    name = args[2]
    key = list(label_cfg.keys())[0]
    val = list(label_cfg.items())[0][1]
    if remove:
        label = key
    else:
        label = "%s=%s" % (key, val)

    cmd_str = "%s label %s %s %s" % (CALICOCTL_BIN, resource, name, label)
    cmd_str += " %s" % (" --remove" if remove else "")
    cmd_str += " %s" % (" --overwrite" if overwrite else "")
    cmd_str_list = shlex.split(cmd_str)

    proc = subprocess.Popen(cmd_str_list, stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, env=env)
    stdout, stderr = proc.communicate()
    if proc.returncode or stderr:
        return (True, stderr)

    return (False, stdout)

def send_tackboard_resp(uuid, resp_data):
    url = TACKBOARD_URL
    uuid_header = {'uuid': uuid}
    os.environ['no_proxy'] = "localhost"
    try:
        resp = requests.post(url, json=resp_data, headers=uuid_header)
        resp.raise_for_status()
    except Exception as e:
        log.info("Failed to send tackboard response: %s", str(e))

    return

def configure_logging():
    fh = logging.FileHandler("/var/log/pf9/calicoctl.log")
    fmt = logging.Formatter("%(asctime)s : %(message)s")
    fh.setFormatter(fmt)
    log.addHandler(fh)
    log.setLevel(logging.DEBUG)

def main():
    err = False
    status = "success"
    action = None
    resource_cfg = {}
    overwrite_label = False
    remove_label = False
    configure_logging()

    try:
        opts, args = getopt.gnu_getopt(sys.argv[1:], "hd:", ["help", "data=", "skip-not-exists", "overwrite", "remove"])
    except getopt.GetoptError as exc:
        resp = str(exc)
        err = True

    for opt, val in opts:
        if opt in ["-d", "data"]:
            try:
                resource_cfg = base64.b64decode(val)
                resource_cfg_json = json.loads(resource_cfg)
            except (ValueError, TypeError) as exc:
                resp = "Resource Config must be a valid JSON"
                err = True
        if opt == "overwrite":
            overwrite_label = True
        if opt == "remove":
            remove_label = True

    # qbert tackboard appends a UUID as last arg of every remote cmd
    tackboard_uuid = args.pop()

    if len(args) < 1 or (args[0] not in CALICOCTL_CMDS):
        log.error("Usage: First argument must be one of %s", CALICOCTL_CMDS)
        sys.exit(1)

    set_kubeconfig_env()
    action = args[0]
    if action == "get":
        err, resp = handle_get(args)
    elif action == "patch":
        err, resp = handle_patch(args, resource_cfg)
    elif action == "label":
        err, resp = handle_label(args, resource_cfg_json, overwrite=overwrite_label, remove=remove_label)
    elif action == "delete":
        err, resp = handle_delete(args)
    elif action in ["create", "replace", "apply"]:
        err, resp = handle_create_update(args, resource_cfg_json)

    if err:
        log.error("Command Failed!!!")
        status = "failed"
    log.info("Sending response:\n%s", resp)

    resp = json.dumps(resp, separators=(',', ':'))

    resp_data = {'status': status, 'calico_resp': resp}
    send_tackboard_resp(tackboard_uuid, resp_data)
    sys.exit(err)

if __name__ == '__main__':
    main()

