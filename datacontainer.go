package main

import (
	"math/bits"

	"github.com/pkg/errors"
)

type OnesBitKeyImageHashMap map[int]*[]*ImageHashInfo

type KeyData struct {
	info    *ImageHashInfo
	onesBit int
}

// TODO: constraintsパッケージ使ったGenerics化を検討する
type signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// abs 絶対値
func abs[T signed](x T) T {
	if x < 0 {
		return -x
	}
	return x
}

// encodeThresholdBit しきい値のエンコード関数
func encodeThresholdBit(bitlist []uint64) int {
	onesCount := 0
	for i, data := range bitlist {
		onesCount |= bits.OnesCount64(data) << (i * 8)
	}

	return onesCount
}

// decodeThresholdBit しきい値のデコード関数
func decodeThresholdBit(x int) [4]int8 {
	result := [4]int8{}
	for i := range result {
		result[i] = int8((x >> (i * 8)) & 0x000000ff)
	}
	return result
}

// IsEmpty 要素があるかどうか
func (onesBitMap *OnesBitKeyImageHashMap) IsEmpty() bool {
	return len(*onesBitMap) == 0
}

// Append 要素追加
func (onesBitMap *OnesBitKeyImageHashMap) Append(info ImageHashInfo) {
	// NOTE: 後で比較するようにビットが立っている数によって先に割り振る
	//       比較アルゴリズムはビットの排他的論理和の結果、0に近ければ似ていると判断する
	//       そのため最初から立っているビット数がしきい値よりも離れていたらそもそも比較する必要がない
	// NOTE: 特定のビット数に偏っていることがあるため64ビット単位でしきい値計算できるようにする
	onesCount := encodeThresholdBit(info.ImageHash.GetHash())

	list, ok := (*onesBitMap)[onesCount]
	if !ok {
		list = &[]*ImageHashInfo{}
		(*onesBitMap)[onesCount] = list
	}

	*list = append(*list, &ImageHashInfo{Filepath: info.Filepath, ImageHash: info.ImageHash})
}

// GetKeyData キーデータを取得
func (onesBitMap *OnesBitKeyImageHashMap) GetKeyData() *KeyData {
	for onesbit, list := range *onesBitMap {
		for i, info := range *list {
			// NOTE: 最初に見つかった要素をキーデータとする
			keydata := info

			// NOTE:  この要素は比較する必要ないので消す
			// FIXME: 本当はgetで消さずに後でいい感じにまとめて消したい
			(*list)[i] = nil
			return &KeyData{info: keydata, onesBit: onesbit}
		}
	}
	return nil
}

// GroupingSimilarImage 似ている画像をグルーピング
func (onesBitMap *OnesBitKeyImageHashMap) GroupingSimilarImage(keydata *KeyData, threshold int) ([]string, error) {
	// NOTE: キーデータのビット数をデコード
	keyDataOnesbitList := decodeThresholdBit(keydata.onesBit)

	similarGroups := []string{}
	for onesbit, list := range *onesBitMap {
		// NOTE: 対象データと各ビット数の差を出してどれだけ異なるかを確認する
		onesbitList := decodeThresholdBit(onesbit)
		distance := 0
		for i := range keyDataOnesbitList {
			distance += int(abs(onesbitList[i] - keyDataOnesbitList[i]))
		}

		if distance > threshold {
			// NOTE: ここと似ることはないはず
			continue
		}

		for i, info := range *list {
			if info == nil {
				continue
			}

			distance, err := keydata.info.ImageHash.Distance(info.ImageHash)
			if err != nil {
				return similarGroups, errors.Wrap(err, "failed ImageHash.Distance")
			}

			if distance <= threshold {
				// NOTE: ここに入れば似ていると判定
				if len(similarGroups) == 0 {
					// NOTE： もし一番最初にグルーピングするならkeydataも含める必要がある
					similarGroups = append(similarGroups, keydata.info.Filepath)
				}
				similarGroups = append(similarGroups, info.Filepath)

				// NOTE: すでに似ている判定されているので他と比較する必要はない
				(*list)[i] = nil
			}
		}
	}

	return similarGroups, nil
}

// Compaction 要素の切り詰めを行う
func (onesBitMap *OnesBitKeyImageHashMap) Compaction() {
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
