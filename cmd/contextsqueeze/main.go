package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"contextsqueezer/internal/ingest"
	"contextsqueezer/internal/pipeline"
	"contextsqueezer/internal/version"
	"contextsqueezer/pkg/api"
)

const (
	exitSuccess  = 0
	exitUsage    = 2
	exitInput    = 3
	exitParse    = 4
	exitTimeout  = 5
	exitInternal = 6
)

type buildInfo struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	CGO    string `json:"cgo"`
}

type jsonResult struct {
	SchemaVersion   int       `json:"schema_version"`
	EngineVersion   string    `json:"engine_version"`
	Build           buildInfo `json:"build"`
	BytesIn         int       `json:"bytes_in"`
	BytesOut        int       `json:"bytes_out"`
	TokensInApprox  int       `json:"tokens_in_approx"`
	TokensOutApprox int       `json:"tokens_out_approx"`
	ReductionPct    float64   `json:"reduction_pct"`
	Aggressiveness  int       `json:"aggressiveness"`
	Profile         string    `json:"profile"`
	BudgetApplied   bool      `json:"budget_applied"`
	Truncated       bool      `json:"truncated"`
	SourceType      string    `json:"source_type"`
	Warnings        []string  `json:"warnings"`
	Text            string    `json:"text,omitempty"`
	TextB64         string    `json:"text_b64,omitempty"`
}

type benchRun struct {
	Run       int    `json:"run"`
	Duration  int64  `json:"duration_ms"`
	Hash      string `json:"sha256"`
	BytesOut  int    `json:"bytes_out"`
	TokensOut int    `json:"tokens_out_approx"`
}

type benchCase struct {
	File        string     `json:"file"`
	Aggressives []int      `json:"aggressives"`
	Runs        []benchRun `json:"runs"`
	MinMS       int64      `json:"min_ms"`
	MedianMS    int64      `json:"median_ms"`
	P95MS       int64      `json:"p95_ms"`
	Determinism bool       `json:"deterministic"`
}

type benchJSON struct {
	SchemaVersion string      `json:"schema_version"`
	Suite         string      `json:"suite"`
	Runs          int         `json:"runs"`
	Warmup        int         `json:"warmup"`
	Cases         []benchCase `json:"cases"`
}

func cgoEnabled() string {
	if strings.Contains(runtime.Version(), "cgo") {
		return "enabled"
	}
	return "unknown"
}

func printErr(stderr io.Writer, code int, msg string, err error) int {
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", msg, err)
	} else {
		_, _ = fmt.Fprintln(stderr, msg)
	}
	if os.Getenv("CSQ_DEBUG") == "1" {
		_, _ = fmt.Fprintf(stderr, "debug stack:\n%s\n", string(debug.Stack()))
	}
	return code
}

func classifyErr(err error) int {
	if err == nil {
		return exitSuccess
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return exitTimeout
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "unsupported binary") || strings.Contains(s, "input") || strings.Contains(s, "max bytes") {
		return exitInput
	}
	if strings.Contains(s, "parse") || strings.Contains(s, "pdf") || strings.Contains(s, "docx") || strings.Contains(s, "html") {
		return exitParse
	}
	return exitInternal
}

