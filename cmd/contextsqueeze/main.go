package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"contextsqueezer/internal/pipeline"
	"contextsqueezer/internal/version"
	"contextsqueezer/pkg/api"
)

type stats struct {
	bytesIn   int
	bytesOut  int
	tokensIn  int
	tokensOut int
	wordsIn   int
	wordsOut  int
	runtimeMS int64
}

type quality struct {
	anchorRetentionOK bool
	sectionCoverageOK bool
	keywordRecallOK   bool
	keywordRecall     float64
}

func approxTokens(n int) int {
	if n <= 0 {
		return 0
	}
	return int(math.Ceil(float64(n) / 4.0))
}

func wordCount(b []byte) int { return len(strings.Fields(string(b))) }

func detectAnchors(text string) []string {
	lines := strings.Split(text, "\n")
	anchors := make([]string, 0)
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		digits := 0
		for _, r := range trim {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		if strings.Contains(trim, "```") || strings.Contains(trim, "http://") || strings.Contains(trim, "https://") || digits >= 4 || strings.HasPrefix(trim, "#") || allCapsRatioHigh(trim) {
			anchors = append(anchors, trim)
		}
	}
	return anchors
}

func allCapsRatioHigh(s string) bool {
	words := strings.Fields(s)
	alphaWords := 0
	capsWords := 0
	for _, w := range words {
		alpha := 0
		caps := 0
		for _, r := range w {
			if r >= 'A' && r <= 'Z' {
				alpha++
				caps++
			} else if r >= 'a' && r <= 'z' {
				alpha++
			}
		}
		if alpha > 1 {
			alphaWords++
			if alpha == caps {
				capsWords++
			}
		}
	}
	if alphaWords == 0 {
		return false
	}
	return float64(capsWords)/float64(alphaWords) >= 0.6
}

func tokensForRecall(text string) []string {
	stop := map[string]struct{}{"the": {}, "and": {}, "or": {}, "a": {}, "an": {}, "is": {}, "are": {}, "to": {}, "of": {}, "in": {}, "for": {}, "on": {}, "with": {}, "as": {}, "at": {}, "by": {}, "be": {}, "this": {}, "that": {}, "it": {}, "from": {}, "was": {}, "were": {}, "will": {}, "can": {}, "if": {}}
	parts := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		if _, ok := stop[p]; ok {
			continue
		}
		out = append(out, p)
	}
	return out
}

