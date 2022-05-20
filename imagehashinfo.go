package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"

	"github.com/corona10/goimagehash"
	"github.com/pkg/errors"
)

type ImageHashInfo struct {
	Filepath  string
	ImageHash *goimagehash.ExtImageHash
}

// EncodeToJson Jsonデータにエンコード
func (p *ImageHashInfo) EncodeToJson(enc *json.Encoder) error {
	b := bytes.Buffer{}
	writer := bufio.NewWriter(&b)
	if err := p.ImageHash.Dump(writer); err != nil {
		return errors.Wrap(err, "failed ImageHash.Dump")
	}

	if err := writer.Flush(); err != nil {
		return errors.Wrap(err, "failed Flush")
	}

	encodeData := struct {
		Filepath      string
		ImageHashDump string
	}{
		Filepath:      p.Filepath,
		ImageHashDump: base64.StdEncoding.EncodeToString(b.Bytes()),
	}

	if err := enc.Encode(encodeData); err != nil {
		return errors.Wrap(err, "failed Encode")
	}

	return nil
}

// DecodeFromJson Jsonデータにデコード
func (p *ImageHashInfo) DecodeFromJson(dec *json.Decoder) error {
	decodeData := struct {
		Filepath      string
		ImageHashDump string
	}{}

	err := dec.Decode(&decodeData)
	if err != nil {
		return errors.Wrap(err, "failed Decode")
	}

	data, err := base64.StdEncoding.DecodeString(decodeData.ImageHashDump)
	if err != nil {
		return errors.Wrap(err, "failed DecodeString")
	}

	reader := bufio.NewReader(bytes.NewBuffer(data))
	p.ImageHash, err = goimagehash.LoadExtImageHash(reader)
	if err != nil {
		return errors.Wrap(err, "failed LoadExtImageHash")
	}
	p.Filepath = decodeData.Filepath

	return nil
}
