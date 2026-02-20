package pipeline

import (
	"bytes"
	"contextsqueezer/pkg/api"
	"errors"
	"math"
)

type Result struct {
	Text            []byte   `json:"-"`
	BytesIn         int      `json:"bytes_in"`
	BytesOut        int      `json:"bytes_out"`
	TokensInApprox  int      `json:"tokens_in_approx"`
	TokensOutApprox int      `json:"tokens_out_approx"`
	ReductionPct    float64  `json:"reduction_pct"`
	Aggressiveness  int      `json:"aggressiveness"`
	Profile         string   `json:"profile"`
	BudgetApplied   bool     `json:"budget_applied"`
	Truncated       bool     `json:"truncated"`
	SourceType      string   `json:"source_type"`
	Warnings        []string `json:"warnings"`
}

func approxTokens(in []byte) int {
	if len(in) == 0 {
		return 0
	}
	words := len(bytes.Fields(in))
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
	if sourceType == "" {
		sourceType = "text"
	}
	aggr := normalizeAggr(opt)
	budgetApplied := opt.MaxTokens > 0
	attempts := 0
	best := in
	current := aggr
	for {
		attempts++
		if attempts > 10 {
			break
		}
		out, err := api.SqueezeBytes(in, api.Options{Aggressiveness: current, MaxTokens: opt.MaxTokens, Profile: opt.Profile})
		if err != nil {
			return Result{}, err
		}
		best = out
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
	if opt.MaxTokens > 0 && approxTokens(best) > opt.MaxTokens {
		return Result{}, errors.New("unable to satisfy max token budget")
	}
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
		Warnings:        warnings,
	}, nil
}
