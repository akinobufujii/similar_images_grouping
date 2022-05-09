package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/akinobufujii/similar_images_grouping/charcodeutil"
	"github.com/akinobufujii/similar_images_grouping/readimageutil"
	"github.com/corona10/goimagehash"
	"github.com/pkg/errors"
)

type ImageHashInfo struct {
	Filepath  string
	ImageHash *goimagehash.ExtImageHash
}

type KeyData struct {
	Filepath  string
	ImageHash *goimagehash.ExtImageHash
	OnesBit   int
}

type OnesBitKeyImageHashMap map[int]*[]*ImageHashInfo

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
	wg := &sync.WaitGroup{}
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

		for _, file := range zipReader.File {
			dispname := file.Name
			reader, err := file.Open()
			if err != nil {
				// NOTE: エラーハンドリング
				continue
			}

			imageData, _, err := readimageutil.DecodeImage(reader)
			if err != nil {
				// NOTE: エラーハンドリング
				continue
			}

			if !utf8.Valid([]byte(dispname)) {
				// NOTE: zipの中身はどうやらshiftjis
				newName, err := charcodeutil.SjisToUTF8(file.Name)
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

// getKeyData キーデータを取得
func getKeyData(onesBitMap OnesBitKeyImageHashMap) *KeyData {
	// NOTE: 最初に見つかった要素をキーデータとする
	for onesbit, list := range onesBitMap {
		for i, info := range *list {
			keydata := &KeyData{
				Filepath:  info.Filepath,
				ImageHash: info.ImageHash,
				OnesBit:   onesbit,
			}

			// NOTE: この要素は比較する必要ないので消す
			//       本当はgetで消さずに後でいい感じにまとめて消したい
			(*list)[i] = nil
			return keydata
		}
	}
	return nil
}

// groupingSimilarImage 似ている画像をグルーピング
func groupingSimilarImage(onesBitMap OnesBitKeyImageHashMap, keydata KeyData, threshold int) []string {
	similarGroups := []string{}
	for onesbit, list := range onesBitMap {
		if int(math.Abs(float64(onesbit-keydata.OnesBit))) > threshold {
			// NOTE: ここと似ることはないはず
			continue
		}

		for i, info := range *list {
			if info == nil {
				continue
			}

			distance, err := keydata.ImageHash.Distance(info.ImageHash)
			if err != nil {
				// TODO: エラーハンドリング
				fmt.Fprint(os.Stderr, err)
				os.Exit(1)
			}

			if distance <= threshold {
				// NOTE: ここに入れば似ていると判定
				if len(similarGroups) == 0 {
					similarGroups = append(similarGroups, keydata.Filepath)
				}
				similarGroups = append(similarGroups, info.Filepath)

				// NOTE: すでに似ている判定されているので他と比較する必要はない
				(*list)[i] = nil
			}
		}
	}

	return similarGroups
}

// compactionOnesBitMap onesBitMapの切り詰めを行う
func compactionOnesBitMap(onesBitMap OnesBitKeyImageHashMap) {
	for onesbit, list := range onesBitMap {
		newList := []*ImageHashInfo{}
		for _, info := range *list {
			if info != nil {
				newList = append(newList, info)
			}
		}

		if len(newList) == 0 {
			delete(onesBitMap, onesbit)
		} else {
			onesBitMap[onesbit] = &newList
		}
	}
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

	// NOTE: 並行して見つけた画像のハッシュを計算する
	ch := streamCalcImageHash(
		streamSendWalkFilepath(rootPath),
		cmd.SampleWidth, cmd.SampleHeight, parallels)

	onesBitMap := OnesBitKeyImageHashMap{}
	for info := range ch {
		// NOTE: 後で比較するようにビットが立っている数によって先に割り振る
		//       比較アルゴリズムはビットの排他的論理和の結果、0に近ければ似ていると判断する
		//       そのため最初から立っているビット数がしきい値よりも離れていたらそもそも比較する必要がない
		onesCount := 0
		for _, data := range info.ImageHash.GetHash() {
			onesCount += bits.OnesCount64(data)
		}

		list, ok := onesBitMap[onesCount]
		if !ok {
			list = &[]*ImageHashInfo{}
			onesBitMap[onesCount] = list
		}

		*list = append(*list, &ImageHashInfo{Filepath: info.Filepath, ImageHash: info.ImageHash})
	}

	similarGroupsList := [][]string{}
	// NOTE: 似ている画像をグルーピングする
	for len(onesBitMap) > 0 {
		keydata := getKeyData(onesBitMap)

		// NOTE: 全部を比較する（比較して似ていたら消す）
		similarGroups := groupingSimilarImage(onesBitMap, *keydata, cmd.Threshold)

		if len(similarGroups) > 0 {
			// NOTE: 一つ以上要素が入っていれば何かしら似ていると判定
			similarGroupsList = append(similarGroupsList, similarGroups)
		}

		// NOTE: onesBitMapを比較が必要なものだけにするためのコンパクションを行う
		compactionOnesBitMap(onesBitMap)
	}

	// TODO: csv書き出し
	writeJson("similar_groups.json", similarGroupsList)
}
