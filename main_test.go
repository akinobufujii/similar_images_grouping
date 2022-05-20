package main

import (
	"encoding/json"
	"math/bits"
	"os"
	"reflect"
	"testing"

	"github.com/akinobufujii/similar_images_grouping/readimageutil"
	"github.com/corona10/goimagehash"
)

func TestImageHash(t *testing.T) {
	path := "samples/Cerberus_Front_Pres_01.jpg"
	imageData, imageType, err := readimageutil.ReadImage(path)
	if err != nil {
		t.Fatal(err)
	}

	// NOTE: pHashを計算
	imagehash, err := goimagehash.ExtPerceptionHash(imageData, 16, 16)
	if err != nil {
		t.Fatal(err)
	}

	onesCount := 0
	for _, data := range imagehash.GetHash() {
		onesCount += bits.OnesCount64(data)
	}

	t.Logf("filename: %v\n", path)
	t.Logf("filetype: %v\n", imageType)
	t.Logf("hash: %v\n", imagehash.ToString())
	t.Logf("onesCount: %v\n", onesCount)
}

// TestEncodeDecodeImageHashInfo エンコード・デコードテスト
func TestEncodeDecodeImageHashInfo(t *testing.T) {
	readPathList := []string{
		"samples/Cerberus_Front_Pres_01.jpg",
		"samples/sample1.jpg",
	}

	encodeImageHashInfoList := ImageHashInfoList{}
	for _, path := range readPathList {
		imageData, _, err := readimageutil.ReadImage(path)
		if err != nil {
			t.Fatal(err)
		}

		// NOTE: pHashを計算
		imagehash, err := goimagehash.ExtPerceptionHash(imageData, 16, 16)
		if err != nil {
			t.Fatal(err)
		}
		encodeImageHashInfoList = append(encodeImageHashInfoList, ImageHashInfo{Filepath: path, ImageHash: imagehash})
	}

	// NOTE: 複数ファイルエンコード・デコードテスト
	saveFile := "imagehash_temp.json"
	file, err := os.Create(saveFile)
	if err != nil {
		t.Fatal(err)
	}

	jsonEncoder := json.NewEncoder(file)
	jsonEncoder.SetIndent("", "  ")
	if err := jsonEncoder.Encode(encodeImageHashInfoList); err != nil {
		t.Fatal(err)
	}

	file.Close()

	file, err = os.Open(saveFile)
	if err != nil {
		t.Fatal(err)
	}

	decodeImageHashInfoList := ImageHashInfoList{}
	if err := json.NewDecoder(file).Decode(&decodeImageHashInfoList); err != nil {
		t.Fatal(err)
	}

	// NOTE: 内容一致確認
	if !reflect.DeepEqual(encodeImageHashInfoList, decodeImageHashInfoList) {
		t.Fatal("failed encode/decode imageHashInfo")
	}

	for i := range encodeImageHashInfoList {
		encodeImageInfo := encodeImageHashInfoList[i]
		decodeImageInfo := decodeImageHashInfoList[i]

		t.Logf("%02v encodeImageInfo.filename: %v\n", i, encodeImageInfo.Filepath)
		t.Logf("%02v decodeImageInfo.filename: %v\n", i, decodeImageInfo.Filepath)

		t.Logf("%02v encodeImageInfo.hash: %v\n", i, encodeImageInfo.ImageHash.ToString())
		t.Logf("%02v decodeImageInfo.hash: %v\n", i, decodeImageInfo.ImageHash.ToString())
	}
}
