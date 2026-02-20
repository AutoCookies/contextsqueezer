package pipeline

import (
	"contextsqueezer/internal/metrics"
	"contextsqueezer/internal/runtime"
	"contextsqueezer/pkg/api"
	"errors"
	"math"
	"time"
)

type Result struct {
	Text            []byte               `json:"-"`
	BytesIn         int                  `json:"bytes_in"`
	BytesOut        int                  `json:"bytes_out"`
	TokensInApprox  int                  `json:"tokens_in_approx"`
	TokensOutApprox int                  `json:"tokens_out_approx"`
	ReductionPct    float64              `json:"reduction_pct"`
	Aggressiveness  int                  `json:"aggressiveness"`
	Profile         string               `json:"profile"`
	BudgetApplied   bool                 `json:"budget_applied"`
	Truncated       bool                 `json:"truncated"`
	SourceType      string               `json:"source_type"`
	Warnings        []string             `json:"warnings"`
	Metrics         metrics.StageMetrics `json:"metrics"`
}

func fastWordCount(in []byte) int {
	inWord := false
	count := 0
	for _, b := range in {
		ws := b == ' ' || b == '\n' || b == '\t' || b == '\r'
		if ws {
			inWord = false
			continue
		}
		if !inWord {
			count++
			inWord = true
		}
	}
	return count
}

func approxTokens(in []byte) int {
	if len(in) == 0 {
		return 0
	}
	words := fastWordCount(in)
	return int(math.Ceil(float64(len(in))/4.0)) + words
}

func reductionPct(in, out int) float64 {
	if in == 0 {
		return 0
	}
	return (1.0 - float64(out)/float64(in)) * 100.0
}

func normalizeAggr(opt api.Options) int {
	if opt.Aggressiveness >= 0 && opt.Aggressiveness <= 9 {
		return opt.Aggressiveness
	}
	if opt.Profile == "local" {
		return 6
	}
	if opt.Profile == "api" {
		return 4
	}
	return 4
}

func Run(in []byte, opt api.Options) ([]byte, error) {
	res, err := RunResult(in, opt, "text", nil)
	if err != nil {
		return nil, err
	}
	return res.Text, nil
}

func RunResult(in []byte, opt api.Options, sourceType string, warnings []string) (Result, error) {
	return RunResultWithConfig(in, opt, sourceType, warnings, RunConfig{MaxMemoryMB: 1024})
}

func RunResultWithConfig(in []byte, opt api.Options, sourceType string, warnings []string, cfg RunConfig) (Result, error) {
	if sourceType == "" {
		sourceType = "text"
	}
	if cfg.MaxMemoryMB <= 0 {
		cfg.MaxMemoryMB = 1024
	}
	budgetApplied := opt.MaxTokens > 0
	attempts := 0
	best := in
	current := normalizeAggr(opt)
	tracker := runtime.NewMemoryTracker(cfg.MaxMemoryMB)
	allWarnings := append([]string{}, warnings...)
	m := metrics.StageMetrics{}
	budgetStart := time.Now()

	for {
		attempts++
		if attempts > 10 {
			break
		}
		out, stage, usedAggr, err := squeezeStreamed(in, api.Options{Aggressiveness: current, Profile: opt.Profile, MaxTokens: opt.MaxTokens}, cfg, tracker, &allWarnings)
		if err != nil {
			return Result{}, err
		}
		best = out
		current = usedAggr
		m.SegmentationMS += stage.SegmentationMS
		m.TokenizationMS += stage.TokenizationMS
		m.CandidateFilterMS += stage.CandidateFilterMS
		m.SimilarityMS += stage.SimilarityMS
		m.PruneMS += stage.PruneMS
		m.AssemblyMS += stage.AssemblyMS
		m.CrossChunkRegistryMS += stage.CrossChunkRegistryMS
		m.SimilarityCandidates += stage.SimilarityCandidates
		m.SimilarityPairs += stage.SimilarityPairs
		m.TokensParsed += stage.TokensParsed
		m.SentencesTotal += stage.SentencesTotal
		if opt.MaxTokens <= 0 || approxTokens(out) <= opt.MaxTokens || current >= 9 {
			break
		}
		current++
	}

	truncated := false
	if opt.MaxTokens > 0 && approxTokens(best) > opt.MaxTokens {
		var err error
		best, err = truncateToBudget(best, opt.MaxTokens)
		if err != nil {
			return Result{}, err
		}
		truncated = true
	}
	best = ensureHeadingContinuity(in, best, truncated)
	if opt.MaxTokens > 0 && approxTokens(best) > opt.MaxTokens {
		return Result{}, errors.New("unable to satisfy max token budget")
	}
	m.BudgetLoopMS = time.Since(budgetStart).Milliseconds()
	m.PeakMemoryEstimateB = tracker.Peak

	return Result{
		Text:            best,
		BytesIn:         len(in),
		BytesOut:        len(best),
		TokensInApprox:  approxTokens(in),
		TokensOutApprox: approxTokens(best),
		ReductionPct:    reductionPct(len(in), len(best)),
		Aggressiveness:  current,
		Profile:         opt.Profile,
		BudgetApplied:   budgetApplied,
		Truncated:       truncated,
		SourceType:      sourceType,
		Warnings:        allWarnings,
		Metrics:         m,
	}, nil
}
