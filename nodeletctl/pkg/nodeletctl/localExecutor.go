package nodeletctl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/platform9/pf9ctl/pkg/ssh"
	"go.uber.org/zap"

	"github.com/kballard/go-shellquote"
)

// Client interface provides ways to run command and upload files to remote hosts
type LocalClient struct {
}

func GetLocalClient() ssh.Client {
	return &LocalClient{}
}

// RunCommand executes the remote command returning the stdout, stderr and any error associated with it
func (client *LocalClient) RunCommand(command string) ([]byte, []byte, error) {
	zap.S().Debugf("Running command: %s", command)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	words, err := shellquote.Split(command)
	if err != nil || len(words) < 1 {
		return nil, nil, fmt.Errorf("error parsing the command line %s  %v", command, err)
	}
	cmd := exec.Command(words[0], words[1:]...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

// Uploadfile uploads the srcFile to remoteDestFilePath and changes the mode to the filemode
func (client *LocalClient) UploadFile(srcFilePath, remoteDstFilePath string, mode os.FileMode, cb func(read int64, total int64)) error {
	zap.S().Debugf("Uploading file %s to %s", srcFilePath, remoteDstFilePath)
	return copy(srcFilePath, remoteDstFilePath, mode)
}

// Downloadfile downloads the remoteFile to localFile and changes the mode to the filemode
func (client *LocalClient) DownloadFile(remoteFile, localPath string, mode os.FileMode, cb func(read int64, total int64)) error {
	zap.S().Debugf("Downloading file %s to %s", remoteFile, localPath)
	return copy(remoteFile, localPath, mode)
}

func copy(srcFile, dstFile string, mode os.FileMode) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("error opening file %s, %v", srcFile, err)
	}
	defer src.Close()
	// create new file
	dest, err := os.Create(dstFile)
	if err != nil {
		return fmt.Errorf("error creating remote file %s %v", dstFile, err)
	}
	err = dest.Chmod(mode)
	if err != nil {
		return fmt.Errorf("error changing the file mode %s, %v", dstFile, err)
	}
	defer dest.Close()
	_, err = io.Copy(dest, src)
	if err != nil {
		return fmt.Errorf("error copying %s to %s, %v", srcFile, dstFile, err)
	}
	return nil
}
