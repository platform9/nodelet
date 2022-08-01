# Make pf9-kube files writable again
chown -R pf9:pf9group /opt/pf9/pf9-kube
chmod +w -R /opt/pf9/pf9-kube

systemctl stop pf9-nodeletd
if /opt/pf9/nodelet/nodeletd phases stop --force; then
    echo "pf9-kube stopped successfully before uninstallation"
else
    echo "pf9-kube could not be stopped before uninstallation"
fi
rm -f /etc/pf9/nodelet/config_*.yaml

# Revert old sysctl settings
cat /etc/pf9/kube.d/bridge-nf-call-iptables.old >> /etc/sysctl.conf || true
sysctl -p
rm -f /etc/pf9/kube.d/bridge-nf-call-iptables.old || true
rm -f /var/opt/pf9/kube_status || true

if [ -f /var/opt/pf9/kube_interface_v4 ]; then
    echo "Platform9 K8s v4 interface cache file present and won't be removed."
fi

if [ -f /var/opt/pf9/kube_interface_v6 ]; then
    echo "Platform9 K8s v6 interface cache file present and won't be removed."
fi

# PMKFT: Remove symlinked kubectl
rm -rf /usr/local/bin/kubectl || true
/usr/bin/cgdelete -g cpu:pf9-kube-status || true

# Uninstall/remove containerd and runc installed by nodeletd
rm -f /usr/local/bin/containerd*
rm -f /usr/local/sbin/runc*
rm -rf /opt/cni/bin

rm -f /usr/local/lib/systemd/system/containerd.service
rm -f /etc/containerd/config.toml
