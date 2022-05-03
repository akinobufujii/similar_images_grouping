package main

import (
	"math/bits"
	"testing"

	"github.com/akinobufujii/similar_images_grouping/readimageutil"
	"github.com/corona10/goimagehash"
)

func TestImageHash(t *testing.T) {
	path := "samples/Cerberus_Front_Pres_01.jpg"
	imageData, err := readimageutil.ReadImage(path)
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
	t.Logf("hash: %v\n", imagehash.ToString())
	t.Logf("onesCount: %v\n", onesCount)
}
