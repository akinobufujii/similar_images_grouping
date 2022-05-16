package main

import (
	"math"
	"math/bits"

	"github.com/pkg/errors"
)

type OnesBitKeyImageHashMap map[int]*[]*ImageHashInfo

// IsEmpty 要素があるかどうか
func (onesBitMap *OnesBitKeyImageHashMap) IsEmpty() bool {
	return len(*onesBitMap) == 0
}

// Append 要素追加
func (onesBitMap *OnesBitKeyImageHashMap) Append(info ImageHashInfo) {
	// NOTE: 後で比較するようにビットが立っている数によって先に割り振る
	//       比較アルゴリズムはビットの排他的論理和の結果、0に近ければ似ていると判断する
	//       そのため最初から立っているビット数がしきい値よりも離れていたらそもそも比較する必要がない
	onesCount := 0
	for _, data := range info.ImageHash.GetHash() {
		onesCount += bits.OnesCount64(data)
	}

	list, ok := (*onesBitMap)[onesCount]
	if !ok {
		list = &[]*ImageHashInfo{}
		(*onesBitMap)[onesCount] = list
	}

	*list = append(*list, &ImageHashInfo{Filepath: info.Filepath, ImageHash: info.ImageHash})
}

// GetKeyData キーデータを取得
func (onesBitMap *OnesBitKeyImageHashMap) GetKeyData() (*ImageHashInfo, int) {
	for onesbit, list := range *onesBitMap {
		for i, info := range *list {
			// NOTE: 最初に見つかった要素をキーデータとする
			keydata := info

			// NOTE:  この要素は比較する必要ないので消す
			// FIXME: 本当はgetで消さずに後でいい感じにまとめて消したい
			(*list)[i] = nil
			return keydata, onesbit
		}
	}
	return nil, 0
}

// GroupingSimilarImage 似ている画像をグルーピング
func (onesBitMap *OnesBitKeyImageHashMap) GroupingSimilarImage(keydata *ImageHashInfo, keyDataOnesbit, threshold int) ([]string, error) {
	similarGroups := []string{}
	for onesbit, list := range *onesBitMap {
		if int(math.Abs(float64(onesbit-keyDataOnesbit))) > threshold {
			// NOTE: ここと似ることはないはず
			continue
		}

		for i, info := range *list {
			if info == nil {
				continue
			}

			distance, err := keydata.ImageHash.Distance(info.ImageHash)
			if err != nil {
				return similarGroups, errors.Wrap(err, "failed ImageHash.Distance")
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

	return similarGroups, nil
}

// CompactionOnesBitMap 要素の切り詰めを行う
func (onesBitMap *OnesBitKeyImageHashMap) CompactionOnesBitMap() {
	for onesbit, list := range *onesBitMap {
		newList := []*ImageHashInfo{}
		for _, info := range *list {
			if info != nil {
				newList = append(newList, info)
			}
		}

		if len(newList) == 0 {
			delete(*onesBitMap, onesbit)
		} else {
			(*onesBitMap)[onesbit] = &newList
		}
	}
}
