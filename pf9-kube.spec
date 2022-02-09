Name:           pf9-kube
Version:        %{_version}
Release:        %{_release}
Summary:        Platform9 Kubernetes Agent
License:        Commercial
URL:            http://www.platform9.net
Provides:       pf9-kube
Provides:       pf9app
Requires:       curl
Requires:       gzip
Requires:       net-tools
Requires:       socat
Requires:       keepalived
Requires:       libcgroup-tools
AutoReqProv:    no

%global __os_install_post %(echo '%{__os_install_post}' | sed -e 's!/usr/lib[^[:space:]]*/brp-python-bytecompile[[:space:]].*$!!g')

%description
Platform9 Kubernetes Agent

%prep

%build

%install
SRC_DIR=%_src_dir

rm -fr $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT
cp -r $SRC_DIR/* $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/var/log/pf9/kube
mkdir -p $RPM_BUILD_ROOT/var/log/pf9/kubelet
mkdir -p $RPM_BUILD_ROOT/etc/pf9/kube.d
mkdir -p $RPM_BUILD_ROOT/etc/cni/net.d
mkdir -p $RPM_BUILD_ROOT/opt/cni/bin

%clean
rm -rf $RPM_BUILD_ROOT

%files
%defattr(-,pf9,pf9group,-)
%attr(0440, root, root) /etc/sudoers.d/pf9-kube
%attr(0440, root, root) /etc/sudoers.d/pf9-nodelet
%attr(0644, root, root) /etc/cron.d/pf9-logrotate
/opt/pf9/pf9-kube/
/opt/cni/bin/
/opt/pf9/nodelet/nodeletd
/lib/systemd/system/pf9-nodeletd.service

%attr(0440, root, root) /etc/logrotate.d/pf9-kube
%attr(0440, root, root) /etc/logrotate.d/pf9-kubelet
%attr(0440, root, root) /etc/logrotate.d/pf9-nodeletd
# Make the extension read-write-executable by pf9group
%dir /var/log/pf9/
%dir /var/log/pf9/kube/
%dir /var/log/pf9/kubelet/
%dir /etc/pf9/kube.d/
%dir /etc/cni/net.d/
%dir /opt/cni/bin

%post
## Adding ownership tweak (PMK-4129) to %post section instead of %files section since it's not sure if this file exists or not.
## Check for existence of file is NOT appropriate in %files section.
if [ -e /var/opt/pf9/kube_interface ] ; then
    chown pf9:pf9group /var/opt/pf9/kube_interface
fi
kube_override=/etc/pf9/kube_override.env
if [ -f "$kube_override" ]; then
    source $kube_override
fi
# $1==1 Initial installation
# $1==2 Upgrade
if [ "$1" == 1 ]; then

    # 1.18.10-pmk.1513 was a DoA release which can lead to improper clean up during node upgrade.
    # This prevents the pf9-kube "service" from start successfully post upgrade. Invoke a force stop
    # to attempt a clean up prior to hostagent starting nodelet.
    # Side-effects: Certificates will now always be rotated on a node upgrade. Prior to this commit if
    # the certificates had not expired and were generated for the same cluster UUID and role then they
    # would not have been re-generated.
    # TODO: Remove this for the 5.2 equivalent pf9-kube releases.
    /opt/pf9/nodelet/nodeletd phases stop --force || true

    # Clear any docker network configuration before installation of any
    # network plugin.
    # See: https://platform9.atlassian.net/browse/IAAS-7740
    # If docker networking was configured even once from the beginning
    # of history, it will persist in the local db present under
    # the below directory. Remove it opportunistically
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        rm -rf /var/lib/docker/network || true
    fi

    # Remove Docker's local storage information. There is a Docker issue where
    # the devicemapper storage driver by default uses loopback mode
    # (metadata and data files are loopback-mounted sparse files). Restarting
    # Docker using direct-lvm (thin-pool) mode corrupts /var/lib/docker and the
    # Docker daemon no longer starts.
    # Note: only need to do this if host has hit this issue.
    # See: https://platform9.atlassian.net/browse/PMK-917
    if [ "$PF9_MANAGED_DOCKER" != "false" ]; then
        docker_started_in_loopback_mode=$(grep '"storage-opts": \[ \]' /etc/docker/daemon.json)
        if [ "$docker_started_in_loopback_mode" ] && vgs docker-vg >/dev/null 2>&1; then
            rm -rf /var/lib/docker || true
        fi
    fi

    # Enable calling iptables for packets ingress/egress'ing bridges
    # See IAAS-7747
    sysctl net/bridge/bridge-nf-call-iptables > /etc/pf9/kube.d/bridge-nf-call-iptables.old
    echo net.bridge.bridge-nf-call-iptables = 1 >> /etc/sysctl.conf
    sysctl -p
    systemctl daemon-reload

    # PMKFT: Make it easier to run "kubectl" commands after adding a master
    ln -s /opt/pf9/pf9-kube/bin/kubectl /usr/local/bin/kubectl
    # clean up existing rotated log files
    rm -f /var/log/pf9/kubelet/*.gz

    # When implementing selection of interface in qbert/sunpike API, these cache files can be removed.
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

    # Make all pf9-kube files non-writable by pf9 user
    # To prevent files from being written using vim + :wq! make the root user owner of all files
    chown -R root:pf9group /opt/pf9/pf9-kube || true
    # Remove write permissions
    chmod -w -R /opt/pf9/pf9-kube || true
    mkdir -p /var/opt/pf9/kube/apiserver-config
    chown -R pf9:pf9group /var/opt/pf9/kube/apiserver-config
fi

%preun
systemctl stop pf9-nodeletd
# $1==0: remove the last version of the package
# $1==1: install the first time
# $1>=2: upgrade
if [ "$1" == 0 ]; then
    # Make pf9-kube files writable again
    chown -R pf9:pf9group /opt/pf9/pf9-kube
    chmod +w -R /opt/pf9/pf9-kube
    /opt/pf9/nodelet/nodeletd phases stop --force || true
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
fi
/usr/bin/cgdelete -g cpu:pf9-kube-status || true

%postun
# $1==0: remove the last version of the package
# $1==1: install the first time
# $1>=2: upgrade
if [ "$1" == 0 ]; then
    # Some of the files e.g. /opt/pf9/pf9-kube/bin/requester/easy-rsa-master are not owned by pf9-kube package and hence need to be removed separately.
    # rpm -q --whatprovides /opt/pf9/pf9-kube/bin/requester/easy-rsa-master/
    #   file /opt/pf9/pf9-kube/bin/requester/easy-rsa-master is not owned by any package
    rm -rf /opt/pf9/pf9-kube
fi
%changelog