func writeOutput(path string, data []byte, stdout io.Writer) error {
	if path == "" {
		_, err := stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func parseInputArg(fs *flag.FlagSet, inputFlag string) (string, error) {
	if inputFlag != "" {
		return inputFlag, nil
	}
	if fs.NArg() == 1 {
		return fs.Arg(0), nil
	}
	return "", errors.New("input file required")
}

func parseAggrRange(s string) ([]int, error) {
	if s == "" || s == "0..9" {
		arr := make([]int, 10)
		for i := 0; i <= 9; i++ {
			arr[i] = i
		}
		return arr, nil
	}
	if strings.Contains(s, "..") {
		parts := strings.Split(s, "..")
		if len(parts) != 2 {
			return nil, errors.New("invalid aggr range")
		}
		a, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, err
		}
		b, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		if a < 0 {
			a = 0
		}
		if b > 9 {
			b = 9
		}
		if a > b {
			return nil, errors.New("invalid aggr range")
		}
		arr := make([]int, 0, b-a+1)
		for i := a; i <= b; i++ {
			arr = append(arr, i)
		}
		return arr, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	if v < 0 || v > 9 {
		return nil, errors.New("aggr must be 0..9")
	}
	return []int{v}, nil
}

func runSqueeze(args []string, stdout io.Writer, stderr io.Writer, statsMode bool) int {
	fs := flag.NewFlagSet("contextsqueeze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version")
	inPath := fs.String("input", "", "input file")
	outPath := fs.String("out", "", "output file path")
	aggr := fs.Int("aggr", -1, "aggressiveness 0..9")
	profile := fs.String("profile", "", "profile local|api")
	maxTokens := fs.Int("max-tokens", 0, "approx token budget")
	maxMemMB := fs.Int("max-memory-mb", 1024, "soft memory ceiling in MB")
	asJSON := fs.Bool("json", false, "emit json")
	source := fs.String("source", "auto", "source override: auto|pdf|docx|html|text")
	quiet := fs.Bool("quiet", false, "suppress warnings")
	verbose := fs.Bool("verbose", false, "print stage timing")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version.Current())
		return exitSuccess
	}
	path, err := parseInputArg(fs, *inPath)
	if err != nil {
		return printErr(stderr, exitUsage, "usage: contextsqueeze [file] [--input file] [--max-tokens N] [--json] [--out path] [--source auto|pdf|docx|html|text]", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ingStart := time.Now()
	ing, err := ingest.Run(ctx, path, *source)
	ingestMS := time.Since(ingStart).Milliseconds()
	if err != nil {
		return printErr(stderr, classifyErr(err), "ingest error", err)
	}

	res, err := pipeline.RunResultWithConfig(
		ing.Text,
		api.Options{Aggressiveness: *aggr, MaxTokens: *maxTokens, Profile: *profile},
		ing.SourceType,
		ing.Warnings,
		pipeline.RunConfig{MaxMemoryMB: *maxMemMB},
	)
	if err != nil {
		return printErr(stderr, classifyErr(err), "squeeze error", err)
	}
	res.Metrics.IngestMS = ingestMS

	if statsMode {
		_, _ = fmt.Fprintf(stderr, "source: %s\n", res.SourceType)
		_, _ = fmt.Fprintf(stderr, "bytes in/out: %d/%d\n", res.BytesIn, res.BytesOut)
		_, _ = fmt.Fprintf(stderr, "tokens in/out (approx): %d/%d\n", res.TokensInApprox, res.TokensOutApprox)
		_, _ = fmt.Fprintf(stderr, "reduction: %.2f%%\n", res.ReductionPct)
		_, _ = fmt.Fprintf(stderr, "aggressiveness: %d\n", res.Aggressiveness)
		_, _ = fmt.Fprintf(stderr, "budget applied: %v\n", res.BudgetApplied)
		_, _ = fmt.Fprintf(stderr, "truncated: %v\n", res.Truncated)
		if *verbose {
			_, _ = fmt.Fprintf(stderr, "stage ms ingest/segment/tokenize/filter/sim/prune/assembly/registry/budget: %d/%d/%d/%d/%d/%d/%d/%d/%d\n",
				res.Metrics.IngestMS, res.Metrics.SegmentationMS, res.Metrics.TokenizationMS, res.Metrics.CandidateFilterMS,
				res.Metrics.SimilarityMS, res.Metrics.PruneMS, res.Metrics.AssemblyMS, res.Metrics.CrossChunkRegistryMS, res.Metrics.BudgetLoopMS)
		}
		_, _ = fmt.Fprintf(stderr, "counters tokens/sentences/candidates/pairs: %d/%d/%d/%d\n",
			res.Metrics.TokensParsed, res.Metrics.SentencesTotal, res.Metrics.SimilarityCandidates, res.Metrics.SimilarityPairs)
		if !*quiet && len(res.Warnings) > 0 {
			_, _ = fmt.Fprintf(stderr, "warnings: %s\n", strings.Join(res.Warnings, "; "))
		}
		return exitSuccess
	}

	if *asJSON {
		jr := jsonResult{SchemaVersion: 1, EngineVersion: version.Current(), Build: buildInfo{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CGO: cgoEnabled()},
			BytesIn: res.BytesIn, BytesOut: res.BytesOut, TokensInApprox: res.TokensInApprox, TokensOutApprox: res.TokensOutApprox,
			ReductionPct: res.ReductionPct, Aggressiveness: res.Aggressiveness, Profile: res.Profile, BudgetApplied: res.BudgetApplied,
			Truncated: res.Truncated, SourceType: res.SourceType, Warnings: res.Warnings}
		if utf8.Valid(res.Text) {
			jr.Text = string(res.Text)
		} else {
			jr.TextB64 = base64.StdEncoding.EncodeToString(res.Text)
		}
		buf, err := json.MarshalIndent(jr, "", "  ")
		if err != nil {
			return printErr(stderr, exitInternal, "json error", err)
		}
		buf = append(buf, '\n')
		if err := writeOutput(*outPath, buf, stdout); err != nil {
			return printErr(stderr, exitInternal, "write output", err)
		}
		return exitSuccess
	}

	if !*quiet && len(res.Warnings) > 0 {
		_, _ = fmt.Fprintf(stderr, "warnings: %s\n", strings.Join(res.Warnings, "; "))
	}
	if *verbose {
		_, _ = fmt.Fprintf(stderr, "stage ms ingest/segment/tokenize/filter/sim/prune/assembly/registry/budget: %d/%d/%d/%d/%d/%d/%d/%d/%d\n",
			res.Metrics.IngestMS, res.Metrics.SegmentationMS, res.Metrics.TokenizationMS, res.Metrics.CandidateFilterMS,
			res.Metrics.SimilarityMS, res.Metrics.PruneMS, res.Metrics.AssemblyMS, res.Metrics.CrossChunkRegistryMS, res.Metrics.BudgetLoopMS)
	}
	if err := writeOutput(*outPath, res.Text, stdout); err != nil {
		return printErr(stderr, exitInternal, "write output", err)
	}
	return exitSuccess
}

func runProfile(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("contextsqueeze profile", flag.ContinueOnError)
	fs.SetOutput(stderr)
	inPath := fs.String("input", "", "input file")
	maxMemMB := fs.Int("max-memory-mb", 1024, "soft memory ceiling in MB")
	aggr := fs.Int("aggr", -1, "aggressiveness")
	source := fs.String("source", "auto", "source override")
	cpuPath := fs.String("cpu", "", "write cpu pprof file")
	heapPath := fs.String("heap", "", "write heap pprof file")
	seconds := fs.Int("seconds", 10, "loop duration seconds")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	path, err := parseInputArg(fs, *inPath)
	if err != nil {
		return printErr(stderr, exitUsage, "usage: contextsqueeze profile <file> --cpu out/cpu.pprof --heap out/heap.pprof --seconds 10", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ingStart := time.Now()
	ing, err := ingest.Run(ctx, path, *source)
	ingestMS := time.Since(ingStart).Milliseconds()
	if err != nil {
		return printErr(stderr, classifyErr(err), "ingest error", err)
	}

	var cpuFile *os.File
	if *cpuPath != "" {
		cpuFile, err = os.Create(*cpuPath)
		if err != nil {
			return printErr(stderr, exitInput, "cpu profile file error", err)
		}
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			_ = cpuFile.Close()
			return printErr(stderr, exitInternal, "start cpu profile error", err)
		}
	}

	deadline := time.Now().Add(time.Duration(*seconds) * time.Second)
	var last pipeline.Result
	loops := 0
	for time.Now().Before(deadline) {
		res, err := pipeline.RunResultWithConfig(ing.Text, api.Options{Aggressiveness: *aggr}, ing.SourceType, ing.Warnings, pipeline.RunConfig{MaxMemoryMB: *maxMemMB})
		if err != nil {
			if cpuFile != nil {
				pprof.StopCPUProfile()
				_ = cpuFile.Close()
			}
			return printErr(stderr, classifyErr(err), "profile error", err)
		}
		last = res
		loops++
	}

	if cpuFile != nil {
		pprof.StopCPUProfile()
		_ = cpuFile.Close()
	}
	if *heapPath != "" {
		f, err := os.Create(*heapPath)
		if err != nil {
			return printErr(stderr, exitInput, "heap profile file error", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			_ = f.Close()
			return printErr(stderr, exitInternal, "write heap profile error", err)
		}
		_ = f.Close()
	}

	_, _ = fmt.Fprintf(stderr, "loops: %d\n", loops)
	_, _ = fmt.Fprintf(stderr, "total time ms: %d\n", last.Metrics.BudgetLoopMS)
	_, _ = fmt.Fprintf(stderr, "ingest time ms: %d\n", ingestMS)
	_, _ = fmt.Fprintf(stderr, "segmentation time ms: %d\n", last.Metrics.SegmentationMS)
	_, _ = fmt.Fprintf(stderr, "tokenization time ms: %d\n", last.Metrics.TokenizationMS)
	_, _ = fmt.Fprintf(stderr, "candidate filter time ms: %d\n", last.Metrics.CandidateFilterMS)
	_, _ = fmt.Fprintf(stderr, "similarity time ms: %d\n", last.Metrics.SimilarityMS)
	_, _ = fmt.Fprintf(stderr, "prune time ms: %d\n", last.Metrics.PruneMS)
	_, _ = fmt.Fprintf(stderr, "assembly time ms: %d\n", last.Metrics.AssemblyMS)
	_, _ = fmt.Fprintf(stderr, "cross-chunk registry time ms: %d\n", last.Metrics.CrossChunkRegistryMS)
	_, _ = fmt.Fprintf(stderr, "peak memory estimate bytes: %d\n", last.Metrics.PeakMemoryEstimateB)
	if *cpuPath != "" {
		_, _ = fmt.Fprintf(stderr, "view cpu profile: go tool pprof -http=:0 %s\n", *cpuPath)
	}
	if *heapPath != "" {
		_, _ = fmt.Fprintf(stderr, "view heap profile: go tool pprof -http=:0 %s\n", *heapPath)
	}
	return exitSuccess
}

func quantile(sorted []int64, q float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * q)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func resolveBenchPath(p string) string {
	if _, err := os.Stat(p); err == nil {
		return p
	}
	alt := filepath.Join("..", "..", p)
	if _, err := os.Stat(alt); err == nil {
		return alt
	}
	return p
}

func benchFilesFromArgs(suite, fileArg, dirArg, pattern string) ([]string, error) {
	if fileArg != "" {
		return []string{resolveBenchPath(fileArg)}, nil
	}
	if dirArg != "" {
		if pattern == "" {
			pattern = "*.txt"
		}
		return filepath.Glob(filepath.Join(resolveBenchPath(dirArg), pattern))
	}
	if suite == "default" {
		return []string{
			resolveBenchPath("testdata/bench/small.txt"),
			resolveBenchPath("testdata/bench/medium.txt"),
			resolveBenchPath("testdata/bench/large.txt"),
			resolveBenchPath("testdata/bench/boilerplate_heavy.txt"),
			resolveBenchPath("testdata/bench/duplicate_heavy.txt"),
			resolveBenchPath("testdata/bench/mixed.html"),
		}, nil
	}
	return nil, errors.New("unknown suite")
}

func runBench(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("contextsqueeze bench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	suite := fs.String("suite", "default", "suite name")
	runs := fs.Int("runs", 5, "number of measured runs")
	warmup := fs.Int("warmup", 1, "number of warmup runs")
	aggrRange := fs.String("aggr", "0..9", "aggressiveness range")
	maxTokens := fs.Int("max-tokens", 0, "approx token budget")
	jsonOut := fs.Bool("json", false, "emit JSON")
	fileArg := fs.String("file", "", "single file")
	dirArg := fs.String("dir", "", "directory of files")
	pattern := fs.String("pattern", "*.txt", "glob pattern when using --dir")
	maxMemMB := fs.Int("max-memory-mb", 1024, "soft memory ceiling in MB")
	source := fs.String("source", "auto", "source override")
	profile := fs.String("profile", "", "profile")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	aggrs, err := parseAggrRange(*aggrRange)
	if err != nil {
		return printErr(stderr, exitUsage, "invalid --aggr", err)
	}
	files, err := benchFilesFromArgs(*suite, *fileArg, *dirArg, *pattern)
	if err != nil || len(files) == 0 {
		return printErr(stderr, exitUsage, "bench input error", err)
	}

	cases := make([]benchCase, 0)
	for _, file := range files {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		ing, err := ingest.Run(ctx, file, *source)
		cancel()
		if err != nil {
			return printErr(stderr, classifyErr(err), fmt.Sprintf("ingest error for %s", file), err)
		}

		for _, a := range aggrs {
			for i := 0; i < *warmup; i++ {
				_, _ = pipeline.RunResultWithConfig(ing.Text, api.Options{Aggressiveness: a, MaxTokens: *maxTokens, Profile: *profile}, ing.SourceType, ing.Warnings, pipeline.RunConfig{MaxMemoryMB: *maxMemMB})
			}
			runsOut := make([]benchRun, 0, *runs)
			var baseline string
			deterministic := true
			for i := 0; i < *runs; i++ {
				t0 := time.Now()
				res, err := pipeline.RunResultWithConfig(ing.Text, api.Options{Aggressiveness: a, MaxTokens: *maxTokens, Profile: *profile}, ing.SourceType, ing.Warnings, pipeline.RunConfig{MaxMemoryMB: *maxMemMB})
				if err != nil {
					return printErr(stderr, classifyErr(err), fmt.Sprintf("bench run error %s aggr=%d", file, a), err)
				}
				dur := time.Since(t0).Milliseconds()
				hash := sha256.Sum256(res.Text)
				digest := hex.EncodeToString(hash[:])
				if i == 0 {
					baseline = digest
				} else if digest != baseline {
					deterministic = false
				}
				runsOut = append(runsOut, benchRun{Run: i + 1, Duration: dur, Hash: digest, BytesOut: res.BytesOut, TokensOut: res.TokensOutApprox})
			}
			if !deterministic {
				return printErr(stderr, exitInternal, fmt.Sprintf("determinism failed for %s aggr=%d", file, a), nil)
			}
			durs := make([]int64, 0, len(runsOut))
			for _, r := range runsOut {
				durs = append(durs, r.Duration)
			}
			sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
			cases = append(cases, benchCase{File: file, Aggressives: []int{a}, Runs: runsOut, MinMS: durs[0], MedianMS: quantile(durs, 0.5), P95MS: quantile(durs, 0.95), Determinism: true})
		}
	}

	if *jsonOut {
		obj := benchJSON{SchemaVersion: "1", Suite: *suite, Runs: *runs, Warmup: *warmup, Cases: cases}
		buf, _ := json.MarshalIndent(obj, "", "  ")
		buf = append(buf, '\n')
		_, _ = stdout.Write(buf)
		return exitSuccess
	}

	_, _ = fmt.Fprintln(stderr, "| file | aggr | run | ms | bytes out | tokens out | sha256 |")
	_, _ = fmt.Fprintln(stderr, "|---|---:|---:|---:|---:|---:|---|")
	for _, c := range cases {
		a := c.Aggressives[0]
		for _, r := range c.Runs {
			_, _ = fmt.Fprintf(stderr, "| %s | %d | %d | %d | %d | %d | %s |\n", c.File, a, r.Run, r.Duration, r.BytesOut, r.TokensOut, r.Hash)
		}
	}
	_, _ = fmt.Fprintln(stderr, "\n| file | aggr | min ms | median ms | p95 ms | deterministic |")
	_, _ = fmt.Fprintln(stderr, "|---|---:|---:|---:|---:|:---:|")
	for _, c := range cases {
		_, _ = fmt.Fprintf(stderr, "| %s | %d | %d | %d | %d | %v |\n", c.File, c.Aggressives[0], c.MinMS, c.MedianMS, c.P95MS, c.Determinism)
	}
	return exitSuccess
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "bench":
			return runBench(args[1:], stdout, stderr)
		case "stats":
			return runSqueeze(args[1:], stdout, stderr, true)
		case "profile":
			return runProfile(args[1:], stdout, stderr)
		}
	}
	return runSqueeze(args, stdout, stderr, false)
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
