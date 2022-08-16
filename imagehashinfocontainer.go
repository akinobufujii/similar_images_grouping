package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/corona10/goimagehash"
	"golang.org/x/sync/errgroup"
)

type ParallelCompList []*ImageHashInfo

func (container *ParallelCompList) IsEmpty() bool {
	return len(*container) == 0
}

func (container *ParallelCompList) Append(info *ImageHashInfo) {
	*container = append(*container, info)
}

func compSrcImagehash(ch chan<- string, view ParallelCompList, srcImageHash *goimagehash.ExtImageHash, threshold int) error {
	for i, data := range view {
		if data == nil {
			continue
		}

		distance, err := srcImageHash.Distance(data.ImageHash)
		if err != nil {
			return fmt.Errorf("failed ImageHash.Distance: %w", err)
		}

		if distance <= threshold {
			// NOTE: 似てるという判定
			// NOTE: ここで同時にcontainerに書き込みアクセスするが
			//       別々の内容に同時に書き込むだけなので大丈夫なはず
			ch <- data.Filepath
			view[i] = nil
		}
	}
	return nil
}

func (container *ParallelCompList) GroupingSimilarImage(threshold int) ([]string, error) {
	// NOTE: 論理スレッド数分goroutineを生成し
	//       その中で比較元の内容と近いかどうかを総当たりで全比較する
	containerSize := len(*container)
	if containerSize <= 1 {
		// NOTE: 比較するものがないので空にして抜ける
		*container = make(ParallelCompList, 0)
		return nil, nil
	}

	// NOTE: 最初の要素を比較元にする
	src := (*container)[0]

	// NOTE: 残りの要素と比較するのでずらす
	(*container) = (*container)[1:]
	containerSize -= 1

	parallels := runtime.NumCPU()
	if parallels > containerSize {
		parallels = containerSize
	}
	dataViewOffset := containerSize / parallels
	remainder := containerSize % parallels

	ch := make(chan string, parallels)
	eg := errgroup.Group{}
	nextViewBegin := 0
	for i := 0; i < parallels; i++ {
		viewBegin := nextViewBegin
		viewEnd := nextViewBegin + dataViewOffset
		if remainder != 0 && i < remainder {
			// NOTE: 余剰が出る場合は比較数に偏りが出る可能性がある
			//       goroutine起動index < (containerSize % parallels)なら1要素追加する
			viewEnd += 1
		}

		if viewEnd > containerSize || i == parallels-1 {
			viewEnd = containerSize
		}

		nextViewBegin = viewEnd

		eg.Go(func() error {
			return compSrcImagehash(ch, (*container)[viewBegin:viewEnd], src.ImageHash, threshold)
		})
	}

	go func() {
		eg.Wait()
		close(ch)
	}()

	similarGroups := []string{}
	for filepath := range ch {
		// NOTE: ここでは似ているものだけ受け取るので
		//       一つでも受信したら比較元とグルーピングできる
		similarGroups = append(similarGroups, filepath)
	}

	if len(similarGroups) > 0 {
		similarGroups = append(similarGroups, src.Filepath)
	}

	container.compaction()

	return similarGroups, eg.Wait()
}

func (container *ParallelCompList) compaction() {
	newList := make(ParallelCompList, 0, len(*container))
	for _, info := range *container {
		if info != nil {
			newList = append(newList, info)
		}
	}

	*container = newList
}

func (container *ParallelCompList) Serialize(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed os.Create: %s %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(container); err != nil {
		return fmt.Errorf("failed json.Encode: %w", err)
	}

	return nil
}

func (container *ParallelCompList) Deserialize(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed os.Open: %s %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(container); err != nil {
		return fmt.Errorf("failed json.Decode: %s %w", path, err)
	}

	return nil
}
