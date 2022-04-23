package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image/jpeg"
	"os"

	"github.com/corona10/goimagehash"
)

func main() {
	// TODO: ディレクトリ内検査
	file, err := os.Open("samples/Cerberus_Front_Pres_01.jpg")
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	// TODO: 拡張子による分岐
	image, err := jpeg.Decode(file)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	// TODO: サンプル解像度を引数にする
	// TODO: 関数化
	imagehash, err := goimagehash.ExtPerceptionHash(image, 16, 16)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	// TODO: 比較アルゴリズム
	b := bytes.Buffer{}
	writer := bufio.NewWriter(&b)
	if err := imagehash.Dump(writer); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	writer.Flush()
	fmt.Println(b)
}
