package ingest

import (
	"archive/zip"
	"bytes"
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

func makePDF() []byte {
	return []byte(`%PDF-1.1
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 300 300] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj
4 0 obj << /Length 70 >> stream
BT
/F1 12 Tf
40 250 Td
(Phase PDF text) Tj
ET
endstream endobj
5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj
xref
0 6
0000000000 65535 f
0000000010 00000 n
0000000063 00000 n
0000000122 00000 n
0000000270 00000 n
0000000414 00000 n
trailer << /Root 1 0 R /Size 6 >>
startxref
492
%%EOF`)
}

func makeDOCX() []byte {
	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	w, _ := zw.Create("word/document.xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Docx Heading</w:t></w:r></w:p><w:p><w:r><w:t>Paragraph</w:t></w:r></w:p></w:body></w:document>`))
	_ = zw.Close()
	return buf.Bytes()
}

func TestDetectType(t *testing.T) {
	pdf := makePDF()
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
	pdfOut, _, err := ParsePDF(makePDF())
	if err != nil {
		t.Fatalf("ParsePDF: %v", err)
	}
	if !strings.Contains(strings.ToLower(string(pdfOut)), "pdf") {
		t.Fatal("pdf extraction missing expected content")
	}

	docxOut, _, err := ParseDOCX(makeDOCX())
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
