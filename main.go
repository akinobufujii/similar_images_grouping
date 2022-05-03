package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"math/bits"
	"os"
	"path/filepath"
	"sync"

	"github.com/akinobufujii/similar_images_grouping/readfileutil"
	"github.com/akinobufujii/similar_images_grouping/readimageutil"
	"github.com/corona10/goimagehash"
	"github.com/pkg/errors"
)

type ImageDataInfo struct {
	Filepath  string
	ImageData image.Image
}

type ImageHashInfo struct {
	Filepath  string
	ImageHash *goimagehash.ExtImageHash
}

type KeyData struct {
	Filepath  string
	ImageHash *goimagehash.ExtImageHash
	OnesBit   int
}

// streamCalcImageHash 画像ハッシュ計算ストリーム
func streamCalcImageHash(inputStream <-chan ImageDataInfo, samplew, sampleh, int, parallels int) <-chan ImageHashInfo {
	wg := sync.WaitGroup{}
	wg.Add(parallels)

	ch := make(chan ImageHashInfo, parallels)
	for i := 0; i < parallels; i++ {
		go func() {
			for info := range inputStream {
				imagehash, err := goimagehash.ExtPerceptionHash(info.ImageData, samplew, samplew)
				if err != nil {
					// TODO: エラーハンドリング
					continue
				}

				result := ImageHashInfo{}
				result.Filepath = info.Filepath
				result.ImageHash = imagehash
				ch <- result
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
		SampleWidth  int
		SampleHeight int
		Threshold    int
	}{}
	flag.StringVar(&cmd.Root, "root", "", "search dir")
	flag.IntVar(&cmd.SampleWidth, "samplew", 16, "pHash width")
	flag.IntVar(&cmd.SampleHeight, "sampleh", 16, "pHash height")
	flag.IntVar(&cmd.Threshold, "threshold", 10, "pHash threshold")
	flag.Parse()

	rootPath := filepath.Clean(cmd.Root)

	// NOTE: ディレクトリ内検査
	filelist, err := readfileutil.GetFilelistFromDir(rootPath)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	//chInput := make(chan ImageDataInfo, 1)
	//chOutput := streamCalcImageHash(chInput, cmd.SampleWidth, cmd.SampleHeight, runtime.NumCPU())

	onesBitMap := map[int]*[]ImageHashInfo{}
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

		// NOTE: 後で比較するようにビットが立っている数によって先に割り振る
		//       比較アルゴリズムはビットの排他的論理和の結果、0に近ければ似ていると判断する
		//       そのため最初からビット数がしきい値よりも離れていたらそもそも比較する必要がない
		onesCount := 0
		for _, data := range imagehash.GetHash() {
			onesCount += bits.OnesCount64(data)
		}

		list, ok := onesBitMap[onesCount]
		if !ok {
			list = &[]ImageHashInfo{}
			onesBitMap[onesCount] = list
		}

		*list = append(*list, ImageHashInfo{Filepath: path, ImageHash: imagehash})
	}
	//close(chInput)

	// for info := range chOutput {

	// }

	writeJson("result.json", onesBitMap)

	// TODO: 比較アルゴリズム実装
	// similarGroupsList := [][]string{}

	// // NOTE: 最初に見つかった対象を比較対象のキーデータにする
	// keydata := KeyData{}
	// for onesbit, list := range onesBitMap {
	// 	for _, info := range *list {
	// 		keydata.Filepath = info.Filepath
	// 		keydata.ImageHash = info.ImageHash
	// 		keydata.OnesBit = onesbit
	// 		break
	// 	}
	// 	if len(keydata.Filepath) > 0 && keydata.ImageHash != nil {
	// 		break
	// 	}
	// }

	// // NOTE: 全部を比較する（比較して似ていたら消す）
	// similarGroups := []string{}
	// removeOnesbitKey := -1
	// removeIndex := -1
	// for onesbit, list := range onesBitMap {
	// 	if int(math.Abs(float64(onesbit-keydata.OnesBit))) > cmd.Threshold {
	// 		// NOTE: ここと似ることはないはず
	// 		continue
	// 	}

	// 	for i, info := range *list {
	// 		if keydata.ImageHash == info.ImageHash && keydata.Filepath == info.Filepath {
	// 			// NOTE: 同じ画像なので後で削除する
	// 			removeOnesbitKey = onesbit
	// 			removeIndex = i
	// 			continue
	// 		}

	// 		distance, err := keydata.ImageHash.Distance(info.ImageHash)
	// 		if err != nil {
	// 			fmt.Fprint(os.Stderr, err)
	// 			os.Exit(1)
	// 		}

	// 		if distance < cmd.Threshold {
	// 			// NOTE: ここに入れば似ていると判定
	// 			if len(similarGroups) == 0 {
	// 				similarGroups = append(similarGroups, keydata.Filepath)
	// 			}
	// 			similarGroups = append(similarGroups, info.Filepath)
	// 		}
	// 	}
	// }

	// // NOTE: 比較終わったらキーデータをonesBitMapから消す
	// if removeOnesbitKey >= 0 && removeIndex >= 0 {
	// 	removeSliceNoSort := func(array []ImageHashInfo, i int) []ImageHashInfo {
	// 		array[i] = array[len(array)-1]
	// 		return array[:len(array)-1]
	// 	}
	// 	*onesBitMap[removeOnesbitKey] = removeSliceNoSort(*onesBitMap[removeOnesbitKey], removeIndex)
	// }

	// if len(similarGroups) > 0 {
	// 	similarGroupsList = append(similarGroupsList, similarGroups)
	// }

	// // NOTE: onesBitMapの内容が何かあれば最初に戻る

	// // TODO: csv書き出し
	// fmt.Println(similarGroupsList)
}
