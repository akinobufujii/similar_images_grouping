package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/akinobufujii/similar_images_grouping/charcodeutil"
	"github.com/akinobufujii/similar_images_grouping/readimageutil"
	"github.com/bradhe/stopwatch"
	"github.com/corona10/goimagehash"
	"golang.org/x/sync/errgroup"
)

// writeJson json書き込み
func writeJson(path string, targetData any) error {
	data, err := json.Marshal(targetData)
	if err != nil {
		return fmt.Errorf("failed json.Marshal: %s %w", path, err)
	}

	buf := bytes.Buffer{}
	err = json.Indent(&buf, data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed json.Indent: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed os.Create: %s %w", path, err)
	}
	defer file.Close()

	file.Write(buf.Bytes())

	return nil
}

// calcImageHash 画像ハッシュ計算関数
func calcImageHash(imageData image.Image, path string, samplew, sampleh int) (*ImageHashInfo, error) {
	imagehash, err := goimagehash.ExtPerceptionHash(imageData, samplew, sampleh)
	if err != nil {
		return nil, fmt.Errorf("failed goimagehash.ExtPerceptionHash: %w", err)
	}

	imageHash := &ImageHashInfo{
		Filepath:  path,
		ImageHash: imagehash,
	}

	return imageHash, nil
}

// readImageFromZip zipファイルから画像を読み込み、指定のチャネルに送信する
func readImageFromZip(path string, chCalcImagehash chan<- *ImageHashInfo, samplew, sampleh int) error {
	zipReader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("failed zip.OpenReader: %s %w", path, err)
	}
	defer zipReader.Close()

	getImageData := func(file *zip.File) (image.Image, error) {
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		imageData, _, err := readimageutil.DecodeImage(reader)
		if err != nil {
			return nil, err
		}
		return imageData, nil
	}

	for _, file := range zipReader.File {
		imageData, err := getImageData(file)
		if err != nil {
			return fmt.Errorf("failed getImageData: %w", err)
		}

		dispname := file.Name
		if !utf8.Valid([]byte(dispname)) {
			// NOTE: zipの中身はどうやらshiftjis
			newName, err := charcodeutil.SjisToUTF8(dispname)
			if err == nil {
				dispname = newName
			}
		}

		imageHash, err := calcImageHash(imageData, filepath.Join(path, dispname), samplew, sampleh)
		if err != nil {
			return fmt.Errorf("failed calcImageHash: %s %w", path, err)
		}
		chCalcImagehash <- imageHash
	}

	return nil
}

// createParallelCompList ParallelCompListを作成する
func createParallelCompList(container *ParallelCompList, root string, samplew, sampleh, parallels int) error {
	eg := errgroup.Group{}

	// NOTE: ファイルのパスを送り続けるgoroutine
	chPath := make(chan string, 1)
	eg.Go(func() error {
		defer close(chPath)
		return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("failed filepath.WalkDir func: %w", err)
			}

			if d.IsDir() {
				return nil
			}

			chPath <- path
			return nil
		})
	})

	// NOTE: 画像のハッシュを計算し続けるgoroutine
	chCalcImagehash := make(chan *ImageHashInfo, parallels)
	for i := 0; i < parallels; i++ {
		eg.Go(func() error {
			for path := range chPath {
				// NOTE: 拡張子で処理を分岐
				switch strings.ToLower(filepath.Ext(path)) {
				case ".zip": // NOTE: zipファイル
					err := readImageFromZip(path, chCalcImagehash, samplew, sampleh)
					if err != nil {
						return fmt.Errorf("failed readImageFromZip: %s %w", path, err)
					}
				default: // NOTE: その他（画像ファイルとして判断）
					imageData, _, err := readimageutil.ReadImage(path)
					if err != nil {
						// NOTE: 読めなかったものはスルー
						continue
					}

					imageHash, err := calcImageHash(imageData, path, samplew, sampleh)
					if err != nil {
						return fmt.Errorf("failed calcImageHash: %s %w", path, err)
					}
					chCalcImagehash <- imageHash
				}
			}
			return nil
		})
	}

	go func() {
		eg.Wait()
		close(chCalcImagehash)
	}()

	for imageHash := range chCalcImagehash {
		container.Append(imageHash)
	}

	return eg.Wait()
}

func main() {
	cmd := struct {
		Root                      string
		WriteIntermediateFilename string
		ReadIntermediateFilename  string
		Output                    string
		Parallels                 int
		SampleWidth               int
		SampleHeight              int
		Threshold                 int
	}{}
	flag.StringVar(&cmd.Root, "root", "", "search dir")
	flag.StringVar(&cmd.WriteIntermediateFilename, "write-midfile", "midfile.json", "write intermediate filename(json)")
	flag.StringVar(&cmd.ReadIntermediateFilename, "read-midfile", "", "read intermediate filename(json)")
	flag.StringVar(&cmd.Output, "o", "similar_groups.json", "output filename(json)")

	flag.IntVar(&cmd.Parallels, "j", runtime.NumCPU(), "parallel num")
	flag.IntVar(&cmd.SampleWidth, "samplew", 16, "pHash width")
	flag.IntVar(&cmd.SampleHeight, "sampleh", 16, "pHash height")
	flag.IntVar(&cmd.Threshold, "threshold", 10, "pHash threshold")
	flag.Parse()

	isWriteMidFile := len(cmd.WriteIntermediateFilename) != 0
	isReadMidFile := len(cmd.ReadIntermediateFilename) != 0

	watch := stopwatch.Start()

	// NOTE: 要素をすべてコンテナに集約して比較する
	container := &ParallelCompList{}
	if isReadMidFile {
		// NOTE: 中間ファイルがあるならそれをデシリアライズする
		err := container.Deserialize(cmd.ReadIntermediateFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		// NOTE: 並行して見つけた画像のハッシュを計算する
		rootPath := filepath.Clean(cmd.Root)
		parallels := cmd.Parallels
		if parallels < 1 {
			parallels = 1
		}
		err := createParallelCompList(container, rootPath, cmd.SampleWidth, cmd.SampleHeight, parallels)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if isWriteMidFile && !container.IsEmpty() {
			// NOTE: 復帰できるようにSerializeしてファイル保存する
			err := container.Serialize(cmd.WriteIntermediateFilename)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}

	watch.Stop()
	fmt.Printf("ReadFiles: %v\n", watch.String())

	watch = stopwatch.Start()

	// NOTE: 似ている画像をグルーピングする
	similarGroupsList := [][]string{}
	for !container.IsEmpty() {
		// NOTE: 似ている画像を獲得する
		similarGroups, err := container.GroupingSimilarImage(cmd.Threshold)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if len(similarGroups) > 0 {
			// NOTE: 一つ以上要素が入っていれば何かしら似ていると判定
			similarGroupsList = append(similarGroupsList, similarGroups)
		}

		// NOTE: containerを比較が必要なものだけに要素を切り詰める
		container.Compaction()
	}

	watch.Stop()
	fmt.Printf("GroupingFiles: %v\n", watch.String())

	// NOTE: json書き出し
	if err := writeJson(cmd.Output, similarGroupsList); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
