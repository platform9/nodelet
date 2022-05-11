# Nodeletctl

Nodeletctl is a CLI and Golang pkg to deploy and manage Nodelet based clusters. It is completely DU-less, SSH/push based (does not Reconcile/pull/sync via the host-to-DU like current SaaS model), and DB-less (uses input config file + local state dir), It can generate its own root CA and key to distribute to nodes, or use a user supplied CA. It deploys nodelet in standalone mode (no sync with sunpike)

Nodeletctl can be run from any machine, as long it has inbound SSH based access to the target nodes. It can deploy on the local machine itself, as well as be a temporary bastion VM, if the cluster YAML and certs are backed up.

## Pre-reqs

#### SSH keys
Ensure you have SSH access working to each remote node. The SSH user and keypath are specified in the cluster config file

### Nodelet RPM
The deployer expects the nodelet RPM in a .tar.gz format. The location to the nodelet package is specified in the cluster config file.
https://github.com/platform9/nodelet/releases

## Local state

When creating(or scaling) a cluster, nodeletctl will generate each node's nodelet config before uploading to each remote machine. It will store it locally at:

    /etc/nodelet/CLUSTER_NAME/NODENAME/config_sunpike.yaml
    
To sync each node's details it will also pull in each node's kube_status.json and save it at:

    /etc/nodelet/CLUSTER_NAME/NODENAME/kube_status.json
    
Unless provided by the user, nodeletctl will also generate a root CA and private key to distribute to each node, and save it locally at:

    /etc/nodelet/CLUSTER_NAME/certs/
    
After successful creation, it will also generate an admin user keypair and Kubeconfig file located in the same certs directory. It is recommended to backup this folder. Assuming my cluster is named "airctl-mgmt":
```
[root@arjunairdu certs]# pwd
/etc/nodelet/airctl-mgmt/certs
[root@arjunairdu certs]# ls -alh
total 28K
drwxr-xr-x.  2 root root  107 May  5 18:41 .
drwxr-xr-x. 10 root root  195 May  5 18:46 ..
----------.  1 root root 1.9K May  5 21:37 adminCert.pem
-rw-r--r--.  1 root root 3.2K May  5 21:37 adminKey.pem
-rw-r--r--.  1 root root 9.6K May  5 21:37 admin.kubeconfig
----------.  1 root root 1.9K May  5 18:40 rootCA.crt
-rw-r--r--.  1 root root 3.2K May  5 18:40 rootCA.key
```

## How to build
Clone this repo and cd to the nodeletctl directory
```
GOPRIVATE=github.com/platform9/* go build -o nodeletctl
```
## Cluster operations
```
[root@arjunairdu ~]# ./nodeletctl 
nodeletctl is a cluster manager to deploy and configure nodelets on remote machines

Usage:
  nodeletctl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  create      Create a nodelet based cluster
  delete      Delete a nodelet based cluster
  help        Help about any command
  scale       Scale up/down a nodelet based cluster

Flags:
      --config string   config file (default is $HOME/nodeletCluster.yaml) (default "/root/nodeletCluster.yaml")
  -h, --help            help for nodeletctl
      --json            json output for commands (configure-hosts only currently)
      --verbose         print verbose logs to the console

Use "nodeletctl [command] --help" for more information about a command.
```

 - Create a cluster:
	 - ```./nodeletctl create --config <path to cluster.yaml>```
 - Delete a cluster:
	 - ```./nodeletctl delete --config <path to cluster.yaml>```
 - Scale a cluster:
	 - ```./nodeletctl scale --config <path to cluster.yaml>```
	 - The scale operation takes in the final, desired state of the cluster. You may scale up or down, both master and worker nodes in one operation. Behind the scenes, it will calculate the desired nodes to add and remove, scaling up the masters first, followed by scaling down masters serially. It will then scale up the workers in parallel, followed by scaling down workers.

It is imperative to keep your cluster YAML up to date as this is the "source of truth" for your desired cluster. For convenience, after each cluster operation, nodeletctl will generate and save a new copy to the local cluster state directory:

```/etc/nodelet/<CLUSTER_NAME>/<CLUSTER_NAME.yaml```

## Configuration options

Nodeletctl takes in a --config file that will be Unmarshall'd into the following structure:

