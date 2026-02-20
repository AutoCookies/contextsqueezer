package api

import (
	"bytes"
	"testing"
)

func TestVersionNonEmpty(t *testing.T) {
	if Version() == "" {
		t.Fatal("Version() is empty")
	}
}

func TestAggressivenessZeroIdentity(t *testing.T) {
	in := []byte("Hello world. Keep me intact.")
	got, err := SqueezeBytes(in, Options{Aggressiveness: 0})
	if err != nil {
		t.Fatalf("SqueezeBytes returned error: %v", err)
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("aggr=0 should be identity")
	}
}

func TestSqueezeBytesShorterForAggressiveInput(t *testing.T) {
	in := []byte("" +
		"NOTICE: SAMPLE DISCLAIMER NOTICE: SAMPLE DISCLAIMER NOTICE: SAMPLE DISCLAIMER NOTICE: SAMPLE DISCLAIMER NOTICE: SAMPLE DISCLAIMER\n\n" +
		"The quick brown fox jumps over the lazy dog.\n" +
		"The quick brown fox jumps over the lazy dog.\n" +
		"The quick brown fox jumps over the lazy dog.\n" +
		"# IMPORTANT\nThis section must remain with https://example.com and code ```block```.\n")
	got, err := SqueezeBytes(in, Options{Aggressiveness: 6})
	if err != nil {
		t.Fatalf("SqueezeBytes returned error: %v", err)
	}
	if len(got) >= len(in) {
		t.Fatalf("expected compressed output to be shorter: in=%d out=%d", len(in), len(got))
	}
}

func TestSqueezeBytesBinaryWithZeros(t *testing.T) {
	in := []byte{0x00, 0x01, 0x00, 0x02, 0xff, 0x00}
	got, err := SqueezeBytes(in, Options{Aggressiveness: 0})
	if err != nil {
		t.Fatalf("SqueezeBytes returned error: %v", err)
	}
	if !bytes.Equal(got, in) {
		t.Fatalf("SqueezeBytes mismatch: got=%v want=%v", got, in)
	}
}
