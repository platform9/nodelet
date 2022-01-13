#!/opt/pf9/python/bin/python

# If pf9-comms is configured to use a proxy, returns the configuration as the
# string "http://<user>:<pass>@<host>:<port>".  If pf9-comms is not configured
# to use a proxy, returns an empty string.  If host is undefined or an empty
# string, or if only one of user/pass is defined, or both user/pass are defined
# but one is an empty string, the configuration is malformed. When the
# configuration is malformed, returns an empty string to be compatible with
# pf9-comms, which ignores a malformed configuration.  If unable to open or
# read the configuration file, returns an empty string.
# Example configuration:
# {
#     "http_proxy": {
#         "host": "<host>",
#         "port": "<port>",
#         "protocol": "<protocol>",
#         "pass": "<password>",
#         "user": "<user>"
#     }
# }
import json, sys
try:
    with open (sys.argv[1]) as cfg_file:
        j = json.load(cfg_file)
        url = ''

        if j['http_proxy'].get('protocol'):
            url += '%s://' % j['http_proxy']['protocol']
        else:
            url += 'http://'

        if j['http_proxy'].get('user') and j['http_proxy'].get('pass'):
            url += '%s:%s@' % (j['http_proxy']['user'], j['http_proxy']['pass'])
        elif j['http_proxy'].get('user') or j['http_proxy'].get('pass'):
            # only one of user/pass is defined, or both are defined, but
            # one is an empty string -> malformed configuration
            exit()

        if j['http_proxy'].get('host'):
            url += j['http_proxy']['host']
        else:
            # undefined or empty host -> malformed configuration
            exit()

        if j['http_proxy'].get('port'):
            url += ':%s' % j['http_proxy']['port']

        print(url)
except:
    # probably could not open or read file
    exit()
