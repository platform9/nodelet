package untar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// extract the targz file to the destination directory
func Extract(tgzFile string, destDir string) error {

	f, err := os.Open(tgzFile)
	if err != nil {
		zap.S().Infof("error opening tgz file: %v", err)
		return fmt.Errorf("error opening tgz file: %v", err)
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		zap.S().Infof("error creating gzip reader: %v", err)
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	zap.S().Infof("gzip reader created")
	defer gzf.Close()
	tarReader := tar.NewReader(gzf)
	zap.S().Infof("tar reader created")
	for {
		f, err := tarReader.Next()
		if err == io.EOF {
			zap.S().Infof("for breaked : %s: %v", "err == io.EOF", err)
			break
		}

		if err != nil {
			zap.S().Infof("error reading tar file: %v", err)
			return fmt.Errorf("error reading tar file: %v", err)
		}
		zap.S().Infof("tar file read ")
		fpath := filepath.Join(destDir, f.Name)
		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsDir():
			err = os.MkdirAll(fpath, 0755)
			if err != nil {
				zap.S().Infof("error creating directory: %s %v", fpath, err)
				return fmt.Errorf("error creating directory: %s %v", fpath, err)
			}
			zap.S().Infof("in is dir mode and created dir: %v", fpath)
		case mode.IsRegular():
			// this is redundant
			err = os.MkdirAll(filepath.Dir(fpath), 0755)
			if err != nil {
				zap.S().Infof("error creating directory: %s %v", filepath.Dir(fpath), err)
				return fmt.Errorf("error creating directory: %s %v", filepath.Dir(fpath), err)
			}
			zap.S().Infof("in is regular mode and created dir: %v", filepath.Dir(fpath))
			destFile, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0755)
			defer destFile.Close()
			if err != nil {
				zap.S().Infof("error creating file: %s %v", fpath, err)
				return fmt.Errorf("error creating file: %s %v", fpath, err)
			}
			zap.S().Infof("created file: %v", fpath)
			n, err := io.Copy(destFile, tarReader)

			if err != nil {
				zap.S().Infof("error copying file: %s %v", fpath, err)
				return fmt.Errorf("error copying file: %s %v", fpath, err)
			}
			zap.S().Infof("copyied file tarreader to destfile")
			if n != f.Size {
				zap.S().Infof("error copying file size written not equal: expected %d, got %d, %s %v", f.Size, n, fpath, err)
				return fmt.Errorf("error copying file size written not equal: expected %d, got %d, %s %v", f.Size, n, fpath, err)
			}

		}
	}
	zap.S().Infof("exiting untar ")
	return nil
}
