package ingest

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
)

func DetectType(path string, data []byte, override string) (string, error) {
	if override != "" && override != "auto" {
		switch override {
		case "pdf", "docx", "html", "text":
			return override, nil
		default:
			return "", errors.New("invalid source override")
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	if len(data) >= 5 && string(data[:5]) == "%PDF-" {
		return "pdf", nil
	}
	if len(data) >= 4 && bytes.Equal(data[:4], []byte{'P', 'K', 0x03, 0x04}) {
		if ext == ".docx" {
			return "docx", nil
		}
	}
	if ext == ".pdf" {
		return "pdf", nil
	}
	if ext == ".docx" {
		return "docx", nil
	}
	if ext == ".html" || ext == ".htm" {
		return "html", nil
	}
	if ext == ".txt" || ext == ".md" {
		return "text", nil
	}

	nullCount := 0
	for _, b := range data {
		if b == 0 {
			nullCount++
		}
	}
	if len(data) > 0 && float64(nullCount)/float64(len(data)) > 0.02 {
		return "", errors.New("unsupported binary file")
	}
	if bytes.Contains(bytes.ToLower(data), []byte("<html")) {
		return "html", nil
	}
	return "text", nil
}
