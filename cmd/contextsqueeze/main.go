package main

import (
	"bytes"
	"contextsqueezer/internal/pipeline"
	"contextsqueezer/pkg/api"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
	"unicode"
)

type stats struct {
	InBytes    int
	OutBytes   int
	InApprox   int
	OutApprox  int
	InWords    int
	OutWords   int
	RuntimeMS  int64
	Reduction  float64
	Output     []byte
	Aggressive int
}

type quality struct {
	AnchorRetention bool
	SectionCoverage bool
	KeywordRecall   float64
	KeywordPass     bool
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "bench" {
		return runBench(os.Args[2:])
	}

	fs := flag.NewFlagSet("contextsqueeze", flag.ContinueOnError)
	version := fs.Bool("version", false, "print version")
	statsMode := fs.Bool("stats", false, "print stats for input file")
	aggr := fs.Int("aggr", 0, "aggressiveness 0..9")
	profile := fs.String("profile", "", "profile: local|api")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if *version {
		fmt.Println(api.Version())
		return nil
	}

	args := fs.Args()
	if len(args) != 1 {
		return errors.New("usage: contextsqueeze <input-file> [--aggr 0..9 --profile local|api] | --stats <input-file> | --version | bench <input-file>")
	}

	in, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}

	opt := api.Options{Aggressiveness: *aggr, Profile: *profile}
	st, err := squeezeWithStats(in, opt)
	if err != nil {
		return err
	}

	if *statsMode {
		printStats(st)
		return nil
	}

	_, err = os.Stdout.Write(st.Output)
	return err
}

func runBench(args []string) error {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	profile := fs.String("profile", "", "profile: local|api")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return errors.New("usage: contextsqueeze bench <input-file> [--profile local|api]")
	}
	in, err := os.ReadFile(fs.Args()[0])
	if err != nil {
		return err
	}

	fmt.Println("| aggr | in_bytes | out_bytes | in_tok~ | out_tok~ | reduction% | runtime_ms |")
	fmt.Println("|---:|---:|---:|---:|---:|---:|---:|")
	var all []stats
	bestIdx := -1
	bestReduction := -1.0
	for a := 0; a <= 9; a++ {
		st, err := squeezeWithStats(in, api.Options{Aggressiveness: a, Profile: *profile})
		if err != nil {
			return err
		}
		st.Aggressive = a
		all = append(all, st)
		fmt.Printf("| %d | %d | %d | %d | %d | %.2f | %d |\n", a, st.InBytes, st.OutBytes, st.InApprox, st.OutApprox, st.Reduction, st.RuntimeMS)
		q := qualityGates(in, st.Output, a)
		if q.AnchorRetention && st.Reduction > bestReduction {
			bestReduction = st.Reduction
			bestIdx = len(all) - 1
		}
	}
	if bestIdx < 0 {
		bestIdx = 0
	}
	rec := all[bestIdx]
	q := qualityGates(in, rec.Output, rec.Aggressive)
	fmt.Printf("recommended_aggr=%d reduction=%.2f%% anchor_retention=%v section_coverage=%v keyword_recall=%.2f\n", rec.Aggressive, rec.Reduction, q.AnchorRetention, q.SectionCoverage, q.KeywordRecall)
	if !(q.AnchorRetention && q.SectionCoverage && q.KeywordPass) {
		return errors.New("quality gates failed for recommended aggressiveness")
	}
	return nil
}

func squeezeWithStats(in []byte, opt api.Options) (stats, error) {
	start := time.Now()
	out, err := pipeline.Run(in, opt)
	if err != nil {
		return stats{}, err
	}
	elapsed := time.Since(start)
	st := stats{
		InBytes:   len(in),
		OutBytes:  len(out),
		InApprox:  approxTokens(len(in)),
		OutApprox: approxTokens(len(out)),
		InWords:   wordCount(in),
		OutWords:  wordCount(out),
		RuntimeMS: elapsed.Milliseconds(),
		Output:    out,
	}
	if len(in) > 0 {
		st.Reduction = (1.0 - (float64(len(out)) / float64(len(in)))) * 100.0
	}
	return st, nil
}

