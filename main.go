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
	"sync"
	"unicode/utf8"

	"github.com/akinobufujii/similar_images_grouping/charcodeutil"
	"github.com/akinobufujii/similar_images_grouping/readimageutil"
	"github.com/bradhe/stopwatch"
	"github.com/corona10/goimagehash"
	"github.com/pkg/errors"
)

// streamSendWalkFilepath 指定ディレクトリ以下のwalk結果を返していくストリーム
func streamSendWalkFilepath(root string) <-chan string {
	ch := make(chan string, 1)
	go func() {
		walkFunc := func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return errors.Wrap(err, "failed filepath.WalkDir func")
			}

			if d.IsDir() {
				return nil
			}

			ch <- path
			return nil
		}

		// TODO: エラーハンドリング
		filepath.WalkDir(root, walkFunc)
		close(ch)
	}()

	return ch
}

// streamCalcImageHash 画像ハッシュ計算ストリーム
func streamCalcImageHash(inputStream <-chan string, samplew, sampleh, parallels int) <-chan ImageHashInfo {
	wg := sync.WaitGroup{}
	wg.Add(parallels)

	ch := make(chan ImageHashInfo, parallels)

	// TODO: リファクタリング
	sendImagehashResult := func(imageData image.Image, path string) {
		imagehash, err := goimagehash.ExtPerceptionHash(imageData, samplew, samplew)
		if err != nil {
			// TODO: エラーハンドリング
			return
		}

		ch <- ImageHashInfo{
			Filepath:  path,
			ImageHash: imagehash,
		}
	}

	// TODO: リファクタリング
	readImageFromZip := func(path string) {
		zipReader, err := zip.OpenReader(path)
		if err != nil {
			// NOTE: エラーハンドリング
			return
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
				// NOTE: エラーハンドリング
				continue
			}

			dispname := file.Name
			if !utf8.Valid([]byte(dispname)) {
				// NOTE: zipの中身はどうやらshiftjis
				newName, err := charcodeutil.SjisToUTF8(dispname)
				if err == nil {
					dispname = newName
				}
			}

			sendImagehashResult(imageData, filepath.Join(path, dispname))
		}
	}

	for i := 0; i < parallels; i++ {
		go func() {
			for path := range inputStream {
				// NOTE: 拡張子で処理を分岐
				switch strings.ToLower(filepath.Ext(path)) {
				case ".zip": // NOTE: zipファイル
					readImageFromZip(path)
				default: // NOTE: その他（画像ファイルとして判断）
					imageData, _, err := readimageutil.ReadImage(path)
					if err != nil {
						// NOTE: 読めなかったものはスルー
						continue
					}

					sendImagehashResult(imageData, path)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

// writeJson json書き込み
func writeJson(path string, targetData any) error {
	data, err := json.Marshal(targetData)
	if err != nil {
		return errors.Wrap(err, "failed json.Marshal")
	}

	buf := bytes.Buffer{}
	err = json.Indent(&buf, data, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed json.Indent")
	}

	file, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "failed os.Create: "+path)
	}

	file.Write(buf.Bytes())

	return nil
}

// streamSendImageHashFromFile 中間ファイルからImageHashInfoを送り続けるストリーム
func streamSendImageHashFromFile(filename string) <-chan ImageHashInfo {
	ch := make(chan ImageHashInfo, 1)

	data, err := os.ReadFile(filename)
	if err != nil {
		// TODO: error handring
		close(ch)
		return ch
	}

	// TODO: streaming read
	imageHashInfoList := ImageHashInfoList{}
	if err := json.Unmarshal(data, &imageHashInfoList); err != nil {
		// TODO: error handring
		close(ch)
		return ch
	}

	go func() {
		for _, info := range imageHashInfoList {
			ch <- info
		}
		close(ch)
	}()

	return ch
}

func main() {
	cmd := struct {
		Root                      string
		WriteIntermediateFilename string
		ReadIntermediateFilename  string
		Parallels                 int
		SampleWidth               int
		SampleHeight              int
		Threshold                 int
	}{}
	flag.StringVar(&cmd.Root, "root", "", "search dir")
	flag.StringVar(&cmd.WriteIntermediateFilename, "write-midfile", "midfile.json", "write intermediate filename(json)")
	flag.StringVar(&cmd.ReadIntermediateFilename, "read-midfile", "", "read intermediate filename(json)")

	flag.IntVar(&cmd.Parallels, "j", runtime.NumCPU(), "parallel num")
	flag.IntVar(&cmd.SampleWidth, "samplew", 16, "pHash width")
	flag.IntVar(&cmd.SampleHeight, "sampleh", 16, "pHash height")
	flag.IntVar(&cmd.Threshold, "threshold", 10, "pHash threshold")
	flag.Parse()

	isWriteMidFile := len(cmd.WriteIntermediateFilename) != 0
	isReadMidFile := len(cmd.ReadIntermediateFilename) != 0

	watch := stopwatch.Start()

	var ch <-chan ImageHashInfo
	if isReadMidFile {
		// NOTE: 中間ファイルから読み込んでその情報を受け取る
		ch = streamSendImageHashFromFile(cmd.ReadIntermediateFilename)
		isWriteMidFile = false
	} else {
		// NOTE: 並行して見つけた画像のハッシュを計算する
		rootPath := filepath.Clean(cmd.Root)
		parallels := cmd.Parallels
		if parallels < 1 {
			parallels = 1
		}
		ch = streamCalcImageHash(
			streamSendWalkFilepath(rootPath),
			cmd.SampleWidth, cmd.SampleHeight, parallels)
	}

	// NOTE: 要素をすべてコンテナに集約して比較する
	dataContainer := &ParallelCompList{}
	encodeList := ImageHashInfoList{}
	for info := range ch {
		dataContainer.Append(info)
		if isWriteMidFile {
			encodeList = append(encodeList, info)
		}
	}

	if isWriteMidFile && len(encodeList) > 0 {
		// NOTE: 復帰できるようにJsonファイルを保存する
		if err := writeJson(cmd.WriteIntermediateFilename, encodeList); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	watch.Stop()
	fmt.Printf("ReadFiles: %v\n", watch.String())

	watch = stopwatch.Start()

	// NOTE: 似ている画像をグルーピングする
	similarGroupsList := [][]string{}
	for !dataContainer.IsEmpty() {
		keydata := dataContainer.GetKeyData()
		if keydata == nil {
			// NOTE: ここに来ることはないはずだが念のためフェイルセーフしておく
			break
		}

		// NOTE: 似ている画像を獲得する
		similarGroups, err := dataContainer.GroupingSimilarImage(keydata, cmd.Threshold)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if len(similarGroups) > 0 {
			// NOTE: 一つ以上要素が入っていれば何かしら似ていると判定
			similarGroupsList = append(similarGroupsList, similarGroups)
		}

		// NOTE: dataContainerを比較が必要なものだけに要素を切り詰める
		dataContainer.Compaction()
	}

	watch.Stop()
	fmt.Printf("GroupingFiles: %v\n", watch.String())

	// TODO: csv書き出し
	if err := writeJson("similar_groups.json", similarGroupsList); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
