package ingest

import "bytes"

func ParseText(raw []byte) ([]byte, []string, error) {
	norm := bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	return norm, nil, nil
}
