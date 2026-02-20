package ingest

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
)

var rePDFText = regexp.MustCompile(`\(([^()]*)\)\s*Tj`)

func ParsePDF(raw []byte) ([]byte, []string, error) {
	if !bytes.HasPrefix(raw, []byte("%PDF-")) {
		return nil, nil, errors.New("invalid pdf header")
	}
	matches := rePDFText.FindAllSubmatch(raw, -1)
	var b strings.Builder
	for _, m := range matches {
		if len(m) > 1 {
			b.Write(m[1])
			b.WriteString("\n")
		}
	}
	text := strings.TrimSpace(b.String())
	warnings := []string{}
	if text == "" {
		warnings = append(warnings, "pdf extraction produced very little text")
	}
	if len(raw) > 0 && float64(len(text))/float64(len(raw)) < 0.01 {
		warnings = append(warnings, "pdf text ratio is very low")
	}
	return []byte(text), warnings, nil
}
