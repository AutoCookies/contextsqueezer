package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"contextsqueezer/internal/ingest"
	"contextsqueezer/internal/pipeline"
	"contextsqueezer/internal/version"
	"contextsqueezer/pkg/api"
)

type jsonResult struct {
	BytesIn         int      `json:"bytes_in"`
	BytesOut        int      `json:"bytes_out"`
	TokensInApprox  int      `json:"tokens_in_approx"`
	TokensOutApprox int      `json:"tokens_out_approx"`
	ReductionPct    float64  `json:"reduction_pct"`
	Aggressiveness  int      `json:"aggressiveness"`
	Profile         string   `json:"profile"`
	BudgetApplied   bool     `json:"budget_applied"`
	Truncated       bool     `json:"truncated"`
	SourceType      string   `json:"source_type"`
	Warnings        []string `json:"warnings"`
	Text            string   `json:"text,omitempty"`
	TextB64         string   `json:"text_b64,omitempty"`
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

func runSqueeze(args []string, stdout io.Writer, stderr io.Writer, statsMode bool) int {
	fs := flag.NewFlagSet("contextsqueeze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version")
	inPath := fs.String("input", "", "input file")
	outPath := fs.String("out", "", "output file path")
	aggr := fs.Int("aggr", -1, "aggressiveness 0..9")
	profile := fs.String("profile", "", "profile local|api")
	maxTokens := fs.Int("max-tokens", 0, "approx token budget")
	asJSON := fs.Bool("json", false, "emit json")
	source := fs.String("source", "auto", "source override: auto|pdf|docx|html|text")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version.Current())
		return 0
	}
	path, err := parseInputArg(fs, *inPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "usage: contextsqueeze [file] [--input file] [--max-tokens N] [--json] [--out path] [--source auto|pdf|docx|html|text]")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ing, err := ingest.Run(ctx, path, *source)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ingest error: %v\n", err)
		return 1
	}

	res, err := pipeline.RunResult(ing.Text, api.Options{Aggressiveness: *aggr, MaxTokens: *maxTokens, Profile: *profile}, ing.SourceType, ing.Warnings)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			_, _ = fmt.Fprintln(stderr, "timeout exceeded while squeezing")
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "squeeze error: %v\n", err)
		return 1
	}

	if statsMode {
		_, _ = fmt.Fprintf(stdout, "source: %s\n", res.SourceType)
		_, _ = fmt.Fprintf(stdout, "bytes in/out: %d/%d\n", res.BytesIn, res.BytesOut)
		_, _ = fmt.Fprintf(stdout, "tokens in/out (approx): %d/%d\n", res.TokensInApprox, res.TokensOutApprox)
		_, _ = fmt.Fprintf(stdout, "reduction: %.2f%%\n", res.ReductionPct)
		_, _ = fmt.Fprintf(stdout, "aggressiveness: %d\n", res.Aggressiveness)
		_, _ = fmt.Fprintf(stdout, "budget applied: %v\n", res.BudgetApplied)
		_, _ = fmt.Fprintf(stdout, "truncated: %v\n", res.Truncated)
		if len(res.Warnings) > 0 {
			_, _ = fmt.Fprintf(stdout, "warnings: %s\n", strings.Join(res.Warnings, "; "))
		}
		return 0
	}

	if *asJSON {
		jr := jsonResult{
			BytesIn:         res.BytesIn,
			BytesOut:        res.BytesOut,
			TokensInApprox:  res.TokensInApprox,
			TokensOutApprox: res.TokensOutApprox,
			ReductionPct:    res.ReductionPct,
			Aggressiveness:  res.Aggressiveness,
			Profile:         res.Profile,
			BudgetApplied:   res.BudgetApplied,
			Truncated:       res.Truncated,
			SourceType:      res.SourceType,
			Warnings:        res.Warnings,
		}
		if utf8.Valid(res.Text) {
			jr.Text = string(res.Text)
		} else {
			jr.TextB64 = base64.StdEncoding.EncodeToString(res.Text)
		}
		buf, err := json.MarshalIndent(jr, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "json error: %v\n", err)
			return 1
		}
		buf = append(buf, '\n')
		if err := writeOutput(*outPath, buf, stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "write output: %v\n", err)
			return 1
		}
		return 0
	}

	if err := writeOutput(*outPath, res.Text, stdout); err != nil {
		_, _ = fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}
	return 0
}

func runBench(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("contextsqueeze bench", flag.ContinueOnError)
	fs.SetOutput(stderr)
	inPath := fs.String("input", "", "input file")
	maxTokens := fs.Int("max-tokens", 0, "approx token budget")
	source := fs.String("source", "auto", "source override")
	profile := fs.String("profile", "", "profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	path, err := parseInputArg(fs, *inPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "usage: contextsqueeze bench <file> [--max-tokens N] [--source auto|pdf|docx|html|text]")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ing, err := ingest.Run(ctx, path, *source)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ingest error: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "| aggr | tokens out | reduction % | truncated |")
	_, _ = fmt.Fprintln(stdout, "|---:|---:|---:|:---:|")
	for a := 0; a <= 9; a++ {
		res, err := pipeline.RunResult(ing.Text, api.Options{Aggressiveness: a, MaxTokens: *maxTokens, Profile: *profile}, ing.SourceType, ing.Warnings)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "bench error at aggr %d: %v\n", a, err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "| %d | %d | %.2f | %v |\n", a, res.TokensOutApprox, res.ReductionPct, res.Truncated)
	}
	return 0
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "bench":
			return runBench(args[1:], stdout, stderr)
		case "stats":
			return runSqueeze(args[1:], stdout, stderr, true)
		}
	}
	return runSqueeze(args, stdout, stderr, false)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
