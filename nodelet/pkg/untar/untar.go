package untar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// extract the targz file to the destination directory
func Extract(tgzFile string, destDir string) error {

	f, err := os.Open(tgzFile)
	if err != nil {
		return fmt.Errorf("error opening tgz file: %s %v", tgzFile, err)
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzf.Close()
	tarReader := tar.NewReader(gzf)

	for {
		f, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading tar file: %v", err)
		}
		fpath := filepath.Join(destDir, f.Name)
		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsDir():
			err = os.MkdirAll(fpath, 0755)
			if err != nil {
				return fmt.Errorf("error creating directory: %s %v", fpath, err)
			}
		case mode.IsRegular():
			// this is redundant
			err = os.MkdirAll(filepath.Dir(fpath), 0755)
			if err != nil {
				return fmt.Errorf("error creating directory: %s %v", filepath.Dir(fpath), err)
			}
			destFile, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, 0755)
			defer destFile.Close()
			if err != nil {
				return fmt.Errorf("error creating file: %s %v", fpath, err)
			}
			n, err := io.Copy(destFile, tarReader)

			if err != nil {
				return fmt.Errorf("error copying file: %s %v", fpath, err)
			}
			if n != f.Size {
				return fmt.Errorf("error copying file size written not equal: expected %d, got %d, %s %v", f.Size, n, fpath, err)
			}

		}
	}
	return nil
}
