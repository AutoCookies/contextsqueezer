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

func TestSqueezeBytesIdentity(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{name: "empty", in: []byte{}},
		{name: "text", in: []byte("hello context squeezer")},
		{name: "binary with zeros", in: []byte{0x00, 0x01, 0x00, 0x02, 0xff, 0x00}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SqueezeBytes(tc.in, Options{})
			if err != nil {
				t.Fatalf("SqueezeBytes returned error: %v", err)
			}
			if !bytes.Equal(got, tc.in) {
				t.Fatalf("SqueezeBytes mismatch: got=%v want=%v", got, tc.in)
			}
		})
	}
}
