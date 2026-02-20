package api

import "testing"

func TestVersionNonEmpty(t *testing.T) {
	if Version() == "" {
		t.Fatal("expected non-empty version")
	}
}

func TestSqueezeIdentityCases(t *testing.T) {
	cases := [][]byte{
		{},
		[]byte("hello world"),
		{'a', 0, 'b', '\n', 'c'},
	}
	for _, in := range cases {
		out, err := SqueezeBytes(in, Options{Aggressiveness: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(out) != string(in) || len(out) != len(in) {
			t.Fatalf("expected identity output")
		}
	}
}

func TestAggressiveShortensCraftedInput(t *testing.T) {
	in := []byte("Alpha beta gamma are present. Alpha beta gamma are present! " +
		"DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER DISCLAIMER.\n\n" +
		"Tail sentence remains.")
	out, err := SqueezeBytes(in, Options{Aggressiveness: 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) >= len(in) {
		t.Fatalf("expected compressed output, got in=%d out=%d", len(in), len(out))
	}
}
