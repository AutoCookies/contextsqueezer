package ingest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

const defaultMaxBytes = 50 * 1024 * 1024

type Result struct {
	Text       []byte
	SourceType string
	Warnings   []string
}

func maxBytes() int64 {
	v := os.Getenv("CSQ_MAX_BYTES")
	if v == "" {
		return defaultMaxBytes
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return defaultMaxBytes
	}
	return n
}

func ReadFileLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	limit := maxBytes()
	lr := &io.LimitedReader{R: f, N: limit + 1}
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("input exceeds max bytes limit (%d); set CSQ_MAX_BYTES to override", limit)
	}
	return b, nil
}

func Run(ctx context.Context, path string, sourceOverride string) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
	}
	raw, err := ReadFileLimited(path)
	if err != nil {
		return Result{}, err
	}
	kind, err := DetectType(path, raw, sourceOverride)
	if err != nil {
		return Result{}, err
	}
	var text []byte
	var warnings []string
	switch kind {
	case "pdf":
		text, warnings, err = ParsePDF(raw)
	case "docx":
		text, warnings, err = ParseDOCX(raw)
	case "html":
		text, warnings, err = ParseHTML(raw)
	case "text":
		text, warnings, err = ParseText(raw)
	default:
		err = errors.New("unsupported source type")
	}
	if err != nil {
		return Result{}, err
	}
	return Result{Text: text, SourceType: kind, Warnings: warnings}, nil
}
