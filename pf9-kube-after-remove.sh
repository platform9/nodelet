echo "Post removal script of pf9-kube package"
# Some of the files e.g. /opt/pf9/pf9-kube/bin/requester/easy-rsa-master are not owned by pf9-kube package and hence need to be removed separately.
# dpkg -S /opt/pf9/pf9-kube/bin/requester/easy-rsa-master/
#   dpkg-query: no path found matching pattern /opt/pf9/pf9-kube/bin/requester/easy-rsa-master/
rm -rf /opt/pf9/pf9-kube