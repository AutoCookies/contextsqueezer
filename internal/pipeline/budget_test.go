package pipeline

import (
	"bytes"
	"contextsqueezer/pkg/api"
	"testing"
)

func TestBudgetHonored(t *testing.T) {
	in := []byte("# HEAD\nVisit https://example.com\n" +
		"Noise sentence. Noise sentence. Noise sentence. Noise sentence.\n" +
		"Another noise sentence. Another noise sentence.\n")
	res, err := RunResult(in, api.Options{Aggressiveness: 2, MaxTokens: 20, Profile: "api"}, "text", nil)
	if err != nil {
		t.Fatalf("RunResult: %v", err)
	}
	if res.TokensOutApprox > 20 {
		t.Fatalf("budget not honored: %d", res.TokensOutApprox)
	}
}

func TestTruncationOnlyWhenNeeded(t *testing.T) {
	in := []byte("Short text.\n")
	res, err := RunResult(in, api.Options{Aggressiveness: 0, MaxTokens: 100, Profile: "api"}, "text", nil)
	if err != nil {
		t.Fatalf("RunResult: %v", err)
	}
	if res.Truncated {
		t.Fatal("should not truncate when under budget")
	}
}

func TestAnchorRetentionNonTruncation(t *testing.T) {
	in := []byte("# HEADER\nVisit https://example.com/docs\nKeep this sentence.\n")
	res, err := RunResult(in, api.Options{Aggressiveness: 4, MaxTokens: 200}, "text", nil)
	if err != nil {
		t.Fatalf("RunResult: %v", err)
	}
	if res.Truncated {
		t.Fatal("unexpected truncation")
	}
	if !bytes.Contains(res.Text, []byte("# HEADER")) || !bytes.Contains(res.Text, []byte("https://example.com/docs")) {
		t.Fatal("anchors must be retained in non-truncation case")
	}
}