Here is the full set of configuration options:
```
type BootstrapConfig struct {
	SSHUser                string                 `json:"sshUser,omitempty"`
	SSHPrivateKeyFile      string                 `json:"sshPrivateKeyFile,omitempty"`
	CertsDir               string                 `json:"certsDir,omitempty"`
	KubeConfig             string                 `json:"kubeconfig,omitempty"`
	Pf9KubePkg             string                 `json:"nodeletPkg,omitempty"`
	ClusterId              string                 `json:"clusterName,omitempty"`
	AllowWorkloadsOnMaster bool                   `json:"allowWorkloadsOnMaster,omitempty"`
	K8sApiPort             string                 `json:"k8sApiPort,omitempty"`
	MasterIp               string                 `json:"masterIp,omitempty"`
	MasterVipEnabled       bool                   `json:"masterVipEnabled,omitempty"`
	MasterVipInterface     string                 `json:"masterVipInterface,omitempty"`
	MasterVipVrouterId     int                    `json:"masterVipVrouterId,omitempty"`
	CalicoV4Interface      string                 `json:"calicoV4Interface,omitempty"`
	CalicoV6Interface      string                 `json:"calicoV6Interface,omitempty"`
	MTU                    string                 `json:"mtu,omitempty"`
	Privileged             string                 `json:"privileged,omitempty"`
	ContainerRuntime       ContainerRuntimeConfig `json:"containerRuntime,omitempty"`
	MasterNodes            []HostConfig           `json:"masterNodes"`
	WorkerNodes            []HostConfig           `json:"workerNodes"`
}

type ContainerRuntimeConfig struct {
	Name         string `json:"name,omitempty"`
	CgroupDriver string `json:"cgroupDriver,omitempty"`
}

type HostConfig struct {
	NodeName            string  `json:"nodeName"`
	NodeIP              *string `json:"nodeIP,omitempty"`
	V4InterfaceOverride *string `json:"calicoV4Interface,omitempty"`
	V6InterfaceOverride *string `json:"calicoV6Interface,omitempty"`
}
```

### Default values

**AllowWorkloadsOnMaster**: false,
**CalicoV4Interface**:      "first-found",
**CalicoV6Interface**:      "first-found",
**ClusterName**:              "airctl-mgmt",
**ContainerRuntime**:       "containerd",
**SSHUser**:                "root",
**SSHPrivateKeyFile**:      "/root/.ssh/id_rsa",
**Pf9KubePkg**:             /opt/pf9/airctl/nodelet/nodelet.tar.gz,
**Privileged**:             "true",
**K8sApiPort**:             "443",
**MasterVipEnabled**:       false,
**MTU**:                    "1440",

Only the Calico CNI is supported. For more information on configuring the Calico options, please see: https://projectcalico.docs.tigera.io/networking/ip-autodetection

## How to use
It is best to show by example:

### Create a single-master cluster

First, create a sample cluster YAML file:
```
clusterName: airctl-mgmt
shUser: root
sshPrivateKeyFile: /root/.ssh/id_rsa
nodeletPkg: /opt/pf9/airctl/nodelet/nodelet.tar.gz
allowWorkloadsOnMaster: true
masterIp: 10.128.144.161
masterVipEnabled: true
masterVipInterface: eth0
masterVipVrouterId: 209
calicoV4Interface: "interface=eth0"
privileged: true
masterNodes:
  - nodeName: 10.128.144.151
workerNodes:
  - nodeName: 10.128.145.202
```
Some of the default values have been shown for completeness.

The masterVipVrouterId is optional. If unspecified, one will randomly be generated and can be found in the updated cluster spec saved in the cluster state directory. It is recommended to specify one if you will deploy multiple clusters in the same VLAN to avoid collision

Additionally, setting a masterVIPEnabled is optional if the cluster will always be single master. In this case, masterIp should match the single master node.

```
[root@arjunairdu ~]# ./nodeletctl create --config ~/cluster.yaml
Saved updated cluster spec to /etc/nodelet/airctl-mgmt/airctl-mgmt.yaml
```

### Scale up the cluster:

We will now scaleup to 5 workers and 2 masters. Our new cluster YAML may look like:
```
clusterName: airctl-mgmt
sshUser: root
sshPrivateKeyFile: /root/.ssh/id_rsa
kubeconfig: /etc/nodelet/airctl-mgmt/certs/admin.kubeconfig
nodeletPkg: /opt/pf9/airctl/nodelet/nodelet.tar.gz
allowWorkloadsOnMaster: true
masterIp: 10.128.144.161
masterVipEnabled: true
masterVipInterface: eth0
masterVipVrouterId: 209
calicoV4Interface: "interface=eth0"
privileged: true
masterNodes:
  - nodeName: 10.128.144.151
  - nodeName: 10.128.145.63
  - nodeName: 10.128.145.219
  - nodeName: 10.128.145.137
  - nodeName: 10.128.145.76
workerNodes:
  - nodeName: 10.128.145.202
  - nodeName: 10.128.145.197
```
Besides adding the new list of master and worker nodes, we also specified the generated admin kubeconfig file

``` ./nodeletctl scale --config ~/cluster.yaml```

