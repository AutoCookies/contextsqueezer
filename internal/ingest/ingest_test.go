package ingest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("fixtures", name)
}

func TestDetectType(t *testing.T) {
	pdf, _ := os.ReadFile(fixture(t, "sample.pdf"))
	if got, _ := DetectType("a.pdf", pdf, "auto"); got != "pdf" {
		t.Fatalf("detect pdf failed: %s", got)
	}
	html, _ := os.ReadFile(fixture(t, "sample.html"))
	if got, _ := DetectType("a.unknown", html, "auto"); got != "html" {
		t.Fatalf("detect html failed: %s", got)
	}
	if _, err := DetectType("x.bin", []byte{0, 0, 0, 1, 2}, "auto"); err == nil {
		t.Fatal("expected binary detection error")
	}
}

func TestParseHTML(t *testing.T) {
	raw, _ := os.ReadFile(fixture(t, "sample.html"))
	out, _, err := ParseHTML(raw)
	if err != nil {
		t.Fatalf("ParseHTML: %v", err)
	}
	s := string(out)
	if strings.Contains(s, "ignore me") {
		t.Fatal("script content must be removed")
	}
	if !strings.Contains(s, "# Main Heading") {
		t.Fatalf("expected heading marker, got: %s", s)
	}
}

func TestParsersExtract(t *testing.T) {
	pdfRaw, _ := os.ReadFile(fixture(t, "sample.pdf"))
	pdfOut, _, err := ParsePDF(pdfRaw)
	if err != nil {
		t.Fatalf("ParsePDF: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(pdfOut)), "pdf") {
		t.Fatal("pdf extraction missing expected content")
	}

	docxRaw, _ := os.ReadFile(fixture(t, "sample.docx"))
	docxOut, _, err := ParseDOCX(docxRaw)
	if err != nil {
		t.Fatalf("ParseDOCX: %v", err)
	}
	if !strings.Contains(string(docxOut), "Docx Heading") {
		t.Fatal("docx extraction missing heading")
	}

	textRaw, _ := os.ReadFile(fixture(t, "sample.txt"))
	textOut, _, err := ParseText(textRaw)
	if err != nil {
		t.Fatalf("ParseText: %v", err)
	}
	if len(textOut) == 0 {
		t.Fatal("text extraction empty")
	}
}

func TestRunIngest(t *testing.T) {
	res, err := Run(context.Background(), fixture(t, "sample.txt"), "auto")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.SourceType != "text" {
		t.Fatalf("source type mismatch: %s", res.SourceType)
	}
}
