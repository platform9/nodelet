package pf9kube

import (
	"embed"

	"github.com/platform9/nodelet/nodelet/pkg/embedutil"
	"go.uber.org/zap"
)

//go:embed pf9/*
var content embed.FS

func Extract(fs embed.FS) error {
	zap.S().Infof("Extracting pf9-kube to '%s'", "/opt/pf9/")
	efs := &embedutil.EmbedFS{fs: content, root: "/opt/pf9"}
	return efs.Extract(content, "/opt/pf9/")
}
