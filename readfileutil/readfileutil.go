package readfileutil

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// GetFilelistFromDir 指定ディレクトリからファイルリスト獲得
func GetFilelistFromDir(root string) ([]string, error) {
	filelist := []string{}

	walkFunc := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return errors.Wrap(err, "failed filepath.WalkDir func")
		}

		if d.IsDir() {
			return nil
		}

		filelist = append(filelist, path)
		return nil
	}

	err := filepath.WalkDir(root, walkFunc)
	if err != nil {
		return filelist, errors.Wrap(err, "failed filepath.WalkDir")
	}

	return filelist, nil
}
