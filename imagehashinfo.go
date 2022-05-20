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

type ImageHashInfoList []ImageHashInfo

// MarshalJSON Jsonデータにエンコード
func (p *ImageHashInfo) MarshalJSON() ([]byte, error) {
	b := bytes.Buffer{}
	writer := bufio.NewWriter(&b)
	if err := p.ImageHash.Dump(writer); err != nil {
		return []byte{}, errors.Wrap(err, "failed ImageHash.Dump")
	}

	if err := writer.Flush(); err != nil {
		return []byte{}, errors.Wrap(err, "failed Flush")
	}

	encodeData := struct {
		Filepath      string
		ImageHashDump string
	}{
		Filepath:      p.Filepath,
		ImageHashDump: base64.StdEncoding.EncodeToString(b.Bytes()),
	}

	data, err := json.Marshal(encodeData)
	if err != nil {
		return []byte{}, errors.Wrap(err, "failed Marshal")
	}

	return data, nil
}

// UnmarshalJSON Jsonデータからデコード
func (p *ImageHashInfo) UnmarshalJSON(b []byte) error {
	decodeData := struct {
		Filepath      string
		ImageHashDump string
	}{}

	err := json.Unmarshal(b, &decodeData)
	if err != nil {
		return errors.Wrap(err, "failed Unmarshal")
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
