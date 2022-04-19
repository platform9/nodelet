package embedutil_test

import (
	"embed"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"testing"

	"github.com/platform9/nodelet/nodelet/pkg/utils/embedutil"
)

//go:embed pf9/*
var content embed.FS

func TestEmbedUtil(t *testing.T) {
	t.Logf("TestEmbedUtil")
	outdir, err := ioutil.TempDir("/tmp", "embedtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(outdir)
	efs := embedutil.EmbedFS{Fs: content, Root: "pf9"}

	err = efs.Extract(outdir)
	if err != nil {
		t.Errorf("Failed to extract pf9-kube: %s", err)
	}
	// now check if the extracted directory is same as the src directory
	checkDirs(t, "pf9", outdir)
}

func checkDirs(t *testing.T, first string, second string) {

	secondFiles, err := ioutil.ReadDir(second)
	if err != nil {
		t.Fatalf("Failed to read dir %s: %v", second, err)
	}
	firstFiles, err := ioutil.ReadDir(first)
	if err != nil {
		t.Fatalf("Failed to read dir %s: %v", first, err)
	}
	if len(secondFiles) != len(firstFiles) {
		t.Fatalf("Expected %d files in %s, got %d", len(firstFiles), second, len(secondFiles))
	}

	sort.Slice(firstFiles, func(i, j int) bool {
		return firstFiles[i].Name() > firstFiles[j].Name()
	})

	sort.Slice(secondFiles, func(i, j int) bool {
		return secondFiles[i].Name() > secondFiles[j].Name()
	})

	for idx, ff := range firstFiles {
		sf := secondFiles[idx]
		if ff.IsDir() != sf.IsDir() {
			t.Fatalf("Expected dir attr to be same, got %s %v %s %v", ff.Name(), ff.IsDir(), sf.Name(), sf.IsDir())
		}
		if ff.Name() != sf.Name() {
			t.Fatalf("Expected file name to be same, got %s %s", ff.Name(), sf.Name())
		}

		if ff.IsDir() {
			checkDirs(t, path.Join(first, ff.Name()), path.Join(second, sf.Name()))
		}
	}
}
