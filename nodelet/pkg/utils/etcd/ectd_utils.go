package etcd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"go.uber.org/zap"
)

type EtcdUtils interface {
	EnsureEtcdDataStoredOnHost() (bool, error)
	IsEligibleForEtcdBackup() (bool, error)
	EnsureEtcdDataBackup(cfg config.Config) error
	WriteEtcdVersionToFile() error
	EnsureEtcdDestroyed(ctx context.Context) error
	CheckEtcdRaftIndex() error
	EnsureEtcdClusterStatus() error
	InspectEtcd() (bool, error)
	EnsureEtcdRunning(ctx context.Context, cfg config.Config) error
}

type EtcdImpl struct {
	file fileio.FileInterface
}

func New() EtcdUtils {
	return &EtcdImpl{
		file: fileio.New(),
	}
}

// Todo :
//IsContainerExist
//isContainerRunning
//isEtcdHealthy
//get_container_logs() by logpath
//getetcdlogs by pf9ctr_run logs etcd > "${logfile}"

func (e *EtcdImpl) EnsureEtcdDataStoredOnHost() (bool, error) {
	zap.S().Infof("Ensuring etcd data is stored on host")
	exist, err := e.InspectEtcd()
	if err != nil {

		zap.S().Errorf("error when checking etcd container exist")
		return false, err
	}
	return exist, nil
}

func (e *EtcdImpl) InspectEtcd() (bool, error) {
	// cont, err := cr.NewContainerUtil()
	// if err != nil {
	// 	return false, err
	// }
	//exist, err := cont.IsContainerExist()
	// if err != nil {
	// 	return false, err
	// }
	return false, nil
}

func (e *EtcdImpl) IsEligibleForEtcdBackup() (bool, error) {

	eligible := true

	if _, err := os.Stat(constants.EtcdVersionFile); os.IsNotExist(err) {
		// writing etcd version to a file during start_master instead of stop_master,
		// During the upgrade new package sequence is : status --> stop --> start
		// Due to above, cannot rely on writing version during stop, as that will lead
		// to false assumption.
		// With this, backup and raft check shall happen once during both fresh install
		// and upgrade
		e.WriteEtcdVersionToFile()
	} else {
		oldVersion, err := e.file.ReadFile(constants.EtcdVersionFile)
		if err != nil {
			return false, err
		}
		// no backup done if etcd version are the same
		if string(oldVersion) == constants.EtcdVersion {
			//return false, nil
			eligible = false
		} else {
			// when etcd version is a mismatch, that indicates upgrade
			// perform backup and raft check and update the etcd version to most recent
			e.WriteEtcdVersionToFile()
		}
	}

	return eligible, nil
}

func (e *EtcdImpl) EnsureEtcdDataBackup(cfg config.Config) error {

	EtcdDataMemberDir := fmt.Sprintf("%s/member", cfg.EtcdDataDir)

	if _, err := os.Stat(EtcdDataMemberDir); err == nil {

		if _, err := os.Stat(constants.EtcdBackupDir); err == nil {
			zap.S().Infof("%s dir already present", constants.EtcdBackupDir)
		} else if errors.Is(err, os.ErrNotExist) {
			zap.S().Infof("creating %s", constants.EtcdBackupDir)
			err = os.MkdirAll(constants.EtcdBackupDir, 0660)
			if err != nil {
				return err
			}
		} else {
			return err
		}

		if _, err := os.Stat(constants.EtcdBackUpLoc); err == nil {
			zap.S().Infof("cleaning existing etcdv3 backup and taking a new backup")
			err = os.Remove(constants.EtcdBackUpLoc)
			if err != nil {
				return err
			}
		}
		dbfile := fmt.Sprintf("%s/member/snap/db", cfg.EtcdDataDir)
		err = e.file.CopyFile(dbfile, constants.EtcdBackUpLoc)
		if err != nil {
			zap.S().Errorf("etcdv3 backup failed:%v", err)
			return errors.Wrapf(err, "etcdv3 backup failed")
		}
		zap.S().Infof("etcdv3 backup success")

	} else if errors.Is(err, os.ErrNotExist) {
		zap.S().Infof("etcd %s directory not found. skipping etcd data backup", cfg.EtcdDataDir)
	}
	return nil
}

func (e *EtcdImpl) WriteEtcdVersionToFile() error {
	err := e.file.WriteToFile(constants.EtcdVersionFile, constants.EtcdVersion, false)
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdImpl) EnsureEtcdDestroyed(ctx context.Context) error {
	cont, err := cr.NewContainerUtil()
	if err != nil {
		return err
	}
	err = cont.EnsureContainerDestroyed(ctx, "etcd", "10s")
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdImpl) EnsureEtcdClusterStatus() error {
	err := e.CheckEtcdRaftIndex()
	if err != nil {
		zap.S().Infof("etcd cluster status not ok")
		return err
	}
	zap.S().Infof("etcd cluster status ok")
	return nil
}