func evaluateQuality(orig, out string, aggr int) quality {
	anchors := detectAnchors(orig)
	anchorOK := true
	for _, a := range anchors {
		if !strings.Contains(out, a) {
			anchorOK = false
			break
		}
	}

	sectionOK := true
	lines := strings.Split(orig, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || !(strings.HasPrefix(trim, "#") || allCapsRatioHigh(trim)) {
			continue
		}
		next := ""
		for j := i + 1; j < len(lines); j++ {
			n := strings.TrimSpace(lines[j])
			if n != "" {
				next = n
				break
			}
		}
		if next != "" && !strings.Contains(out, next) {
			sectionOK = false
			break
		}
	}

	origToks := tokensForRecall(orig)
	df := map[string]int{}
	tf := map[string]int{}
	for _, t := range origToks {
		tf[t]++
		df[t] = 1
	}
	N := 1.0
	type pair struct {
		tok   string
		score float64
	}
	scores := make([]pair, 0, len(tf))
	for tok, cnt := range tf {
		idf := math.Log(1.0 + N/(1.0+float64(df[tok])))
		scores = append(scores, pair{tok: tok, score: float64(cnt) * idf})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	if len(scores) > 20 {
		scores = scores[:20]
	}
	outSet := map[string]struct{}{}
	for _, t := range tokensForRecall(out) {
		outSet[t] = struct{}{}
	}
	hit := 0
	for _, p := range scores {
		if _, ok := outSet[p.tok]; ok {
			hit++
		}
	}
	recall := 1.0
	if len(scores) > 0 {
		recall = float64(hit) / float64(len(scores))
	}
	limit := 0.7
	if aggr >= 7 {
		limit = 0.6
	}

	return quality{anchorRetentionOK: anchorOK, sectionCoverageOK: sectionOK, keywordRecallOK: recall >= limit, keywordRecall: recall}
}

func executeSqueeze(data []byte, aggr int, profile string) ([]byte, stats, quality, error) {
	start := time.Now()
	out, err := pipeline.Run(data, api.Options{Aggressiveness: aggr, Profile: profile})
	dur := time.Since(start)
	if err != nil {
		return nil, stats{}, quality{}, err
	}
	st := stats{bytesIn: len(data), bytesOut: len(out), tokensIn: approxTokens(len(data)), tokensOut: approxTokens(len(out)), wordsIn: wordCount(data), wordsOut: wordCount(out), runtimeMS: dur.Milliseconds()}
	q := evaluateQuality(string(data), string(out), aggr)
	return out, st, q, nil
}

func reductionPct(in, out int) float64 {
	if in == 0 {
		return 0
	}
	return (1.0 - float64(out)/float64(in)) * 100.0
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "bench" {
		return runBench(args[1:], stdout, stderr)
	}

	fs := flag.NewFlagSet("contextsqueeze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version")
	showStats := fs.Bool("stats", false, "print squeeze stats")
	aggr := fs.Int("aggr", -1, "aggressiveness 0..9")
	profile := fs.String("profile", "", "profile: local|api")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version.Current())
		return 0
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: contextsqueeze [--version] [--stats] [--aggr 0..9] [--profile local|api] <input-file>")
		return 2
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	out, st, _, err := executeSqueeze(data, *aggr, *profile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "squeeze: %v\n", err)
		return 1
	}
	if *showStats {
		_, _ = fmt.Fprintf(stdout, "bytes in/out: %d/%d\n", st.bytesIn, st.bytesOut)
		_, _ = fmt.Fprintf(stdout, "approx tokens in/out: %d/%d\n", st.tokensIn, st.tokensOut)
		_, _ = fmt.Fprintf(stdout, "word count in/out: %d/%d\n", st.wordsIn, st.wordsOut)
		_, _ = fmt.Fprintf(stdout, "reduction: %.2f%%\n", reductionPct(st.bytesIn, st.bytesOut))
		_, _ = fmt.Fprintf(stdout, "runtime ms: %d\n", st.runtimeMS)
		return 0
	}
	if _, err := stdout.Write(out); err != nil {
		if !errors.Is(err, io.EOF) {
			_, _ = fmt.Fprintf(stderr, "write output: %v\n", err)
		}
		return 1
	}
	return 0
}

func runBench(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("contextsqueeze bench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	profile := fs.String("profile", "", "profile: local|api")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: contextsqueeze bench [--profile local|api] <input-file>")
		return 2
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	type row struct {
		aggr int
		st   stats
		q    quality
	}
	rows := make([]row, 0, 10)
	for a := 0; a <= 9; a++ {
		_, st, q, err := executeSqueeze(data, a, *profile)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "bench aggr=%d failed: %v\n", a, err)
			return 1
		}
		rows = append(rows, row{aggr: a, st: st, q: q})
	}
	_, _ = fmt.Fprintln(stdout, "| aggr | bytes out | reduction % | approx tokens out | runtime ms | anchors | sections | keyword recall |")
	_, _ = fmt.Fprintln(stdout, "|---:|---:|---:|---:|---:|:---:|:---:|---:|")
	for _, r := range rows {
		_, _ = fmt.Fprintf(stdout, "| %d | %d | %.2f | %d | %d | %v | %v | %.2f |\n", r.aggr, r.st.bytesOut, reductionPct(r.st.bytesIn, r.st.bytesOut), r.st.tokensOut, r.st.runtimeMS, r.q.anchorRetentionOK, r.q.sectionCoverageOK, r.q.keywordRecall)
	}

	best := -1
	bestRed := -1.0
	for _, r := range rows {
		if r.q.anchorRetentionOK && r.q.sectionCoverageOK && r.q.keywordRecallOK {
			red := reductionPct(r.st.bytesIn, r.st.bytesOut)
			if red > bestRed {
				bestRed = red
				best = r.aggr
			}
		}
	}
	if best < 0 {
		_, _ = fmt.Fprintln(stderr, "warning: no aggressiveness level passed quality gates")
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "recommended aggressiveness: %d\n", best)
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
