package api

import (
	"bytes"
	"testing"
)

func TestVersionNonEmpty(t *testing.T) {
	if Version() == "" {
		t.Fatal("version is empty")
	}
}

func TestSqueezeAggressivenessZeroIdentity(t *testing.T) {
	in := []byte("alpha\n\nalpha\n\n# HEAD\nVisit https://example.com\n")
	out, err := SqueezeBytes(in, Options{Aggressiveness: 0})
	if err != nil {
		t.Fatalf("SqueezeBytes returned error: %v", err)
	}
	if !bytes.Equal(in, out) {
		t.Fatalf("expected identity output")
	}
}

func TestSqueezeAggressivenessHighShorter(t *testing.T) {
	in := []byte("Noise sentence repeated. Noise sentence repeated. Noise sentence repeated.\n" +
		"Unique insight with token quartz99 appears once.\n" +
		"Noise sentence repeated.\n")
	out, err := SqueezeBytes(in, Options{Aggressiveness: 6})
	if err != nil {
		t.Fatalf("SqueezeBytes returned error: %v", err)
	}
	if len(out) >= len(in) {
		t.Fatalf("expected shorter output at higher aggressiveness; in=%d out=%d", len(in), len(out))
	}
}
