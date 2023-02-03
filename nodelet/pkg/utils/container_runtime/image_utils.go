package containerruntime

//  Note:  There are only e2e tests for containerd… No unit tests

import (
	"context"
	"fmt"
	"github.com/containerd/containerd/leases"
	"io/ioutil"
	"os"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/pkg/errors"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	"go.uber.org/zap"
)

type ImageUtility struct{}

func NewImageUtil() ImageUtils {
	return &ImageUtility{}
}

// LoadImagesFromDir loads images from all tar files in the given directory to container runtime with given namespace
func (i *ImageUtility) LoadImagesFromDir(ctx context.Context, imageDir string, namespace string) error {
	items, err := ioutil.ReadDir(imageDir)
	if err != nil {
		return errors.Wrapf(err, "could not read dir: %s", imageDir)
	}
	for _, item := range items {
		if !item.IsDir() {
			imageFile := fmt.Sprintf("%s/%s", imageDir, item.Name())
			ctx = namespaces.WithNamespace(ctx, namespace)
			err := i.LoadImagesFromFile(ctx, imageFile)
			if err != nil {
				return errors.Wrapf(err, "could not load images from : %s", imageFile)
			}
		}
	}
	return nil
}

// LoadImagesFromFile loads images from given tar file to container runtime
func (i *ImageUtility) LoadImagesFromFile(ctx context.Context, fileName string) error {
	zap.S().Infof("Loading images from file: %s", fileName)

	// Commenting out below until we can better understand containerd Golang client or there is some documentation
	// seeing inconsistencies with importing images (as well as pushing/pulling)

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
	defer client.Close()

	// Create a lease with random ID and add that to context, so all the images imported with this context is never garbage collected

	zap.S().Infof("Creating a lease label")
	leaseLabels := make(map[string]string)
	leaseExpiryTime := time.Now().Add(5000 * time.Hour).UTC().Format(time.RFC3339)
	zap.S().Infof("Setting expiry to %v", leaseExpiryTime)

	leaseLabels["containerd.io/gc.expire"] = leaseExpiryTime
	manager := client.LeasesService()
	l, err := manager.Create(ctx, leases.WithRandomID(), leases.WithLabels(leaseLabels))
	zap.S().Infof("Created a lease with ID: %s", l.ID)
	if err != nil {
		return errors.Wrap(err, "failed to create leases with RandomID")
	}
	ctx = leases.WithLease(ctx, l.ID)
	zap.S().Infof("Assigned context to the lease")
	lid, errBool := leases.FromContext(ctx)
	zap.S().Infof("Lease ID from context: %s, err_bool: %t", lid, errBool)

	imgs, err := client.Import(ctx, decompressor, containerd.WithDigestRef(archive.DigestTranslator(constants.DefaultSnapShotter)), containerd.WithSkipDigestRef(func(name string) bool { return name != "" }), containerd.WithImportPlatform(platform))
	if err != nil {
		return errors.Wrapf(err, "failed to import images from: %s", fileName)
	}
	for _, img := range imgs {
		image := containerd.NewImageWithPlatform(client, img, platform)
		zap.S().Infof("not Unpacking image: %s", image.Name())

		//imggg, _ := client.GetImage(ctx, "cdsca")
		//imggg.Labels()
		//client.ContentStore().Info()
		//
		//imggg.Metadata().RootFS()
		/* Unpacking and snapshotting may not be needed. They also consume double the disk space
		 * as it makes a copy of each layer and repliates the filesystem then differs next layer, so forth
		 * it also adds time to unpack and snapshot each image.
		 * It does not appear to be necessary, k8s runtime seems to do this when it createsa new container
		 * leaving commented out for now
		 */
		resource := leases.Resource{ID: image.Name(), Type: "image"}
		zap.S().Infof("Adding resource %s to lease %s", image.Name(), l.ID)
		err = manager.AddResource(ctx, l, resource)
		if err != nil {
			return errors.Wrapf(err, "failed to add resource image %s to lease", image.Name())
		}
		err = image.Unpack(ctx, constants.DefaultSnapShotter)
		if err != nil {
			return errors.Wrapf(err, "failed to unpack image: %s", image.Name())
		}

		diffids, err := image.RootFS(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to get RootFS: %s", image.Name())
		}

		for _, diffId := range diffids {
			info, err := client.ContentStore().Info(ctx, diffId)
			if err != nil {
				return errors.Wrapf(err, "failed to get info for a content digest: %s", image.Name())
			}

			contentLabels := info.Labels
			if contentLabels == nil || len(contentLabels) == 0 {
				contentLabels = make(map[string]string)
			}

			contentLabels["containerd.io/gc.root"] = time.Now().UTC().Format(time.RFC3339)
			info.Labels = contentLabels

			_, err = client.ContentStore().Update(ctx, info)
			if err != nil {
				return errors.Wrapf(err, "failed to update info for a content digest: %s", image.Name())
			}
		}

	}

	lid2, errBool2 := leases.FromContext(ctx)
	zap.S().Infof("Lease ID from context after image import: %s, err_bool: %t", lid2, errBool2)

	return nil
}
