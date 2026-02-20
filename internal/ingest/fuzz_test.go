package ingest

import (
	"archive/zip"
	"bytes"
	"testing"
)

func FuzzDetectType(f *testing.F) {
	f.Add("a.txt", []byte("hello"))
	f.Add("a.html", []byte("<html><body>x</body></html>"))
	f.Fuzz(func(t *testing.T, name string, data []byte) {
		_, _ = DetectType(name, data, "auto")
	})
}

func FuzzParseHTML(f *testing.F) {
	f.Add([]byte("<html><script>x</script><h1>A</h1><p>B</p></html>"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseHTML(data)
	})
}

func FuzzParseDOCXXMLFallback(f *testing.F) {
	f.Add([]byte("<?xml version=\"1.0\"?><w:document xmlns:w=\"x\"><w:body><w:p><w:r><w:t>a</w:t></w:r></w:p></w:body></w:document>"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseDOCX(data)
		buf := bytes.NewBuffer(nil)
		zw := zip.NewWriter(buf)
		w, _ := zw.Create("word/document.xml")
		_, _ = w.Write(data)
		_ = zw.Close()
		_, _, _ = ParseDOCX(buf.Bytes())
	})
}
