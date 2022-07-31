package main

import (
	"runtime"
	"sync"

	"github.com/corona10/goimagehash"
	"github.com/pkg/errors"
)

type KeyData struct {
	info *ImageHashInfo
}

type ParallelCompList []*ImageHashInfo

func (container *ParallelCompList) IsEmpty() bool {
	return len(*container) == 0
}

func (container *ParallelCompList) Append(info ImageHashInfo) {
	*container = append(*container, &info)
}

func (container *ParallelCompList) GetKeyData() *KeyData {
	keydata := &ImageHashInfo{}
	for i, data := range *container {
		if data != nil {
			keydata = data
			(*container)[i] = nil
			break
		}
	}

	if keydata != nil {
		return &KeyData{info: keydata}
	}

	return nil
}

func compKeydata(ch chan<- string, view ParallelCompList, imageHash *goimagehash.ExtImageHash, threshold int) error {
	for i, data := range view {
		if data == nil {
			continue
		}

		distance, err := data.ImageHash.Distance(imageHash)
		if err != nil {
			return errors.Wrap(err, "failed ImageHash.Distance")
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

func (container *ParallelCompList) GroupingSimilarImage(keydata *KeyData, threshold int) ([]string, error) {
	// NOTE: 論理スレッド数分goroutineを生成し
	//       その中でkeydataの内容と近いかどうかを総当たりで全比較する
	similarGroups := []string{}
	parallels := runtime.NumCPU()
	containerSize := len(*container)
	if parallels > containerSize {
		parallels = containerSize
	}
	dataViewOffset := containerSize / parallels

	ch := make(chan string, parallels)
	wg := sync.WaitGroup{}
	wg.Add(parallels)
	for i := 0; i < parallels; i++ {
		viewBegin := i * dataViewOffset
		viewEnd := (i + 1) * dataViewOffset
		if viewEnd > containerSize {
			viewEnd = containerSize
		}

		go func(view ParallelCompList, imageHash *goimagehash.ExtImageHash, threshold int) {
			compKeydata(ch, view, imageHash, threshold)
			wg.Done()
		}((*container)[viewBegin:viewEnd], keydata.info.ImageHash, threshold)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for filepath := range ch {
		// NOTE: ここでは似ているものだけ受け取るので
		//       一つでも受信したらkeydataとグルーピングできる
		similarGroups = append(similarGroups, filepath)
	}

	if len(similarGroups) > 0 {
		similarGroups = append(similarGroups, keydata.info.Filepath)
	}

	return similarGroups, nil
}

func (container *ParallelCompList) Compaction() {
	newList := make(ParallelCompList, 0, len(*container))
	for _, info := range *container {
		if info != nil {
			newList = append(newList, info)
		}
	}

	*container = newList
}
