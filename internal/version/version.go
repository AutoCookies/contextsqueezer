package version

import "contextsqueezer/pkg/api"

var Version = ""

func Current() string {
	if Version != "" {
		return Version
	}
	return api.Version()
}