Using the generated kubeconfig at ```/etc/nodelet/<CLUSTER_NAME>/certs/admin.kubeconfig```, we can see the cluster has been scaled up:
```
[root@arjunairdu certs]# kubectl get nodes -o wide
NAME             STATUS   ROLES    AGE     VERSION   INTERNAL-IP      EXTERNAL-IP   OS-IMAGE                KERNEL-VERSION                CONTAINER-RUNTIME
10.128.144.151   Ready    master   3m59s   v1.21.3   10.128.144.151   <none>        CentOS Linux 7 (Core)   3.10.0-1127.19.1.el7.x86_64   containerd://1.4.12
10.128.145.137   Ready    master   3m41s   v1.21.3   10.128.145.137   <none>        CentOS Linux 7 (Core)   3.10.0-1160.42.2.el7.x86_64   containerd://1.4.12
10.128.145.197   Ready    worker   2m56s   v1.21.3   10.128.145.197   <none>        CentOS Linux 7 (Core)   3.10.0-1160.42.2.el7.x86_64   containerd://1.4.12
10.128.145.202   Ready    worker   2m49s   v1.21.3   10.128.145.202   <none>        CentOS Linux 7 (Core)   3.10.0-1127.19.1.el7.x86_64   containerd://1.4.12
10.128.145.219   Ready    master   3m52s   v1.21.3   10.128.145.219   <none>        CentOS Linux 7 (Core)   3.10.0-1127.19.1.el7.x86_64   containerd://1.4.12
10.128.145.63    Ready    master   3m57s   v1.21.3   10.128.145.63    <none>        CentOS Linux 7 (Core)   3.10.0-1127.19.1.el7.x86_64   containerd://1.4.12
10.128.145.76    Ready    master   3m17s   v1.21.3   10.128.145.76    <none>        CentOS Linux 7 (Core)   3.10.0-1160.42.2.el7.x86_64   containerd://1.4.12
[root@arjunairdu certs]# kubectl get pods -A -o wide
NAMESPACE     NAME                                       READY   STATUS    RESTARTS   AGE     IP               NODE             NOMINATED NODE   READINESS GATES
kube-system   calico-kube-controllers-5fcd6c885b-n9k2l   1/1     Running   1          4m5s    10.20.3.65       10.128.145.137   <none>           <none>
kube-system   calico-node-68xwc                          1/1     Running   0          2m56s   10.128.145.202   10.128.145.202   <none>           <none>
kube-system   calico-node-6nsw2                          1/1     Running   0          4m5s    10.128.144.151   10.128.144.151   <none>           <none>
kube-system   calico-node-ldfhd                          1/1     Running   0          3m3s    10.128.145.197   10.128.145.197   <none>           <none>
kube-system   calico-node-nrwh7                          1/1     Running   0          4m4s    10.128.145.63    10.128.145.63    <none>           <none>
kube-system   calico-node-pzh69                          1/1     Running   0          3m59s   10.128.145.219   10.128.145.219   <none>           <none>
kube-system   calico-node-rpsxq                          1/1     Running   0          3m48s   10.128.145.137   10.128.145.137   <none>           <none>
kube-system   calico-node-xtssq                          1/1     Running   0          3m24s   10.128.145.76    10.128.145.76    <none>           <none>
kube-system   calico-typha-84d9f8c679-9xmhb              1/1     Running   0          4m5s    10.128.145.219   10.128.145.219   <none>           <none>
kube-system   calico-typha-84d9f8c679-rkq2f              1/1     Running   0          4m5s    10.128.145.63    10.128.145.63    <none>           <none>
kube-system   calico-typha-84d9f8c679-szvsq              1/1     Running   0          4m5s    10.128.145.137   10.128.145.137   <none>           <none>
kube-system   coredns-8597d6fb74-lsvr5                   1/1     Running   0          4m      10.20.3.195      10.128.145.219   <none>           <none>
kube-system   k8s-master-10.128.144.151                  3/3     Running   0          3m27s   10.128.144.151   10.128.144.151   <none>           <none>
kube-system   k8s-master-10.128.145.137                  3/3     Running   0          3m4s    10.128.145.137   10.128.145.137   <none>           <none>
kube-system   k8s-master-10.128.145.219                  3/3     Running   0          3m15s   10.128.145.219   10.128.145.219   <none>           <none>
kube-system   k8s-master-10.128.145.63                   3/3     Running   0          3m30s   10.128.145.63    10.128.145.63    <none>           <none>
kube-system   k8s-master-10.128.145.76                   3/3     Running   0          2m49s   10.128.145.76    10.128.145.76    <none>           <none>
kube-system   kube-dns-autoscaler-b6fd76964-xn8mj        1/1     Running   0          4m      10.20.3.194      10.128.145.219   <none>           <none>
```

### Scale down the cluster:

We will scale down to 3 master nodes. To do so, all we need to do is edit the node list like so:
```
masterNodes:
  - nodeName: 10.128.144.151
  - nodeName: 10.128.145.137
  - nodeName: 10.128.145.76
workerNodes:
  - nodeName: 10.128.145.202
  - nodeName: 10.128.145.197
```
We can then re-run the scale command:
```./nodeletctl scale --config ~/cluster.yaml```

### Delete the cluster:
Deleting the cluster will erase the nodelet package as well as clear out all nodelet related folders. This is intended to permanently delete the cluster and re-purpose the nodes.

```./nodeletctl delete --config ~/cluster.yaml```

The cluster.yaml should include the all the nodes in the cluster. So if we wanted to delete the above cluster, we would simply call delete with the last cluster.yaml we used to scale. 