func printStats(s stats) {
	fmt.Printf("bytes in/out: %d/%d\n", s.InBytes, s.OutBytes)
	fmt.Printf("approx tokens in/out: %d/%d\n", s.InApprox, s.OutApprox)
	fmt.Printf("word count in/out: %d/%d\n", s.InWords, s.OutWords)
	fmt.Printf("reduction: %.2f%%\n", s.Reduction)
	fmt.Printf("runtime ms: %d\n", s.RuntimeMS)
}

func approxTokens(n int) int {
	if n == 0 {
		return 0
	}
	return int(math.Ceil(float64(n) / 4.0))
}

func wordCount(b []byte) int {
	return len(strings.Fields(string(b)))
}

func qualityGates(in, out []byte, aggr int) quality {
	anchors := detectAnchors(in)
	anchorRetention := true
	for _, a := range anchors {
		if !bytes.Contains(out, []byte(a)) {
			anchorRetention = false
			break
		}
	}
	sectionCoverage := true
	for _, heading := range detectHeadings(in) {
		idx := bytes.Index(out, []byte(heading))
		if idx == -1 {
			sectionCoverage = false
			break
		}
		rest := out[idx+len(heading):]
		if len(bytes.TrimSpace(rest)) == 0 {
			sectionCoverage = false
			break
		}
	}
	top := topTokens(in, 20)
	have := tokenSet(out)
	hit := 0
	for _, t := range top {
		if _, ok := have[t]; ok {
			hit++
		}
	}
	recall := 1.0
	if len(top) > 0 {
		recall = float64(hit) / float64(len(top))
	}
	threshold := 0.7
	if aggr >= 7 {
		threshold = 0.6
	}
	return quality{
		AnchorRetention: anchorRetention,
		SectionCoverage: sectionCoverage,
		KeywordRecall:   recall,
		KeywordPass:     recall >= threshold,
	}
}

func detectAnchors(b []byte) []string {
	lines := strings.Split(string(b), "\n")
	var out []string
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if trim == "" {
			continue
		}
		if strings.Contains(trim, "```") || strings.Contains(trim, "http://") || strings.Contains(trim, "https://") {
			out = append(out, trim)
			continue
		}
		digits := 0
		letters := 0
		upper := 0
		for _, r := range trim {
			if unicode.IsDigit(r) {
				digits++
			}
			if unicode.IsLetter(r) {
				letters++
				if unicode.IsUpper(r) {
					upper++
				}
			}
		}
		if digits >= 4 || strings.HasPrefix(trim, "#") || (letters > 0 && float64(upper)/float64(letters) >= 0.7) {
			out = append(out, trim)
		}
	}
	return out
}

func detectHeadings(b []byte) []string {
	lines := strings.Split(string(b), "\n")
	var h []string
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(trim, "#") {
			h = append(h, trim)
		}
	}
	return h
}

func topTokens(b []byte, n int) []string {
	parts := strings.FieldsFunc(strings.ToLower(string(b)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	stop := map[string]struct{}{"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "to": {}, "of": {}, "in": {}, "on": {}, "for": {}, "with": {}, "is": {}, "are": {}, "was": {}, "were": {}}
	freq := map[string]int{}
	for _, p := range parts {
		if _, ok := stop[p]; ok || p == "" {
			continue
		}
		freq[p]++
	}
	type kv struct {
		K string
		V int
	}
	var all []kv
	for k, v := range freq {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].V != all[j].V {
			return all[i].V > all[j].V
		}
		return all[i].K < all[j].K
	})
	if len(all) > n {
		all = all[:n]
	}
	out := make([]string, 0, len(all))
	for _, it := range all {
		out = append(out, it.K)
	}
	return out
}

func tokenSet(b []byte) map[string]struct{} {
	s := map[string]struct{}{}
	parts := strings.FieldsFunc(strings.ToLower(string(b)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	for _, p := range parts {
		if p != "" {
			s[p] = struct{}{}
		}
	}
	return s
}
