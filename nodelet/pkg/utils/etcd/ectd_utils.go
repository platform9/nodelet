package etcd

import (
	"errors"
	"fmt"
	"os"

	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	"go.uber.org/zap"
)

func EnsureEtcdDataStoredOnHost() error {
	zap.S().Infof("Ensuring etcd data is stored on host")
	err := InspectEtcd()
	if err != nil {
		zap.S().Infof("Skipping; etcd container does not exist")
		return nil
	}
}
func IsEligibleForEtcdBackup() (bool, error) {

	etcdVersionFile := "/var/opt/pf9/etcd_version"
	eligible := true
	if _, err := os.Stat(etcdVersionFile); err == nil {

		file := fileio.New()
		oldVersion, err := file.ReadFile(etcdVersionFile)
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
			WriteEtcdVersionToFile()
		}
	} else if errors.Is(err, os.ErrNotExist) {
		// writing etcd version to a file during start_master instead of stop_master,
		// During the upgrade new package sequence is : status --> stop --> start
		// Due to above, cannot rely on writing version during stop, as that will lead
		// to false assumption.
		// With this, backup and raft check shall happen once during both fresh install
		// and upgrade
		WriteEtcdVersionToFile()
	} else {
		// what to do here or remove this else block?
		// this block will come into the case when os.stat returns err which is not ErrNotExist
	}
	return eligible, nil
}

func EnsureEtcdDataBackup() error {

	EtcdCtlBin := "/opt/pf9/pf9-kube/bin/etcdctl"
	EtcdBackupDir := "/var/opt/pf9/kube/etcd/etcd-backup"
	EtcdBackUpLoc := fmt.Spritf("%s/etcdv3_backup.db", EtcdBackupDir)
	EtcdDataMemberDir = fmt.Sprintf("%s/member", constants.EtcdDataDir)

	if _, err := os.Stat(EtcdDataMemberDir); err == nil {

		if _, err := os.Stat(EtcdBackupDir); err == nil {
			zap.S().Infof("%s dir already present", EtcdBackupDir)
		} else if errors.Is(err, os.ErrNotExist) {
			zap.S().Infof("creating %s", EtcdBackupDir)
			err = os.MkdirAll(EtcdBackupDir, 0660)
			if err != nil {
				return err
			}
		} else {
			return err
		}

		if _, err := os.Stat(EtcdBackUpLoc); err == nil {
			zap.S().Infof("cleaning existing etcdv3 backup and taking a new backup")
			err = os.Remove(EtcdBackUpLoc)
			if err != nil {
				return err
			}
		}

		cp -aR ${ETCD_DATA_DIR}/member/snap/db ${ETCD_BACKUP_LOC} && rc=$? || rc=$?
        if [ $rc -ne 0 ]; then
            echo "etcdv3 backup failed"
            exitCode=1
        else
            echo "etcdv3 backup success"
        fi

	} else if errors.Is(err, os.ErrNotExist) {
		zap.S().Infof("etcd %s directory not found. skipping etcd data backup",constants.EtcdDataDir)
	}
	return nil
}

func WriteEtcdVersionToFile() error {
	etcdVersionFile := "/var/opt/pf9/etcd_version"
	file := fileio.New()
	err := file.WriteToFile(etcdVersionFile, constants.EtcdVersion, false)
	if err != nil {
		return err
	}
	return nil
}

func EnsureEtcdDestroyed() error {
	cont := cr.NewContainerd()
	err := cont.EnsureContainerDestroyed(ctx,"etcd","10s")
	if err!=nil{
		return err
	}
}

func EnsureEtcdClusterStatus() {

}