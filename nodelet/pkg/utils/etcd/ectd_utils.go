package etcd

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/config"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	cr "github.com/platform9/nodelet/nodelet/pkg/utils/container_runtime"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
	"go.uber.org/zap"
)

type EtcdUtils interface {
}

type EtcdImpl struct {
	file fileio.FileInterface
}

func New() EtcdUtils {
	return &EtcdImpl{
		file: fileio.New(),
	}
}
func (e *EtcdImpl) EnsureEtcdDataStoredOnHost() error {
	zap.S().Infof("Ensuring etcd data is stored on host")
	err := InspectEtcd()
	if err != nil {
		zap.S().Infof("Skipping; etcd container does not exist")
		return nil
	}
}
func (e *EtcdImpl) IsEligibleForEtcdBackup() (bool, error) {

	eligible := true
	if _, err := os.Stat(constants.EtcdVersionFile); err == nil {

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
	} else if errors.Is(err, os.ErrNotExist) {
		// writing etcd version to a file during start_master instead of stop_master,
		// During the upgrade new package sequence is : status --> stop --> start
		// Due to above, cannot rely on writing version during stop, as that will lead
		// to false assumption.
		// With this, backup and raft check shall happen once during both fresh install
		// and upgrade
		e.WriteEtcdVersionToFile()
	} else {
		// what to do here or remove this else block?
		// this block will come into the case when os.stat returns err which is not ErrNotExist
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
	err = cont.EnsureContainerDestroyed(ctx, "etcd", "10s")
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdImpl) EnsureEtcdClusterStatus() {

}
