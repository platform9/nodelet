package nodeletctl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

func InitEtcdClient(clusterCfg *BootstrapConfig, activeMasters *[]HostConfig) (*clientv3.Client, error) {
	etcdEndpoints := []string{}
	for _, host := range *activeMasters {
		var endpoint string
		if host.NodeIP != nil {
			endpoint = "https://" + *host.NodeIP + ":4001"
		} else {
			endpoint = "https://" + host.NodeName + ":4001"
		}
		etcdEndpoints = append(etcdEndpoints, endpoint)
	}

	etcdCert := filepath.Join("/etc/nodelet", clusterCfg.ClusterId, "certs/adminCert.pem")
	etcdCertKey := filepath.Join("/etc/nodelet", clusterCfg.ClusterId, "certs/adminKey.pem")
	etcdCa := filepath.Join("/etc/nodelet", clusterCfg.ClusterId, "certs/rootCA.crt")

	cert, err := tls.LoadX509KeyPair(etcdCert, etcdCertKey)
	if err != nil {
		return nil, fmt.Errorf("etcd client: failed to load X509 etcd certs: %s:", err)
	}

	caData, err := ioutil.ReadFile(etcdCa)
	if err != nil {
		return nil, fmt.Errorf("Failed to read CA file: %s", err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caData)

	_tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            pool,
		InsecureSkipVerify: true,
	}

	etcdConfig := clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 10 * time.Second,
		TLS:         _tlsConfig,
	}

	client, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to init etcd client: %s", err)
	}

	return client, nil
}

func AddNodeToEtcd(clusterCfg *BootstrapConfig, currMasters *[]HostConfig, hostIp string) error {
	etcdClient, err := InitEtcdClient(clusterCfg, currMasters)
	if err != nil {
		return fmt.Errorf("Failed to get etcd client: %s", err)
	}
	defer etcdClient.Close()

	endpoint := "https://" + hostIp + ":2380"
	zap.S().Infof("etcd MemberAdd: endpoint %s", endpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := etcdClient.MemberAdd(ctx, []string{endpoint})
	cancel()
	if resp != nil && resp.Members != nil {
		zap.S().Infof("New resp after MemberAdd: %d members: %+v\n", len(resp.Members), resp.Members)
	}
	if err != nil {
		zap.S().Errorf("Failed to add member! %s", err)
		return fmt.Errorf("Failed to add etcd member: %s", err)
	}
	return nil
}

func RemoveNodeFromEtcd(clusterCfg *BootstrapConfig, currMasters *[]HostConfig, hostIp string) error {
	etcdClient, err := InitEtcdClient(clusterCfg, currMasters)
	if err != nil {
		return fmt.Errorf("Failed to get etcd client: %s", err)
	}
	defer etcdClient.Close()

	memberIdToRemove := findEtcdMemberByIp(etcdClient, hostIp)
	if memberIdToRemove == 0 {
		zap.S().Errorf("Could not find master in etcd cluster")
		return fmt.Errorf("Could not find master in etcd cluster")
	}

	endpoint := "https://" + hostIp + ":2380"
	zap.S().Infof("etcd MemberRemove: endpoint %s", endpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := etcdClient.MemberRemove(ctx, memberIdToRemove)
	cancel()
	if resp != nil && resp.Members != nil {
		zap.S().Infof("New resp after MemberAdd: %d members: %+v\n", len(resp.Members), resp.Members)
	}
	if err != nil {
		zap.S().Errorf("Failed to add member! %s", err)
		return fmt.Errorf("Failed to add etcd member: %s", err)
	}
	return nil
}

func findEtcdMemberByIp(etcdClient *clientv3.Client, hostIp string) uint64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := etcdClient.MemberList(ctx)
	cancel()
	if err != nil {
		zap.S().Errorf("Failed to get MemberList! %s\n", err)
		return 0
	}
	zap.S().Infof("etcd MemberList: %d members: %+v\n", len(resp.Members), resp.Members)
	var memberId uint64 = 0
	for _, member := range resp.Members {
		if hostIp == member.GetName() {
			memberId = member.GetID()
			break
		}
	}

	return memberId
}
