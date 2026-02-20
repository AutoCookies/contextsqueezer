//go:build !cgo

package api

func csqVersion() string { return "1.0.0" }

func csqSqueeze(in []byte, _ int) ([]byte, error) {
	out := make([]byte, len(in))
	copy(out, in)
	return out, nil
}

func csqLastMetrics() NativeMetrics { return NativeMetrics{} }
