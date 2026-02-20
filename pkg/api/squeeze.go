package api

import "fmt"

type Options struct {
	// Reserved for Phase 1+, keep now for forward-compat.
	Aggressiveness int
	MaxTokens      int
	Profile        string
}

func Version() string {
	return csqVersion()
}

// SqueezeBytes must be stable and deterministic.
// In Phase 0 it returns the input unchanged, but still calls into C++.
func SqueezeBytes(in []byte, opt Options) ([]byte, error) {
	_ = opt
	out, err := csqSqueeze(in)
	if err != nil {
		return nil, fmt.Errorf("squeeze failed: %w", err)
	}
	return out, nil
}
