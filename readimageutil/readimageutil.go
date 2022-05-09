package readimageutil

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/pkg/errors"
)

// ReadImage 画像データ読み込み
func ReadImage(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		var empty image.Image
		return empty, "", errors.Wrap(err, "failed os.Open")
	}

	imageData, imageType, err := DecodeImage(file)
	if err != nil {
		return imageData, imageType, errors.Wrap(err, "failed DecodeImage")
	}

	return imageData, imageType, nil
}

// DecodeImage 画像データデコード
func DecodeImage(reader io.Reader) (image.Image, string, error) {
	imageData, imageType, err := image.Decode(reader)
	if err != nil {
		return imageData, imageType, errors.Wrap(err, "failed image.Decode")
	}

	return imageData, imageType, nil
}
