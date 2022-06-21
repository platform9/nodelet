package pf9kube

import (
	"embed"
	"fmt"

	"github.com/platform9/nodelet/nodelet/pkg/embedutil"
	"go.uber.org/zap"
)

//go:embed pf9/*
var kube embed.FS

//go:embed etc/*
var etc embed.FS

//go:embed lib/*
var lib embed.FS

func Extract() error {
	zap.S().Infof("Extracting pf9-kube to '%s'", "/opt/pf9/")
	efs := &embedutil.EmbedFS{Fs: kube, Root: "pf9"}
	err := efs.Extract("/opt/pf9/")
	if err != nil {
		return fmt.Errorf("failed to extract pf9-kube to /opt/pf9: %s", err)
	}
	zap.S().Infof("Extracting etc to '%s'", "/etc/")
	efs = &embedutil.EmbedFS{Fs: etc, Root: "etc"}
	err = efs.Extract("/etc")
	if err != nil {
		return fmt.Errorf("failed to extract pf9-kube to /etc: %s", err)
	}
	zap.S().Info("Extracting lib to /lib")
	efs = &embedutil.EmbedFS{Fs: lib, Root: "lib"}
	err = efs.Extract("/lib")
	if err != nil {
		return fmt.Errorf("failed to extract pf9-kube to /lib: %s", err)
	}
	return nil
}
