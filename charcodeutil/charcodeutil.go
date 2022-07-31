package charcodeutil

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// SjisToUTF8 ShiftJIS → UTF-8
func SjisToUTF8(str string) (string, error) {
	ret, err := io.ReadAll(transform.NewReader(strings.NewReader(str), japanese.ShiftJIS.NewDecoder()))
	if err != nil {
		return "", fmt.Errorf("failed to io.ReadAll: %w", err)
	}
	return string(ret), nil
}
