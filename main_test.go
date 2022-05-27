package main

import (
	"encoding/json"
	"fmt"
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

func TestJsonFormat(t *testing.T) {
	data, err := os.ReadFile("midfile.json")
	if err != nil {
		t.Fatal(err)
	}

	var encodeList any
	if err := json.Unmarshal(data, &encodeList); err != nil {
		t.Fatal(err)
	}

	if err := writeJson("midfile-format.json", encodeList); err != nil {
		t.Fatal(err)
	}
}

// TestOnebitCount ビットが立っている数の比較テスト
func TestOnebitCount(t *testing.T) {
	data, err := os.ReadFile("midfile.json")
	if err != nil {
		t.Fatal(err)
	}

	encodeList := ImageHashInfoList{}
	if err := json.Unmarshal(data, &encodeList); err != nil {
		t.Fatal(err)
	}

	onesBitCountSumMap := map[string]int32{}
	onesBitCountShiftMap := map[string]int32{}
	onesBitCountShiftSumMap := map[string]int32{}
	t.Logf("listnum: %v\n", len(encodeList))
	for _, encodeData := range encodeList {
		onesbitcount := uint32(0)
		onesbitshift := uint32(0)
		onesbitshiftsum := uint32(0)
		thresholdShift := len(encodeData.ImageHash.GetHash()) / 2
		for i, bit64 := range encodeData.ImageHash.GetHash() {
			ones := uint32(bits.OnesCount64(bit64))

			// 単純にビット立ってる数足すだけ
			onesbitcount += ones

			// 64bitごとにビット立ってる数を計算してシフト
			onesbitshift <<= 8
			onesbitshift |= ones

			// hashのビット数を半分に割ってシフト
			// 例：256bitなら128bitのビットを数えてシフト
			if i == thresholdShift {
				onesbitshiftsum <<= 16
			}
			onesbitshiftsum += ones
		}
		onesBitCountSumMap[fmt.Sprintf("%v", onesbitcount)]++
		onesBitCountShiftMap[fmt.Sprintf("%v + %v + %v + %v",
			(onesbitshift>>24)&0x000000ff,
			(onesbitshift>>16)&0x000000ff,
			(onesbitshift>>8)&0x000000ff,
			(onesbitshift)&0x000000ff)]++
		onesBitCountShiftSumMap[fmt.Sprintf("%v + %v",
			(onesbitshiftsum>>16)&0x0000ffff,
			(onesbitshiftsum)&0x0000ffff)]++
	}

	if err := writeJson("onesbitsum.json", onesBitCountSumMap); err != nil {
		t.Fatal(err)
	}
	if err := writeJson("onesbitshift.json", onesBitCountShiftMap); err != nil {
		t.Fatal(err)
	}
	if err := writeJson("onesbitshiftsum.json", onesBitCountShiftSumMap); err != nil {
		t.Fatal(err)
	}
}
