package main

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func makeDOCX(t *testing.T, path string) {
	t.Helper()
	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	w, _ := zw.Create("word/document.xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>E2E DOCX</w:t></w:r></w:p></w:body></w:document>`))
	_ = zw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makePDF(t *testing.T, path string) {
	t.Helper()
	pdf := `%PDF-1.1
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 300 300] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj
4 0 obj << /Length 70 >> stream
BT
/F1 12 Tf
40 250 Td
(E2E PDF text) Tj
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
%%EOF`
	if err := os.WriteFile(path, []byte(pdf), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestE2EFixtures(t *testing.T) {
	tmp := t.TempDir()
	txtPath := filepath.Join(tmp, "sample.txt")
	htmlPath := filepath.Join(tmp, "sample.html")
	pdfPath := filepath.Join(tmp, "sample.pdf")
	docxPath := filepath.Join(tmp, "sample.docx")
	_ = os.WriteFile(txtPath, []byte("# H\ntext\n"), 0o644)
	_ = os.WriteFile(htmlPath, []byte("<html><body><h1>H</h1><p>x</p></body></html>"), 0o644)
	makePDF(t, pdfPath)
	makeDOCX(t, docxPath)

	fixtures := []string{txtPath, htmlPath, pdfPath, docxPath}
	for _, p := range fixtures {
		t.Run(filepath.Base(p), func(t *testing.T) {
			cmd := exec.Command("go", "run", ".", "stats", "--source", "auto", "--max-tokens", "80", p)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("e2e failed: %v\n%s", err, string(out))
			}
			if !strings.Contains(string(out), "tokens in/out") {
				t.Fatalf("unexpected output: %s", string(out))
			}
		})
	}
}
