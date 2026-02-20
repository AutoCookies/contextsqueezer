package pipeline

import (
	"math"
	"sort"
	"strings"
)

type Stats struct {
	BytesIn         int
	BytesOut        int
	ApproxTokensIn  int
	ApproxTokensOut int
	WordsIn         int
	WordsOut        int
	ReductionPct    float64
	RuntimeMS       int64
}

type QualityReport struct {
	AnchorRetentionOK bool
	SectionCoverageOK bool
	KeywordRecallOK   bool
	KeywordRecall     float64
}

func ApproxTokens(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	return int(math.Ceil(float64(len(b)) / 4.0))
}

func WordCount(b []byte) int {
	return len(strings.Fields(string(b)))
}

func AnalyzeQuality(in, out []byte, aggr int) QualityReport {
	anchors := anchorLines(in)
	anchorOK := true
	for _, a := range anchors {
		if !strings.Contains(string(out), a) {
			anchorOK = false
			break
		}
	}

	sectionOK := sectionCoverage(in, out)
	recall := keywordRecall(in, out, 20)
	minRecall := 0.7
	if aggr >= 7 {
		minRecall = 0.6
	}
	return QualityReport{
		AnchorRetentionOK: anchorOK,
		SectionCoverageOK: sectionOK,
		KeywordRecallOK:   recall >= minRecall,
		KeywordRecall:     recall,
	}
}

func anchorLines(b []byte) []string {
	lines := strings.Split(string(b), "\n")
	out := make([]string, 0)
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.Contains(trim, "```") || strings.Contains(trim, "http://") || strings.Contains(trim, "https://") || countDigits(trim) >= 4 || strings.HasPrefix(trim, "#") || isAllCapsHeavy(trim) {
			out = append(out, trim)
		}
	}
	return out
}

func sectionCoverage(in, out []byte) bool {
	inLines := strings.Split(string(in), "\n")
	outText := string(out)
	for i, line := range inLines {
		trim := strings.TrimSpace(line)
		if trim == "" || !(strings.HasPrefix(trim, "#") || isAllCapsHeavy(trim)) {
			continue
		}
		covered := false
		hadCandidate := false
		for j := i + 1; j < len(inLines); j++ {
			next := strings.TrimSpace(inLines[j])
			if next == "" {
				continue
			}
			if strings.HasPrefix(next, "#") || isAllCapsHeavy(next) {
				break
			}
			hadCandidate = true
			if strings.Contains(outText, next) {
				covered = true
				break
			}
		}
		if hadCandidate && !covered {
			return false
		}
	}
	return true
}

func keywordRecall(in, out []byte, topK int) float64 {
	tf, df, n := tfidfInputs(in)
	if n == 0 {
		return 1
	}
	type kv struct {
		token string
		score float64
	}
	items := make([]kv, 0, len(tf))
	for t, f := range tf {
		idf := math.Log(1 + float64(n)/(1+float64(df[t])))
		items = append(items, kv{token: t, score: float64(f) * idf})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].token < items[j].token
		}
		return items[i].score > items[j].score
	})
	if len(items) > topK {
		items = items[:topK]
	}
	if len(items) == 0 {
		return 1
	}
	outSet := tokenSet(out)
	hit := 0
	for _, it := range items {
		if _, ok := outSet[it.token]; ok {
			hit++
		}
	}
	return float64(hit) / float64(len(items))
}

func tfidfInputs(in []byte) (map[string]int, map[string]int, int) {
	lines := strings.Split(string(in), ".")
	tf := map[string]int{}
	df := map[string]int{}
	n := 0
	for _, line := range lines {
		toks := tokens([]byte(line))
		if len(toks) == 0 {
			continue
		}
		n++
		seen := map[string]struct{}{}
		for _, t := range toks {
			tf[t]++
			if _, ok := seen[t]; !ok {
				seen[t] = struct{}{}
				df[t]++
			}
		}
	}
	return tf, df, n
}

func tokenSet(b []byte) map[string]struct{} {
	out := map[string]struct{}{}
	for _, t := range tokens(b) {
		out[t] = struct{}{}
	}
	return out
}

func tokens(b []byte) []string {
	res := make([]string, 0)
	cur := make([]byte, 0)
	flush := func() {
		if len(cur) > 0 {
			res = append(res, string(cur))
			cur = cur[:0]
		}
	}
	for _, c := range b {
		if c >= 'A' && c <= 'Z' {
			cur = append(cur, c-'A'+'a')
		} else if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			cur = append(cur, c)
		} else {
			flush()
		}
	}
	flush()
	return res
}

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

func isAllCapsHeavy(s string) bool {
	letters := 0
	upper := 0
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			letters++
			upper++
		} else if r >= 'a' && r <= 'z' {
			letters++
		}
	}
	return letters >= 4 && float64(upper)/float64(letters) >= 0.8
}
