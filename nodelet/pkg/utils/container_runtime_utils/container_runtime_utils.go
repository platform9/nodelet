package containerruntimeutils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"github.com/platform9/nodelet/nodelet/pkg/utils/fileio"
)

type Runtime interface {
	LoadImagesFromDir(context.Context, string, string) error
	LoadImagesFromFile(context.Context, string) error
	GenerateChecksum(string) error
	VerifyChecksum(string) (bool, error)
	GenerateHashForDir(string) ([]byte, error)
}

type RuntimeUtil struct{}

func New() Runtime {
	return &RuntimeUtil{}
}

//loads images from all tar files in the given directory to container runtime with given namespace
func (r *RuntimeUtil) LoadImagesFromDir(ctx context.Context, imageDir string, namespace string) error {
	items, _ := ioutil.ReadDir(imageDir)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else {
			imageFile := fmt.Sprintf("%s/%s", imageDir, item.Name())
			ctx = namespaces.WithNamespace(ctx, namespace)
			err := r.LoadImagesFromFile(ctx, imageFile)
			if err != nil {
				return errors.Wrapf(err, "could not load images from : %s", imageFile)
			}
		}
	}
	return nil
}

//loads images from given tar file to container runtime
func (r *RuntimeUtil) LoadImagesFromFile(ctx context.Context, fileName string) error {

	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	decompressor, err := compression.DecompressStream(f)
	if err != nil {
		return err
	}

	platform := platforms.DefaultStrict()

	client, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultPlatform(platform))
	if err != nil {
		return errors.Wrap(err, "failed to create client")
	}

	imgs, err := client.Import(ctx, decompressor, containerd.WithDigestRef(archive.DigestTranslator(constants.DefaultSnapShotter)), containerd.WithSkipDigestRef(func(name string) bool { return name != "" }), containerd.WithImportPlatform(platform))
	if err != nil {
		return errors.Wrap(err, "failed to import images")
	}
	for _, img := range imgs {
		image := containerd.NewImageWithPlatform(client, img, platform)
		err = image.Unpack(ctx, constants.DefaultSnapShotter)
		if err != nil {
			return errors.Wrapf(err, "failed to unpack image: %s", image.Name())
		}
	}
	return nil
}

//generates sha256 checksum for files and writes the checksum file in checksum sub-dir in given directory
func (r *RuntimeUtil) GenerateChecksum(imageDir string) error {

	hash, err := r.GenerateHashForDir(imageDir)
	if err != nil {
		return errors.Wrap(err, "could not generate hash")
	}

	if _, err := os.Stat(constants.ChecksumDir); os.IsNotExist(err) {
		if err := os.Mkdir(constants.ChecksumDir, os.ModePerm); err != nil {
			return errors.Wrapf(err, "failed to create directory: %s", constants.ChecksumDir)
		}
	}

	f := fileio.New()
	err = f.WriteToFile(constants.ChecksumFile, hash, false)
	if err != nil {
		return err
	}
	return nil
}

//verifies the current sha256 checksum of all files with checksum file
func (r *RuntimeUtil) VerifyChecksum(imageDir string) (bool, error) {

	currentHash, err := r.GenerateHashForDir(imageDir)
	if err != nil {
		return false, errors.Wrap(err, "could not generate hash")
	}
	prevHash, err := ioutil.ReadFile(constants.ChecksumFile)
	if err != nil {
		return false, err
	}
	res := bytes.Compare(currentHash, prevHash)
	if res == 0 {
		return true, nil
	} else {
		f := fileio.New()
		err = f.WriteToFile(constants.ChecksumFile, currentHash, false)
		if err != nil {
			return false, err
		}
		return false, nil
	}
}

//generates sha256 hash for files in directory and returns byte slice of hash of file
func (r *RuntimeUtil) GenerateHashForDir(imageDir string) ([]byte, error) {
	var data []byte
	items, _ := ioutil.ReadDir(imageDir)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else {
			imageFile := fmt.Sprintf("%s/%s", imageDir, item.Name())
			f, err := os.Open(imageFile)
			if err != nil {
				return data, err
			}
			defer f.Close()
			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				return data, err
			}
			data = h.Sum(data)
		}
	}
	return data, nil
}
