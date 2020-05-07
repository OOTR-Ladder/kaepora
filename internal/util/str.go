package util

import (
	"strings"
	"time"
)

// TruncateDuration prettifies a duration by removing useless units.
// eg. 1h20m0s -> 1h20m
// It does not round/truncate the duration, it only works on the string.
func TruncateDuration(v time.Duration) string {
	ret := strings.TrimSuffix(v.Truncate(time.Second).String(), "0s")
	if strings.HasSuffix(ret, "h0m") {
		return strings.TrimSuffix(ret, "0m")
	}

	return ret
}
