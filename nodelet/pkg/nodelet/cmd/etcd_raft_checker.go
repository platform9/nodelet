package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	etcdctlBin = "/opt/pf9/pf9-kube/bin/etcdctl"

	etcdctlCertFlag = "--cert"
	etcdctlKeyFlag  = "--key"
	etcdctlCaFlag   = "--cacert"

	etcdctlCert = "/etc/pf9/kube.d/certs/etcdctl/etcd/request.crt"
	etcdctlKey  = "/etc/pf9/kube.d/certs/etcdctl/etcd/request.key"
	etcdctlCa   = "/etc/pf9/kube.d/certs/etcdctl/etcd/ca.crt"
)

// Commander interface encapsulates running of a function.
type Commander interface {
	Run(string, ...string) ([]byte, error)
}

// ExecCommander implements the Commander interface.
type ExecCommander struct{}

// Run command defines execution of a function along with specified
// arguments.
func (c ExecCommander) Run(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).CombinedOutput()
}

// EndPointStatus defines structure of response from etcd regarding endpoints
// of cluster.
type EndPointStatus struct {
	Endpoint string `json:"Endpoint"`
	Status   struct {
		Header struct {
			ClusterID uint64 `json:"cluster_id"`
			MemberID  uint64 `json:"member_id"`
			Revision  int64  `json:"revision"`
			RaftTerm  uint64 `json:"raft_term"`
		}
		Version   string `json:"version"`
		DbSize    int64  `json:"dbSize"`
		Leader    uint64 `json:"leader"`
		RaftIndex uint64 `json:"raftIndex"`
		RaftTerm  uint64 `json:"raftTerm"`
	}
}

// checkEndpointStatus checks the health of etcd cluster using raft index based check.
//
// example:
// /opt/pf9/pf9-kube/bin/etcdctl --cert /etc/pf9/kube.d/certs/etcdctl/etcd/request.crt --key /etc/pf9/kube.d/certs/etcdctl/etcd/request.key \
// --cacert /etc/pf9/kube.d/certs/etcdctl/etcd/ca.crt endpoint status --endpoints "https://10.0.3.117:4001,https://10.0.1.205:4001,https://10.0.2.139:4001" --write-out json
//
// output:
// {
//	"Endpoint": "https://10.0.3.39:4001",
//	"Status": {
//		"header": {
//			"cluster_id": 18436746753448134444,
//			"member_id": 7136212046390981183,
//			"revision": 30800,
//			"raft_term": 67
//		},
//		"version": "3.1.20",
//		"dbSize": 4055040,
//		"leader": 15742163283330712808,
//		"raftIndex": 75154,
// 		"raftTerm": 67
// 	}
// }
//
func checkEndpointStatus(commander Commander) error {

	// use v3 API to get cluster status
	os.Setenv("ETCDCTL_API", "3")
	flags := []string{
		etcdctlCertFlag,
		etcdctlCert,
		etcdctlKeyFlag,
		etcdctlKey,
		etcdctlCaFlag,
		etcdctlCa,
	}

	args := []string{
		"endpoint",
		"status",
		"--cluster",
		"--write-out",
		"json",
	}
	out, err := commander.Run(
		etcdctlBin,
		append(flags, args...)...)

	if err != nil {
		return fmt.Errorf("etcd endpoint status command execution failed")
	}
	outLines := strings.Split(string(out), "\n")

	var statusOutput string
	for _, line := range outLines {
		if strings.Contains(line, "Endpoint") {
			statusOutput = line
			break
		}
	}

	var EndPointStatusList []EndPointStatus

	data := []byte(statusOutput)
	error := json.Unmarshal(data, &EndPointStatusList)
	if error != nil {
		return fmt.Errorf("failed to unmarshal JSON: %s, ERROR: %s", data, error)
	}

	if len(EndPointStatusList) == 0 {
		return fmt.Errorf("number of etcd endpoints are Zero (0)")
	}

	// raft index difference should not be more than 1 among any two members
	maxRaftIndexDiff := maxDiffInRaftIndices(EndPointStatusList)
	if maxRaftIndexDiff > 1 {
		return fmt.Errorf("etcd raft index difference is > 1")
	}

	fmt.Printf("etcd raft index check passed. max raft index diff: %d \n", maxRaftIndexDiff)
	return nil
}

func maxDiffInRaftIndices(endPointList []EndPointStatus) uint64 {
	var min uint64 = endPointList[0].Status.RaftIndex
	var max uint64 = endPointList[0].Status.RaftIndex

	for _, endpoint := range endPointList {
		if max < endpoint.Status.RaftIndex {
			max = endpoint.Status.RaftIndex
		}
		if min > endpoint.Status.RaftIndex {
			min = endpoint.Status.RaftIndex
		}
	}
	return max - min
}
