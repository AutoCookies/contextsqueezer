package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EFixtures(t *testing.T) {
	fixtures := []string{"sample.txt", "sample.html", "sample.pdf", "sample.docx"}
	for _, f := range fixtures {
		t.Run(f, func(t *testing.T) {
			p := filepath.Join("..", "..", "internal", "ingest", "fixtures", f)
			cmd := exec.Command("go", "run", ".", "stats", "--source", "auto", "--max-tokens", "80", p)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("e2e failed: %v\n%s", err, string(out))
			}
			if !strings.Contains(string(out), "tokens in/out") {
				t.Fatalf("unexpected output: %s", string(out))
			}
		})
	}
}
