package readimageutil

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
)

// ReadImage 画像データ読み込み
func ReadImage(path string) (image.Image, string, error) {
	file, err := os.Open(path)
	if err != nil {
		var empty image.Image
		return empty, "", fmt.Errorf("failed os.Open: %s %w", path, err)
	}
	defer file.Close()

	imageData, imageType, err := DecodeImage(file)
	if err != nil {
		return imageData, imageType, fmt.Errorf("failed DecodeImage: %s %w", path, err)
	}

	return imageData, imageType, nil
}

// DecodeImage 画像データデコード
func DecodeImage(reader io.Reader) (image.Image, string, error) {
	imageData, imageType, err := image.Decode(reader)
	if err != nil {
		return imageData, imageType, fmt.Errorf("failed image.Decode: %w", err)
	}

	return imageData, imageType, nil
}
