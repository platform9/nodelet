package phaseutils

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/archive/compression"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/platforms"
	"github.com/platform9/nodelet/nodelet/pkg/utils/constants"
	sunpikev1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

func SetHostStatus(hostPhase *sunpikev1alpha1.HostPhase, status string, message string) {
	//TODO: Retry
	hostPhase.Status = status
	hostPhase.Message = message
}

func LoadImagesFromDir(ctx context.Context, imageDir string) error {
	items, _ := ioutil.ReadDir(imageDir)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else {
			imageFile := fmt.Sprintf("%s/%s", imageDir, item.Name())
			err := LoadImagesFromFile(ctx, imageFile)
			if err != nil {
				fmt.Printf("could not load images from : %v, because : %v", imageFile, err)
				return err
			}

		}
	}
	return nil
}
func LoadImagesFromFile(ctx context.Context, input string) error {

	f, err := os.Open(input)
	if err != nil {
		return err
	}
	decompressor, err := compression.DecompressStream(f)
	if err != nil {
		return err
	}

	platform := platforms.DefaultStrict()

	client, err := containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultPlatform(platform)) //containerd.WithDefaultPlatform(platform))
	if err != nil {
		return fmt.Errorf("failed to create client:%w", err)
	}
	ctx = namespaces.WithNamespace(ctx, "k8s.io")

	snapShotter := "overlayfs"

	imgs, err := client.Import(ctx, decompressor, containerd.WithDigestRef(archive.DigestTranslator(snapShotter)), containerd.WithSkipDigestRef(func(name string) bool { return name != "" }), containerd.WithImportPlatform(platform))
	if err != nil {
		return fmt.Errorf("failed to import :%w", err)
	}
	for _, img := range imgs {
		image := containerd.NewImageWithPlatform(client, img, platform)

		fmt.Printf("unpacking %s (%s)...", img.Name, img.Target.Digest)

		err = image.Unpack(ctx, snapShotter)
		if err != nil {
			return err
		}
		fmt.Println("done")
	}
	return nil
}

func GenerateChecksum(imageDir string) error {

	var data []byte
	items, _ := ioutil.ReadDir(imageDir)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else {
			imageFile := fmt.Sprintf("%s/%s", imageDir, item.Name())
			f, err := os.Open(imageFile)
			if err != nil {
				log.Fatal(err)
				return err
			}
			defer f.Close()
			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				log.Fatal(err)
				return err
			}
			data = h.Sum(data)
		}
	}

	if _, err := os.Stat(constants.ChecksumDir); os.IsNotExist(err) {
		if err := os.Mkdir(constants.ChecksumDir, os.ModePerm); err != nil {
			log.Fatal(err)
			return err
		}
	}
	err := ioutil.WriteFile(constants.ChecksumFile, data, 0644)
	if err != nil {
		fmt.Printf("failed to write checksum file")
		return err
	}
	return nil
}
func VerifyChecksum(imageDir string) (bool, error) {

	var data []byte
	items, _ := ioutil.ReadDir(imageDir)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else {
			imageFile := fmt.Sprintf("%s/%s", imageDir, item.Name())
			f, err := os.Open(imageFile)
			if err != nil {
				log.Fatal(err)
				return false, err
			}
			defer f.Close()
			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				log.Fatal(err)
				return false, err
			}
			data = h.Sum(data)
		}
	}
	//checksumFile := fmt.Sprintf("%s/checksum/sha256sums", imageDir)
	actualdata, err := ioutil.ReadFile(constants.ChecksumFile)
	if err != nil {
		fmt.Printf("failed to read file: %v", err)
		return false, err
	}
	res := bytes.Compare(actualdata, data)
	if res == 0 {
		return true, nil
	} else {
		return false, nil
	}
}
