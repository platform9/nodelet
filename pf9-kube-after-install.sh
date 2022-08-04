echo 'after pf9-kube installation'

mkdir -p /opt/pf9/home
groupadd pf9group || true
useradd -d /opt/pf9/home -G pf9group pf9 || true

chmod 0440 /etc/sudoers.d/pf9-kube
chmod 0440 /etc/logrotate.d/pf9-kube
chmod 0440 /etc/logrotate.d/pf9-kubelet
chmod 0440 /etc/logrotate.d/pf9-nodeletd
chmod 0770 /opt/pf9/hostagent/extensions/fetch_pf9_kube_status.py
chmod 0770 /opt/pf9/hostagent/extensions/fetch_pod_info.py
chmod 0644 /etc/cron.d/pf9-logrotate
chmod 0755 /etc/cron.pf9/logrotate
mkdir -p /var/log/pf9/kube
mkdir -p /etc/pf9/kube.d
mkdir -p /etc/cni/net.d
mkdir -p /var/log/pf9/kubelet
mkdir -p /var/opt/pf9/kube/apiserver-config

# to store containerd installation tar/zip files
mkdir -p /opt/pf9/pf9-kube/containerd

# Enable calling iptables for packets ingress/egress'ing bridges
# See IAAS-7747
sysctl net/bridge/bridge-nf-call-iptables > /etc/pf9/kube.d/bridge-nf-call-iptables.old
echo net.bridge.bridge-nf-call-iptables = 1 >> /etc/sysctl.conf
sysctl -p

chown -R pf9:pf9group /var/log/pf9
chown -R pf9:pf9group /var/log/pf9/kube
chown -R pf9:pf9group /opt/pf9/pf9-kube
chown -R pf9:pf9group /var/log/pf9/kubelet
chown -R pf9:pf9group /opt/pf9/hostagent/extensions/fetch_pf9_kube_status.py
chown -R pf9:pf9group /opt/pf9/hostagent/extensions/fetch_pod_info.py
chown -R pf9:pf9group /var/opt/pf9
chown -R pf9:pf9group /etc/pf9

# Clear any docker network configuration before installation of
# any network plugin.
# See: https://platform9.atlassian.net/browse/IAAS-7740
# If docker networking was configured even once from the beginning
# of history, it will persist in the local db present under
# the below directory. Remove it opportunistically
rm -rf /var/lib/docker/network || true

# 1.18.10-pmk.1513 was a DoA release which can lead to improper clean up during node upgrade.
# This prevents the pf9-kube "service" from start successfully post upgrade. Invoke a force stop
# to attempt a clean up prior to hostagent starting nodelet.
# Side-effects: Certificates will now always be rotated on a node upgrade. Prior to this commit if
# the certificates had not expired and were generated for the same cluster UUID and role then they
# would not have been re-generated.
# TODO: Remove this for the 5.2 equivalent pf9-kube releases.
/opt/pf9/nodelet/nodeletd phases stop --force || true

# PMKFT: Make it easier to run "kubectl" commands after adding a master
ln -sf /opt/pf9/pf9-kube/bin/kubectl /usr/local/bin/kubectl

# clean up existing rotated log files
rm -f /var/log/pf9/kubelet/*.gz

# Remove the older kube interface cache file
# When implementing interface selection from qbert/sunpike API these cache files can be removed.
if [ -f /var/opt/pf9/kube_interface ]; then
    grep 'V4_INTERFACE' /var/opt/pf9/kube_interface > /var/opt/pf9/kube_interface_v4
    grep 'V6_INTERFACE' /var/opt/pf9/kube_interface > /var/opt/pf9/kube_interface_v6
    if [ -f /var/opt/pf9/kube_interface_v4] ; then
        chown pf9:pf9group /var/opt/pf9/kube_interface_v4
    fi
    if [ -f /var/opt/pf9/kube_interface_v6] ; then
        chown pf9:pf9group /var/opt/pf9/kube_interface_v6
    fi
    rm -f /var/opt/pf9/kube_interface || true
fi

# Writing polkit rules to allow pf9 service to manage units. 
# e.g. containerd (required to get authentication to connect to system-dbus for operations like start/stop/status/enable ) 
echo "Generating polkit rules"
mkdir -p /etc/polkit-1/rules.d
cat > "/etc/polkit-1/rules.d/48-pf9-allow-service-mgmt.rules" <<EOF
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.systemd1.reload-daemon" &&
        subject.user == "pf9")
    {
        return polkit.Result.YES;
    }
})
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.systemd1.manage-units" &&
        subject.user == "pf9")
    {
        return polkit.Result.YES;
    }
})
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.systemd1.manage-unit-files" &&
        subject.user == "pf9")
    {
        return polkit.Result.YES;
    }
})
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.udisks2.filesystem-take-ownership" &&
        subject.user == "pf9")
    {
        return polkit.Result.YES;
    }
})
EOF


# Make all pf9-kube files non-writable by pf9 user
# To prevent files from being written using vim + :wq! make the root user owner of all files
chown -R root:pf9group /opt/pf9/pf9-kube || true
# Remove write permissions
chmod -w -R /opt/pf9/pf9-kube || true
# Add write and execute permissions /opt/pf9/pf9-kube/conf to allow templates to be rendered
chmod 0770 -R /opt/pf9/pf9-kube/conf/
# Add write and execute permissions /opt/pf9/pf9-kube/containerd to allow installation tar/zips to be stored and extracted.
chmod 0770 -R /opt/pf9/pf9-kube/containerd

