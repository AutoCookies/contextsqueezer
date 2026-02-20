package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"contextsqueezer/internal/pipeline"
	"contextsqueezer/internal/version"
	"contextsqueezer/pkg/api"
)

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("contextsqueeze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		_, _ = fmt.Fprintln(stdout, version.Current())
		return 0
	}

	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: contextsqueeze [--version] <input-file>")
		return 2
	}

	inPath := fs.Arg(0)
	data, err := os.ReadFile(inPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}

	out, err := pipeline.Run(data, api.Options{})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "squeeze: %v\n", err)
		return 1
	}

	if _, err := stdout.Write(out); err != nil {
		if !errors.Is(err, io.EOF) {
			_, _ = fmt.Fprintf(stderr, "write output: %v\n", err)
		}
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
