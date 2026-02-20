package version

import "contextsqueezer/pkg/api"

func Current() string {
	return api.Version()
}
