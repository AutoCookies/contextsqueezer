package api

import "fmt"

type NativeMetrics struct {
	TokensParsed         uint64
	SentencesTotal       uint64
	SimilarityCandidates uint64
	SimilarityPairs      uint64
}

type Options struct {
	Aggressiveness int
	MaxTokens      int
	Profile        string
}

func Version() string {
	return csqVersion()
}

func normalizeAggressiveness(opt Options) int {
	if opt.Aggressiveness >= 0 && opt.Aggressiveness <= 9 {
		return opt.Aggressiveness
	}
	if opt.Aggressiveness > 9 {
		return 9
	}
	if opt.Profile == "local" {
		return 6
	}
	if opt.Profile == "api" {
		return 4
	}
	return 4
}

// SqueezeBytes must be stable and deterministic.
func SqueezeBytes(in []byte, opt Options) ([]byte, error) {
	return SqueezeBytesWithProgress(in, opt, nil)
}

// SqueezeBytesWithProgress supports an optional progress callback.
func SqueezeBytesWithProgress(in []byte, opt Options, cb func(float32)) ([]byte, error) {
	aggr := normalizeAggressiveness(opt)
	var pCB *func(float32)
	if cb != nil {
		pCB = &cb
	}
	out, err := csqSqueeze(in, aggr, pCB)
	if err != nil {
		return nil, fmt.Errorf("squeeze failed: %w", err)
	}
	return out, nil
}

func LastNativeMetrics() NativeMetrics {
	return csqLastMetrics()
}
