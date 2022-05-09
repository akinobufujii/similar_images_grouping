package readimageutil

import "testing"

func TestReadImage(t *testing.T) {
	_, imageType, err := ReadImage("../samples/Cerberus_Front_Pres_01.jpg")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("imageType: %v\n", imageType)
}
