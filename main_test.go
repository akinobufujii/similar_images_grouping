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
	path := "samples/Cerberus_Front_Pres_01.jpg"
	imageData, _, err := readimageutil.ReadImage(path)
	if err != nil {
		t.Fatal(err)
	}

	// NOTE: pHashを計算
	imagehash, err := goimagehash.ExtPerceptionHash(imageData, 16, 16)
	if err != nil {
		t.Fatal(err)
	}
	// NOTE: エンコード・デコード
	imageHashInfoA := ImageHashInfo{Filepath: path, ImageHash: imagehash}
	imageHashInfoB := ImageHashInfo{}

	t.Logf("imageHashInfoA.filename: %v\n", imageHashInfoA.Filepath)
	t.Logf("imageHashInfoA.hash: %v\n", imageHashInfoA.ImageHash.ToString())

	saveFile := "imagehash_temp.json"
	file, err := os.Create(saveFile)
	if err != nil {
		t.Fatal(err)
	}

	jsonEncoder := json.NewEncoder(file)
	jsonEncoder.SetIndent("", "  ")
	if err := imageHashInfoA.EncodeToJson(jsonEncoder); err != nil {
		t.Fatal(err)
	}

	file.Close()

	file, err = os.Open(saveFile)
	if err != nil {
		t.Fatal(err)
	}

	jsonDecoder := json.NewDecoder(file)
	if err := imageHashInfoB.DecodeFromJson(jsonDecoder); err != nil {
		t.Fatal(err)
	}

	t.Logf("imageHashInfoB.filename: %v\n", imageHashInfoB.Filepath)
	t.Logf("imageHashInfoB.hash: %v\n", imageHashInfoB.ImageHash.ToString())

	if !reflect.DeepEqual(imageHashInfoA, imageHashInfoB) {
		t.Fatal("failed encode/decode imageHashInfo")
	}
}

func TestWriteJson(t *testing.T) {
	data := []ImageHashInfo{
		{
			Filepath: "hoge",
		},
	}
	writeJson("test.json", data)
}
