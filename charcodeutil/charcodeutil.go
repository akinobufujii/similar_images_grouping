package charcodeutil

import (
	"io"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// SjisToUTF8 ShiftJIS → UTF-8
func SjisToUTF8(str string) (string, error) {
	ret, err := io.ReadAll(transform.NewReader(strings.NewReader(str), japanese.ShiftJIS.NewDecoder()))
	if err != nil {
		return "", errors.Wrap(err, "failed to io.ReadAll")
	}
	return string(ret), nil
}
