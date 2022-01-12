set -x
sysctl net.bridge.bridge-nf-call-iptables
cat /usr/lib/sysctl.d/00-system.conf
cat /etc/sysctl.conf
ss -lptn
ip addr
ip route
brctl show
getenforce
ps -ef|grep pf9
ps -ef|grep containerd
id pf9
grep pf9 /etc/passwd
ls -ld /opt/pf9
ls -l /opt/pf9
service firewalld status
iptables -L
iptables -L -t nat
service ebtables status
ebtables -L
ebtables -t nat -L
ebtables -t broute -L
ps -ef
rpm -qa