func (e *EtcdImpl) WriteEtcdEnv(cfg config.Config) error {
	zap.S().Info("Deriving local etcd environment")
	if _, err := os.Stat(constants.EtcdEnvFile); os.IsNotExist(err) {
		err := e.file.TouchFile(constants.EtcdEnvFile)
		if err != nil {
			return err
		}
	}
	err := e.file.WriteToFile(constants.EtcdEnvFile, cfg.EtcdEnv, false)
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdImpl) EnsureEtcdRunning(ctx context.Context, cfg config.Config) error {

	err := createEtcdDirsIfnotPresent(ctx, cfg)
	if err != nil {
		return err
	}
	err = e.WriteEtcdEnv(cfg)
	if err != nil {
		return errors.Wrapf(err, "could not write etcd env")
	}

	volumes := getEtcdVolume(cfg)
	etcdEnv := getEtcdEnv(cfg)
	etcdEnvFiles := []string{constants.EtcdEnvFile}
	etcdContainerNetwork := cr.Host

	etcdRunOpts := cr.RunOpts{}

	etcdRunOpts.Volumes = volumes
	etcdRunOpts.Env = etcdEnv
	etcdRunOpts.EnvFiles = etcdEnvFiles
	etcdRunOpts.Network = etcdContainerNetwork
	etcdRunOpts.Privileged = false

	etcdContainerName := "etcd"
	etcdContainerImage := getEtcdContainerImage(cfg)
	etcdCmdArgs := getEtcdCmdArgs()

	cont, err := cr.NewContainerUtil()
	if err != nil {
		return err
	}
	err = cont.EnsureFreshContainerRunning(ctx, constants.K8sNamespace, etcdContainerName, etcdContainerImage, etcdRunOpts, etcdCmdArgs)
	if err != nil {
		zap.S().Errorf("running etcd container failed: %v", err)
		return err
	}

	// healthy := false
	// retries := 90
	// for i := 0; i < retries; i++ {
	// 	running = isContainerRunning("etcd")
	// 	if !running {
	// 		//local logfile="/var/log/pf9/kube/etcd-${timestamp}.log"
	// 		//pf9ctr_run logs etcd > "${logfile}" 2>&1
	// 		zap.S().Info("Restarting failed etcd")
	// 		err = cont.EnsureFreshContainerRunning(ctx, etcdContainerName, etcdContainerImage, etcdRunOpts, etcdCmdArgs)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		continue
	// 	}
	// 	healthy = isEtcdHealthy()
	// 	if healthy {
	// 		break
	// 	}
	// 	zap.S().Infof("Waiting for healthy etcd cluster")
	// }
	// if !healthy {
	// 	zap.S().Infof("timed out waiting for etcd initialization")
	// 	return err
	// }
	return nil
}

func getEtcdVolume(cfg config.Config) []string {
	var volumes []string
	volumes = append(volumes,
		"/etc/ssl:/etc/ssl",
		"/etc/pki:/etc/pki",
		"/etc/pf9/kube.d/certs/etcd:/certs/etcd",
		"/etc/pf9/kube.d/certs/apiserver:/certs/apiserver",
		fmt.Sprintf("%s:/var/etcd/data", cfg.EtcdDataDir),
	)
	return volumes
}

