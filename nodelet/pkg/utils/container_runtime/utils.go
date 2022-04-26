package containerruntime

//  Note:  There are only e2e tests for containerd… No unit tests

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
)

type Runtime interface {
	LoadImagesFromDir(context.Context, string, string) error
	LoadImagesFromFile(context.Context, string) error
}

type RuntimeUtil struct{}

func New() Runtime {
	return &RuntimeUtil{}
}

// LoadImagesFromDir loads images from all tar files in the given directory to container runtime with given namespace
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

// LoadImagesFromFile loads images from given tar file to container runtime
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

	client, err := containerd.New(constants.ContainerdSocket, containerd.WithDefaultPlatform(platform))
	if err != nil {
		return errors.Wrap(err, "failed to create container runtime client")
	}

	imgs, err := client.Import(ctx, decompressor, containerd.WithDigestRef(archive.DigestTranslator(constants.DefaultSnapShotter)), containerd.WithSkipDigestRef(func(name string) bool { return name != "" }), containerd.WithImportPlatform(platform))
	if err != nil {
		return errors.Wrapf(err, "failed to import images from: %s", fileName)
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
