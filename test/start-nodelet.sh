#!/usr/bin/env bash
IP_ADDRESS=${1:-172.17.0.2}
apt-get -y update
apt-get -y install sudo lsb-core neovim kmod iptables
groupadd pf9group
useradd -d /opt/pf9/home -G pf9group pf9
mkdir -p /etc/pf9
mkdir -p /var/opt/pf9/images
chown -R pf9:pf9 /etc/pf9
chown -R pf9:pf9 /var/opt/pf9
echo "Adding override for cgroup driver of containerd"
echo "export CONTAINERD_CGROUP=cgroupfs" > /etc/pf9/kube_override.env
sed  s/__NODE_IP_ADDRESS__/$IP_ADDRESS/ /work/test/nodelet.yaml.tmpl > /work/test/nodelet.yaml
/work/nodeletctl/nodeletctl --verbose create --config /work/test/nodelet.yaml 
/opt/pf9/pf9-kube/bin/kubectl --kubeconfig /etc/pf9/kube.d/kubeconfigs/admin.yaml get pods -A