func getEtcdEnv(cfg config.Config) []string {

	// ETCD_LOG_LEVEL: --debug flag and ETCD_DEBUG to be deprecated in v3.5
	// ETCD_LOGGER: default logger capnslog to be deprecated in v3.5, using zap
	// ETCD_ENABLE_V2: Need this for flannel's compatibility with etcd v3.4.14

	var etcdEnv []string
	etcdLogLevel := "info"
	if cfg.Debug == "true" {
		etcdLogLevel = "debug"
	}

	etcdEnv = append(etcdEnv,
		fmt.Sprintf("ETCD_LOG_LEVEL=%s", etcdLogLevel),
		"ETCD_LOGGER=zap",
		"ETCD_ENABLE_V2=true",
		"ETCD_PEER_CLIENT_CERT_AUTH=true",
	)

	if cfg.EtcdHeartBeatInterval != "" {
		etcdEnv = append(etcdEnv, cfg.EtcdHeartBeatInterval)
	}
	if cfg.EtcdElectionTimeOut != "" {
		etcdEnv = append(etcdEnv, cfg.EtcdElectionTimeOut)
	}

	// TODO
	// PMK-3665: Customise ETCD in platform9 managed kubernetes cluster

	// The flexibility of customizing ETCD with the help of environment variables
	// needs support from DU side as well if we want it to be truly customizable
	// at the time of cluster creation or at the time of cluster update.
	// For now, we are only checking for the following two environment variables
	// that can be provided via override enironment file at /etc/pf9/kube_override.env
	// on all master nodes. This will make such customizations persistent across node
	// reboots and upgrades.

	// Default snapshot count is set to 100000 from ETCD v3.2 onwards as compared
	// 10000 in earlier versions.

	// If ETCD is getting OOM Killed, this could be one of the possible
	// reasons. ETCD retains all the snapshots in memory so that new nodes joining
	// the ETCD cluster or slow nodes can catch up.

	// Provide an override environment variable in /etc/pf9/kube_override.env
	// set to a lower value exported under name ETCD_SNAPSHOT_COUNT
	// This also results in lower WAL files or write action log files which may
	// consume huge disk space if the snapshot count is high.

	etcdSnapshotCount := os.Getenv("ETCD_SNAPSHOT_COUNT")
	if etcdSnapshotCount != "" {
		etcdEnv = append(etcdEnv, etcdSnapshotCount)
	}

	// Default max DB size for ETCD is set to 2.1 GB
	// Incase the max DB size is reached, ETCD stops responding to any get/put/watch
	// calls resulting into k8s cluster control plane going down.

	// One of the reasons why this can happen is due to huge amount of older revisions
	// of key values in ETCD database. Although auto compation happens in ETCD every 5 mins,
	// if during these 5 minutes, there are frequent writes happening on the cluster,
	// the revisions pile up during those 5 minutes and even though compaction happens every
	// 5 mins, the space claimed by DB is not released back to the system. In order to release
	// the space, one needs to defrag ETCD manually.

	// If you are expecting intensive writes over a period of 5 mins, it is best to increase
	// the default quota bytes for DB and set it to a higher value, max can be 8GB

	// Provide an override environment variable in /etc/pf9/kube_override.env
	// set to value in bytes, exported under name ETCD_QUOTA_BACKEND_BYTES

	etcdQuotaBaclendBytes := os.Getenv("ETCD_QUOTA_BACKEND_BYTES")
	if etcdQuotaBaclendBytes != "" {
		etcdEnv = append(etcdEnv, etcdQuotaBaclendBytes)
	}

	// One can control the frequency and extent of compaction using following two environment
	// variables:
	// a) ETCD_AUTO_COMPACTION_MODE
	// b) ETCD_AUTO_COMPACTION_RETENTION

	// ETCD_AUTO_COMPACTION_MODE: which can be set to 'periodic' or 'revision'
	//                            default value is periodic.

	//              periodic can be used if you want to retain key value revisions from the
	//              last time window specified in ETCD_AUTO_COMPACTION_RETENTION env variable.
	//              e.g. 1h or 30m

	//              revision can be used if you want to retains last n revisions of key values.
	//              You can specify the value in in ETCD_AUTO_COMPACTION_RETENTION env variable.

	etcdAutoCompactionMode := os.Getenv("ETCD_AUTO_COMPACTION_MODE")
	if etcdAutoCompactionMode != "" {
		etcdEnv = append(etcdEnv, etcdAutoCompactionMode)
	}

	etcdAutoComactionRetention := os.Getenv("ETCD_AUTO_COMPACTION_RETENTION")
	if etcdAutoComactionRetention != "" {
		etcdEnv = append(etcdEnv, etcdAutoComactionRetention)
	}
	return etcdEnv
}

func getEtcdContainerImage(cfg config.Config) string {
	gcrRegistry := constants.GcrRegistry
	if cfg.GcrPrivateRegistry != "" {
		gcrRegistry = cfg.GcrPrivateRegistry
	}
	etcdContainerImage := strings.ReplaceAll(constants.EtcdContainerImg, constants.GcrRegistry, gcrRegistry)
	return etcdContainerImage
}

func getEtcdCmdArgs() []string {
	var etcdCmdArgs []string
	containerCmd := "/usr/local/bin/etcd"
	extraOptEtcdFlags := os.Getenv("EXTRA_OPT_ETCD_FLAGS")
	etcdCmdArgs = append(etcdCmdArgs, containerCmd)
	if extraOptEtcdFlags != "" {
		etcdCmdArgs = append(etcdCmdArgs, extraOptEtcdFlags)
	}
	return etcdCmdArgs
}

func createEtcdDirsIfnotPresent(ctx context.Context, cfg config.Config) error {
	cmd := command.New()

	err := os.MkdirAll(cfg.EtcdDataDir, 0700)
	if err != nil {
		return errors.Wrapf(err, "could not create etcd data dir:%v", cfg.EtcdDataDir)
	}
	if _, err := os.Stat(constants.EtcdConfDir); errors.Is(err, os.ErrNotExist) {
		zap.S().Infof("creating dir: %s", constants.EtcdConfDir)
		// exitCode, stdErr, err := cmd.RunCommandWithStdErr(ctx, nil, 0, "", "sudo", "mkdir", constants.EtcdConfDir)
		// if err != nil || exitCode != 0 {
		// 	return fmt.Errorf("could not create dir %s: %v, STDERR:%v", constants.EtcdConfDir, err, stdErr)
		// }
		err = os.Mkdir(constants.EtcdConfDir, 0700)
		if err != nil {
			return errors.Wrapf(err, "could not create etcd conf dir:%v", constants.EtcdConfDir)
		}
	}
	if _, err := os.Stat("/etc/pki"); errors.Is(err, os.ErrNotExist) {
		zap.S().Info("creating dir /etc/pki")
		exitCode, stdErr, err := cmd.RunCommandWithStdErr(ctx, nil, 0, "", "sudo", "mkdir", "/etc/pki")
		if err != nil || exitCode != 0 {
			return fmt.Errorf("could not create dir /etc/pki: %v, STDERR:%v", err, stdErr)
		}
	}
	return nil
}
