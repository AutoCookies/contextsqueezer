package pipeline

import "contextsqueezer/pkg/api"

func Run(in []byte, opt api.Options) ([]byte, error) {
	return api.SqueezeBytes(in, opt)
}
