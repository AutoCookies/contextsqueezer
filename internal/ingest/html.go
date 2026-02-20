package ingest

import (
	"bytes"
	"regexp"
	"strings"
)

var (
	reScript = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reTag    = regexp.MustCompile(`(?is)<[^>]+>`)
	reH      = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
)

func ParseHTML(raw []byte) ([]byte, []string, error) {
	s := string(raw)
	s = reScript.ReplaceAllString(s, " ")
	s = reStyle.ReplaceAllString(s, " ")
	s = reH.ReplaceAllStringFunc(s, func(m string) string {
		sub := reH.FindStringSubmatch(m)
		if len(sub) != 3 {
			return m
		}
		lvl := sub[1]
		t := strings.TrimSpace(reTag.ReplaceAllString(sub[2], " "))
		return strings.Repeat("#", int(lvl[0]-'0')) + " " + t + "\n\n"
	})
	s = reTag.ReplaceAllString(s, " ")
	s = collapseWhitespace(strings.ReplaceAll(s, "\r\n", "\n"))
	warnings := []string{}
	if len(strings.TrimSpace(s)) < 20 {
		warnings = append(warnings, "html extraction produced very little visible text")
	}
	return bytes.TrimSpace([]byte(s)), warnings, nil
}

func collapseWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	empty := false
	for _, l := range lines {
		t := strings.Join(strings.Fields(l), " ")
		if t == "" {
			if !empty {
				out = append(out, "")
				empty = true
			}
			continue
		}
		empty = false
		out = append(out, t)
	}
	return strings.TrimSpace(strings.Join(out, "\n")) + "\n"
}
