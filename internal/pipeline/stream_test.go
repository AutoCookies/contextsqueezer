package pipeline

import (
	"bytes"
	"contextsqueezer/pkg/api"
	"strconv"
	"strings"
	"testing"
	"time"
)

func makeLargeDoc(sentences int) []byte {
	var b strings.Builder
	for i := 0; i < sentences; i++ {
		if i%1000 == 0 {
			b.WriteString("# SECTION ")
			b.WriteString(strconv.Itoa(i / 1000))
			b.WriteString("\n")
		}
		b.WriteString("This is repeated boilerplate line for long document testing. ")
		b.WriteString("Variant token ")
		b.WriteString(strconv.Itoa(i % 37))
		b.WriteString(".\n")
	}
	return []byte(b.String())
}

func TestLargeDocDeterministicAndFast(t *testing.T) {
	in := makeLargeDoc(200000)
	start := time.Now()
	r1, err := RunResultWithConfig(in, api.Options{Aggressiveness: 0, Profile: "api"}, "text", nil, RunConfig{MaxMemoryMB: 1024})
	if err != nil {
		t.Fatalf("run1: %v", err)
	}
	r2, err := RunResultWithConfig(in, api.Options{Aggressiveness: 0, Profile: "api"}, "text", nil, RunConfig{MaxMemoryMB: 1024})
	if err != nil {
		t.Fatalf("run2: %v", err)
	}
	if !bytes.Equal(r1.Text, r2.Text) {
		t.Fatal("non-deterministic output")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("too slow: %v", elapsed)
	}
	if r1.Metrics.PeakMemoryEstimateB > int64(1024)*1024*1024 {
		t.Fatalf("memory estimate too high: %d", r1.Metrics.PeakMemoryEstimateB)
	}
}

func TestHeadingContinuity(t *testing.T) {
	in := []byte("# A\nkeep sentence after heading.\nnoise.\n# B\nanother keep sentence.\n")
	res, err := RunResultWithConfig(in, api.Options{Aggressiveness: 9}, "text", nil, RunConfig{MaxMemoryMB: 32})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(res.Text, []byte("keep sentence after heading")) {
		t.Fatal("missing continuity sentence after heading A")
	}
	if !bytes.Contains(res.Text, []byte("another keep sentence")) {
		t.Fatal("missing continuity sentence after heading B")
	}
}
