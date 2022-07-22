package embedutil

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path"

	"go.uber.org/zap"
)

type EmbedFS struct {
	Fs   embed.FS
	Root string
}

// Extract the embedded content and the root to the dest
func (efs *EmbedFS) Extract(dest string) error {
	return efs.extract(efs.Root, dest)
}

func (efs *EmbedFS) Copy(filepath string, destpath string) error {
	//zap.S().Infof("Copying '%s' to '%s'", filepath, destpath)
	zap.S().Infof("In Copying '%s' to '%s'", "filepath", "destpath")
	err := os.MkdirAll(path.Dir(destpath), 0755)
	if err != nil && !os.IsExist(err) {
		zap.S().Infof("failed to create directory '%s': %s", path.Dir(destpath), err)
		return fmt.Errorf("failed to create directory '%s': %s", path.Dir(destpath), err)
	}
	zap.S().Infof("Created/exist dir:'%s'", path.Dir(destpath))
	srcFile, err := efs.Fs.Open(filepath)
	if err != nil {
		zap.S().Infof("failed to open file in : %s :%s", path.Dir(filepath), err)
		return fmt.Errorf("failed to open '%s': %s", filepath, err)
	}
	zap.S().Infof("file in '%s' opened", path.Dir(filepath))
	defer srcFile.Close()
	destFile, err := os.Create(destpath)
	if err != nil {
		zap.S().Infof("failed to create '%s': %s", destpath, err)
		return fmt.Errorf("failed to create '%s': %s", destpath, err)
	}
	zap.S().Infof("Created dest file:'%s'", destpath)
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		zap.S().Infof("Copying data from '%s' to '%s'", "srcFile", "destFile")
		return fmt.Errorf("failed to Copy data from '%s' to '%s'", "srcFile", "destFile")
	}
	zap.S().Infof("Copied src file to destfile ", srcFile, destFile)
	return nil
}

func (efs *EmbedFS) extract(root string, dest string) error {
	zap.S().Infof("Extracting %s to '%s'", root, dest)
	items, err := efs.Fs.ReadDir(root)
	if err != nil {
		zap.S().Infof("failed to read root directory: %s", err)
		return fmt.Errorf("failed to read pf9-kube directory: %s", err)
	}
	for _, item := range items {
		filepath := path.Join(root, item.Name())
		destpath := path.Join(dest, item.Name())
		if item.IsDir() {
			if err := efs.extract(filepath, destpath); err != nil {
				zap.S().Infof("failed to extract: %s", err)
				return err
			}
			continue
		}
		zap.S().Infof("Copying '%s' to '%s'", item.Name(), dest)
		err := efs.Copy(filepath, destpath)
		if err != nil {
			zap.S().Infof("failed to copy '%s': %s", item.Name(), err)
			return fmt.Errorf("failed to copy '%s': %s", item.Name(), err)
		}
	}
	return nil
}
