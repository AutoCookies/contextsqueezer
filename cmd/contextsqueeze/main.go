package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"contextsqueezer/internal/pipeline"
	"contextsqueezer/pkg/api"
)

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "bench" {
		return runBench(args[1:], stdout, stderr)
	}

	fs := flag.NewFlagSet("contextsqueeze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version")
	showStats := fs.Bool("stats", false, "print stats")
	aggr := fs.Int("aggr", -1, "aggressiveness 0..9")
	profile := fs.String("profile", "", "profile: local|api")

	parseArgs := reorderArgs(args)
	if err := fs.Parse(parseArgs); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, api.Version())
		return 0
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: contextsqueeze <input-file> [--aggr 0..9] [--profile local|api] [--stats]")
		fmt.Fprintln(stderr, "   or: contextsqueeze bench <input-file>")
		return 2
	}
	if *profile != "" && *profile != "local" && *profile != "api" {
		fmt.Fprintln(stderr, "invalid --profile (must be local|api)")
		return 2
	}
	if *aggr < -1 || *aggr > 9 {
		fmt.Fprintln(stderr, "invalid --aggr (must be -1..9)")
		return 2
	}

	in, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	opt := api.Options{Aggressiveness: *aggr, Profile: *profile}
	start := time.Now()
	out, err := pipeline.Run(in, opt)
	if err != nil {
		fmt.Fprintf(stderr, "squeeze: %v\n", err)
		return 1
	}
	elapsed := time.Since(start)

	if *showStats {
		printStats(stdout, in, out, elapsed)
		return 0
	}
	_, _ = stdout.Write(out)
	return 0
}

func reorderArgs(args []string) []string {
	flags := make([]string, 0, len(args))
	pos := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if a == "--version" || a == "--stats" {
			flags = append(flags, a)
			continue
		}
		if a == "--aggr" || a == "--profile" {
			flags = append(flags, a)
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

func runBench(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: contextsqueeze bench <input-file>")
		return 2
	}
	in, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "| aggr | bytes in | bytes out | approx tok in | approx tok out | words in | words out | reduction % | runtime ms | quality |")
	fmt.Fprintln(stdout, "|---:|---:|---:|---:|---:|---:|---:|---:|---:|:---:|")

	type row struct {
		aggr      int
		reduction float64
		quality   bool
	}
	rows := make([]row, 0, 10)
	for a := 0; a <= 9; a++ {
		start := time.Now()
		out, runErr := pipeline.Run(in, api.Options{Aggressiveness: a})
		if runErr != nil {
			fmt.Fprintf(stderr, "bench failed at aggr=%d: %v\n", a, runErr)
			return 1
		}
		elapsed := time.Since(start)
		qr := pipeline.AnalyzeQuality(in, out, a)
		qok := qr.AnchorRetentionOK && qr.SectionCoverageOK && qr.KeywordRecallOK
		reduction := reductionPercent(len(in), len(out))
		fmt.Fprintf(stdout, "| %d | %d | %d | %d | %d | %d | %d | %.2f | %d | %t |\n",
			a, len(in), len(out), pipeline.ApproxTokens(in), pipeline.ApproxTokens(out),
			pipeline.WordCount(in), pipeline.WordCount(out), reduction, elapsed.Milliseconds(), qok)
		rows = append(rows, row{aggr: a, reduction: reduction, quality: qok})
	}

	best := row{aggr: -1, reduction: -1, quality: false}
	for _, r := range rows {
		if r.quality && r.reduction > best.reduction {
			best = r
		}
	}
	if best.aggr < 0 {
		fmt.Fprintln(stderr, "warning: no aggressiveness level passed quality gates")
		return 1
	}
	fmt.Fprintf(stdout, "\nRecommended aggressiveness: %d (reduction %.2f%%, anchor retention 100%%)\n", best.aggr, best.reduction)
	return 0
}

func printStats(w io.Writer, in, out []byte, elapsed time.Duration) {
	fmt.Fprintf(w, "bytes in: %d\n", len(in))
	fmt.Fprintf(w, "bytes out: %d\n", len(out))
	fmt.Fprintf(w, "approx tokens in: %d\n", pipeline.ApproxTokens(in))
	fmt.Fprintf(w, "approx tokens out: %d\n", pipeline.ApproxTokens(out))
	fmt.Fprintf(w, "reduction %%: %.2f\n", reductionPercent(len(in), len(out)))
	fmt.Fprintf(w, "runtime ms: %d\n", elapsed.Milliseconds())
}

func reductionPercent(inBytes, outBytes int) float64 {
	if inBytes == 0 {
		return 0
	}
	return (1.0 - (float64(outBytes) / float64(inBytes))) * 100.0
}
