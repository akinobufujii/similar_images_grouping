package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
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

type ImageHashInfo struct {
	Filepath  string
	ImageHash *goimagehash.ExtImageHash
}

// EncodeJson Jsonデータにエンコード
func (p *ImageHashInfo) EncodeJson(enc *json.Encoder) error {
	b := bytes.Buffer{}
	writer := bufio.NewWriter(&b)
	if err := p.ImageHash.Dump(writer); err != nil {
		return errors.Wrap(err, "failed ImageHash.Dump")
	}

	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "failed Flush")
	}

	encodeData := struct {
		Filepath      string
		ImageHashDump string
	}{
		Filepath:      p.Filepath,
		ImageHashDump: base64.StdEncoding.EncodeToString(b.Bytes()),
	}

	if err := enc.Encode(encodeData); err != nil {
		return errors.Wrap(err, "failed Encode")
	}

	return nil
}

// DecodeJson Jsonデータにデコード
func (p *ImageHashInfo) DecodeJson(dec *json.Decoder) error {
	decodeData := struct {
		Filepath      string
		ImageHashDump string
	}{}

	err := dec.Decode(&decodeData)
	if err != nil {
		return errors.Wrap(err, "failed Decode")
	}

	data, err := base64.StdEncoding.DecodeString(decodeData.ImageHashDump)
	if err != nil {
		return errors.Wrap(err, "failed DecodeString")
	}

	reader := bufio.NewReader(bytes.NewBuffer(data))
	p.ImageHash, err = goimagehash.LoadExtImageHash(reader)
	if err != nil {
		return errors.Wrap(err, "failed LoadExtImageHash")
	}
	p.Filepath = decodeData.Filepath

	return nil
}

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

func main() {
	cmd := struct {
		Root         string
		Parallels    int
		SampleWidth  int
		SampleHeight int
		Threshold    int
	}{}
	flag.StringVar(&cmd.Root, "root", "", "search dir")
	flag.IntVar(&cmd.Parallels, "j", runtime.NumCPU(), "parallel num")
	flag.IntVar(&cmd.SampleWidth, "samplew", 16, "pHash width")
	flag.IntVar(&cmd.SampleHeight, "sampleh", 16, "pHash height")
	flag.IntVar(&cmd.Threshold, "threshold", 10, "pHash threshold")
	flag.Parse()

	rootPath := filepath.Clean(cmd.Root)
	parallels := cmd.Parallels
	if parallels < 1 {
		parallels = 1
	}

	watch := stopwatch.Start()

	// NOTE: 並行して見つけた画像のハッシュを計算する
	ch := streamCalcImageHash(
		streamSendWalkFilepath(rootPath),
		cmd.SampleWidth, cmd.SampleHeight, parallels)

	// NOTE: 要素をすべてコンテナに集約して比較する
	onesBitMap := OnesBitKeyImageHashMap{}
	for info := range ch {
		onesBitMap.Append(info)
	}

	watch.Stop()
	fmt.Printf("ReadFiles: %v\n", watch.String())

	watch = stopwatch.Start()

	// NOTE: 似ている画像をグルーピングする
	similarGroupsList := [][]string{}
	for !onesBitMap.IsEmpty() {
		keydata, keydataOnesBit := onesBitMap.GetKeyData()
		if keydata == nil {
			// NOTE: ここに来ることはないはずだが念のためフェイルセーフしておく
			break
		}

		// NOTE: 似ている画像を獲得する
		similarGroups, err := onesBitMap.GroupingSimilarImage(keydata, keydataOnesBit, cmd.Threshold)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if len(similarGroups) > 0 {
			// NOTE: 一つ以上要素が入っていれば何かしら似ていると判定
			similarGroupsList = append(similarGroupsList, similarGroups)
		}

		// NOTE: onesBitMapを比較が必要なものだけに要素を切り詰める
		onesBitMap.CompactionOnesBitMap()
	}

	watch.Stop()
	fmt.Printf("GroupingFiles: %v\n", watch.String())

	// TODO: csv書き出し
	writeJson("similar_groups.json", similarGroupsList)
}
