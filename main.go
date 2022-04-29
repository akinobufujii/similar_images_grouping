package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/akinobufujii/similar_images_grouping/readfileutil"
	"github.com/akinobufujii/similar_images_grouping/readimageutil"
	"github.com/corona10/goimagehash"
)

func main() {
	cmd := struct {
		Root         string
		SampleWidth  int
		SampleHeight int
	}{}
	flag.StringVar(&cmd.Root, "root", "", "search dir")
	flag.IntVar(&cmd.SampleWidth, "samplew", 16, "pHash width")
	flag.IntVar(&cmd.SampleHeight, "sampleh", 16, "pHash height")
	flag.Parse()

	rootPath := filepath.Clean(cmd.Root)

	// NOTE: ディレクトリ内検査
	filelist, err := readfileutil.GetFilelistFromDir(rootPath)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	for _, path := range filelist {
		// TODO: 並列化
		imageData, err := readimageutil.ReadImage(path)
		if err != nil {
			// NOTE: 読めなかったものはスルー
			continue
		}

		// NOTE: pHashを計算
		imagehash, err := goimagehash.ExtPerceptionHash(imageData, cmd.SampleWidth, cmd.SampleHeight)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}

		// TODO: 比較アルゴリズム
		fmt.Println(imagehash.GetHash())
		fmt.Println(imagehash.ToString())
	}
}
