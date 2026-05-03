package cram

import (
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

func sentencePinyin(text string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Tone
	args.Heteronym = false

	chunks := make([]string, 0, len(text))
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			parts := pinyin.SinglePinyin(r, args)
			if len(parts) > 0 {
				chunks = append(chunks, parts[0])
			}
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			chunks = append(chunks, string(r))
		}
	}
	return strings.Join(chunks, " ")
}
