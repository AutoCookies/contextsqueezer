package pipeline

import (
	"bytes"
	"container/list"
	"contextsqueezer/internal/metrics"
	"contextsqueezer/internal/runtime"
	"contextsqueezer/pkg/api"
	"hash/fnv"
	"strings"
	"time"
)

const (
	defaultChunkSentences = 500
	defaultRegistryCap    = 100000
)

type RunConfig struct {
	MaxMemoryMB int
}

type sigRegistry struct {
	cap   int
	lru   *list.List
	index map[uint64]*list.Element
}

func newSigRegistry(capacity int) *sigRegistry {
	if capacity <= 0 {
		capacity = defaultRegistryCap
	}
	return &sigRegistry{cap: capacity, lru: list.New(), index: map[uint64]*list.Element{}}
}

func (r *sigRegistry) has(h uint64) bool {
	el, ok := r.index[h]
	if !ok {
		return false
	}
	r.lru.MoveToFront(el)
	return true
}

func (r *sigRegistry) add(h uint64) {
	if el, ok := r.index[h]; ok {
		r.lru.MoveToFront(el)
		return
	}
	el := r.lru.PushFront(h)
	r.index[h] = el
	if r.lru.Len() > r.cap {
		last := r.lru.Back()
		if last != nil {
			v := last.Value.(uint64)
			delete(r.index, v)
			r.lru.Remove(last)
		}
	}
}

// Complexity: O(n log n) over token sorting per sentence; no full pairwise document comparisons.
func sentenceSignature(in []byte) uint64 {
	toks := strings.FieldsFunc(strings.ToLower(string(in)), func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	})
	if len(toks) == 0 {
		h := fnv.New64a()
		_, _ = h.Write(bytes.TrimSpace(in))
		return h.Sum64()
	}
	uniq := map[string]struct{}{}
	for _, t := range toks {
		if t != "" {
			uniq[t] = struct{}{}
		}
	}
	type kv struct {
		h uint64
		t string
	}
	top := make([]kv, 0, 6)
	for tok := range uniq {
		th := fnv.New64a()
		_, _ = th.Write([]byte(tok))
		item := kv{h: th.Sum64(), t: tok}
		if len(top) < 6 {
			top = append(top, item)
			for i := len(top) - 1; i > 0 && top[i].h < top[i-1].h; i-- {
				top[i], top[i-1] = top[i-1], top[i]
			}
			continue
		}
		if item.h >= top[len(top)-1].h {
			continue
		}
		top[len(top)-1] = item
		for i := len(top) - 1; i > 0 && top[i].h < top[i-1].h; i-- {
			top[i], top[i-1] = top[i-1], top[i]
		}
	}
	h := fnv.New64a()
	for i, t := range top {
		if i > 0 {
			_, _ = h.Write([]byte{'|'})
		}
		_, _ = h.Write([]byte(t.t))
	}
	return h.Sum64()
}

func splitChunks(in []byte) [][]byte {
	spans := segmentSentences(in)
	if len(spans) == 0 {
		return [][]byte{in}
	}
	chunks := make([][]byte, 0)
	start := 0
	count := 0
	for i, sp := range spans {
		s := bytes.TrimSpace(in[sp.s:sp.e])
		isHeading := len(s) > 0 && s[0] == '#'
		if i > start && (isHeading || count >= defaultChunkSentences) {
			chunks = append(chunks, append([]byte{}, in[spans[start].s:spans[i-1].e]...))
			start = i
			count = 0
		}
		count++
	}
	chunks = append(chunks, append([]byte{}, in[spans[start].s:spans[len(spans)-1].e]...))
	return chunks
}

func ensureHeadingContinuity(in []byte, out []byte, truncated bool) []byte {
	if truncated {
		return out
	}
	orig := segmentSentences(in)
	kept := segmentSentences(out)
	if len(orig) == 0 || len(kept) == 0 {
		return out
	}
	outSet := map[string]struct{}{}
	for _, sp := range kept {
		outSet[string(bytes.TrimSpace(out[sp.s:sp.e]))] = struct{}{}
	}
	for i, sp := range orig {
		s := bytes.TrimSpace(in[sp.s:sp.e])
		if len(s) == 0 || s[0] != '#' {
			continue
		}
		if i+1 >= len(orig) {
			continue
		}
		next := bytes.TrimSpace(in[orig[i+1].s:orig[i+1].e])
		if len(next) == 0 || next[0] == '#' {
			continue
		}
		if _, ok := outSet[string(next)]; !ok {
			out = append(out, '\n')
			out = append(out, next...)
		}
	}
	return out
}

func squeezeStreamed(in []byte, opt api.Options, _ RunConfig, tracker *runtime.MemoryTracker, warnings *[]string) ([]byte, metrics.StageMetrics, int, error) {
	m := metrics.StageMetrics{}
	segStart := time.Now()
	chunks := splitChunks(in)
	m.SegmentationMS = time.Since(segStart).Milliseconds()
	reg := newSigRegistry(defaultRegistryCap)
	keptChunks := make([][]byte, 0, len(chunks))
	currentAggr := normalizeAggr(opt)

	for _, ch := range chunks {
		if tracker.Add(int64(len(ch))) {
			if currentAggr > 0 {
				currentAggr--
			}
			*warnings = append(*warnings, "memory soft limit exceeded; reducing aggressiveness")
		}
		pruneStart := time.Now()
		out, err := api.SqueezeBytes(ch, api.Options{Aggressiveness: currentAggr, Profile: opt.Profile, MaxTokens: opt.MaxTokens})
		m.PruneMS += time.Since(pruneStart).Milliseconds()
		tracker.Release(int64(len(ch)))
		if err != nil {
			return nil, m, currentAggr, err
		}
		nm := api.LastNativeMetrics()
		m.TokensParsed += nm.TokensParsed
		m.SentencesTotal += nm.SentencesTotal
		m.SimilarityCandidates += nm.SimilarityCandidates
		m.SimilarityPairs += nm.SimilarityPairs

		dedStart := time.Now()
		sp := segmentSentences(out)
		var b bytes.Buffer
		for _, s := range sp {
			sent := out[s.s:s.e]
			h := sentenceSignature(sent)
			if reg.has(h) {
				continue
			}
			reg.add(h)
			_, _ = b.Write(sent)
			_ = tracker.Add(int64(len(sent)) + 64)
		}
		m.CrossChunkRegistryMS += time.Since(dedStart).Milliseconds()
		keptChunks = append(keptChunks, b.Bytes())
	}

	reStart := time.Now()
	out := bytes.Join(keptChunks, []byte("\n"))
	m.AssemblyMS = time.Since(reStart).Milliseconds()
	m.PeakMemoryEstimateB = tracker.Peak
	return out, m, currentAggr, nil
}
