package api

import "testing"

func FuzzSqueezeBytes(f *testing.F) {
	f.Add([]byte("hello world"), 0)
	f.Add([]byte("# H\nVisit https://example.com\n"), 6)
	f.Fuzz(func(t *testing.T, data []byte, aggr int) {
		if aggr < 0 {
			aggr = 0
		}
		if aggr > 9 {
			aggr = 9
		}
		_, _ = SqueezeBytes(data, Options{Aggressiveness: aggr})
	})
}
