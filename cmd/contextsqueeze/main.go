package main

import (
	"flag"
	"fmt"
	"os"

	"contextsqueezer/internal/pipeline"
	"contextsqueezer/pkg/api"
)

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.Parse()

	if *showVersion {
		fmt.Println(api.Version())
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: contextsqueeze <input-file>\n")
		os.Exit(2)
	}

	inputPath := flag.Arg(0)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	out, err := pipeline.Run(data, api.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "squeeze: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stdout.Write(out); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}
