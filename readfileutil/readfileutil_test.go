package readfileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFilelistFromDir(t *testing.T) {
	root := filepath.Clean(os.Getenv("GOPATH"))
	filelist, err := GetFilelistFromDir(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range filelist {
		t.Log(path)
	}
}
