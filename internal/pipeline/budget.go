package pipeline

import (
	"bytes"
	"errors"
)

type span struct{ s, e int }

func segmentSentences(in []byte) []span {
	spans := make([]span, 0)
	start := 0
	for i := 0; i < len(in); i++ {
		if i+1 < len(in) && in[i] == '\n' && in[i+1] == '\n' {
			if i > start {
				spans = append(spans, span{start, i})
			}
			spans = append(spans, span{i, i + 2})
			i++
			start = i + 1
			continue
		}
		if in[i] == '.' || in[i] == '?' || in[i] == '!' {
			end := i + 1
			for end < len(in) && (in[end] == ' ' || in[end] == '\n' || in[end] == '\t' || in[end] == '\r') {
				if end+1 < len(in) && in[end] == '\n' && in[end+1] == '\n' {
					break
				}
				end++
			}
			spans = append(spans, span{start, end})
			start = end
			i = end - 1
		}
	}
	if start < len(in) {
		spans = append(spans, span{start, len(in)})
	}
	return spans
}

func isAnchorSentence(s []byte) bool {
	if bytes.Contains(s, []byte("```")) || bytes.Contains(s, []byte("http://")) || bytes.Contains(s, []byte("https://")) {
		return true
	}
	digits := 0
	for _, b := range s {
		if b >= '0' && b <= '9' {
			digits++
		}
	}
	if digits >= 4 {
		return true
	}
	trim := bytes.TrimSpace(s)
	return len(trim) > 0 && trim[0] == '#'
}

func joinChunks(chunks [][]byte) []byte { return bytes.Join(chunks, nil) }

func joinedWith(chunks [][]byte, extra []byte) []byte {
	all := make([][]byte, 0, len(chunks)+1)
	all = append(all, chunks...)
	all = append(all, extra)
	return joinChunks(all)
}

func truncateToBudget(in []byte, maxTokens int) ([]byte, error) {
	if maxTokens <= 0 {
		return in, nil
	}
	spans := segmentSentences(in)
	if len(spans) == 0 {
		return []byte{}, nil
	}
	kept := make([][]byte, 0, len(spans))
	for _, sp := range spans {
		s := append([]byte{}, in[sp.s:sp.e]...)
		trial := joinedWith(kept, s)
		if approxTokens(trial) <= maxTokens {
			kept = append(kept, s)
			continue
		}
		if isAnchorSentence(s) {
			for len(kept) > 0 && approxTokens(joinedWith(kept, s)) > maxTokens {
				if !isAnchorSentence(kept[len(kept)-1]) {
					kept = kept[:len(kept)-1]
				} else {
					break
				}
			}
			if approxTokens(joinedWith(kept, s)) <= maxTokens {
				kept = append(kept, s)
			}
		}
		break
	}
	if len(kept) == 0 {
		return nil, errors.New("budget too small to keep any sentence")
	}
	return joinChunks(kept), nil
}
