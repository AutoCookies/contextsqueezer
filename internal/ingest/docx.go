package ingest

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

type xmlNode struct {
	XMLName xml.Name
	Content string    `xml:",chardata"`
	Nodes   []xmlNode `xml:",any"`
}

func ParseDOCX(raw []byte) ([]byte, []string, error) {
	warnings := []string{}
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		if bytes.Contains(raw, []byte("<w:document")) || bytes.Contains(raw, []byte("<document")) {
			text := extractDocxXMLText(raw)
			warnings = append(warnings, "docx parsed from xml fallback")
			return []byte(text), warnings, nil
		}
		return nil, warnings, err
	}
	var doc []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, openErr := f.Open()
			if openErr != nil {
				return nil, warnings, openErr
			}
			doc, err = io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, warnings, err
			}
			break
		}
	}
	if len(doc) == 0 {
		warnings = append(warnings, "docx missing word/document.xml")
		return nil, warnings, errors.New("docx missing document.xml")
	}
	text := extractDocxXMLText(doc)
	return []byte(text), warnings, nil
}

func extractDocxXMLText(doc []byte) string {
	var root xmlNode
	if err := xml.Unmarshal(doc, &root); err != nil {
		return ""
	}
	paras := make([]string, 0)
	var walk func(xmlNode)
	walk = func(n xmlNode) {
		if strings.ToLower(n.XMLName.Local) == "p" {
			var sb strings.Builder
			var gather func(xmlNode)
			gather = func(x xmlNode) {
				if strings.ToLower(x.XMLName.Local) == "t" {
					t := strings.TrimSpace(x.Content)
					if t != "" {
						sb.WriteString(t)
						sb.WriteByte(' ')
					}
				}
				for _, c := range x.Nodes {
					gather(c)
				}
			}
			gather(n)
			p := strings.TrimSpace(strings.Join(strings.Fields(sb.String()), " "))
			if p != "" {
				paras = append(paras, p)
			}
		}
		for _, c := range n.Nodes {
			walk(c)
		}
	}
	walk(root)
	if len(paras) == 0 {
		return ""
	}
	return strings.Join(paras, "\n\n") + "\n"
}
