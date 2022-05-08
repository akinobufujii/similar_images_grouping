package readimageutil

import (
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ReadImage 画像データ読み込み（内部で拡張子による分岐を行う）
func ReadImage(path string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		imageData, err := ReadJpeg(path)
		if err != nil {
			return imageData, errors.Wrap(err, "failed ReadJpeg")
		}
		return imageData, nil

	case ".png":
		imageData, err := ReadPng(path)
		if err != nil {
			return imageData, errors.Wrap(err, "failed ReadPng")
		}
		return imageData, nil
	}

	var imageData image.Image
	return imageData, errors.New("unknown image " + ext)
}

// ReadJpeg jpeg画像読み込み
func ReadJpeg(path string) (image.Image, error) {
	var imageData image.Image
	file, err := os.Open(path)
	if err != nil {
		return imageData, errors.Wrap(err, "failed os.Open")
	}

	imageData, err = jpeg.Decode(file)
	if err != nil {
		return imageData, errors.Wrap(err, "failed jpeg.Decode")
	}

	return imageData, nil
}

// ReadPng png画像読み込み
func ReadPng(path string) (image.Image, error) {
	var imageData image.Image
	file, err := os.Open(path)
	if err != nil {
		return imageData, errors.Wrap(err, "failed os.Open")
	}

	imageData, err = png.Decode(file)
	if err != nil {
		return imageData, errors.Wrap(err, "failed png.Decode")
	}

	return imageData, nil
}
