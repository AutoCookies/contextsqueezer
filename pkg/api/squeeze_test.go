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

func TestSqueezeBytesIdentity(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "empty", input: []byte{}},
		{name: "text", input: []byte("hello world")},
		{name: "binary", input: []byte{'a', 0, 'b', '\n', 0, 'z'}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := SqueezeBytes(tc.input, Options{})
			if err != nil {
				t.Fatalf("SqueezeBytes returned error: %v", err)
			}
			if !bytes.Equal(tc.input, out) {
				t.Fatalf("output mismatch, got %v want %v", out, tc.input)
			}
		})
	}
}
