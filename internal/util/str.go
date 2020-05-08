package util

import (
	"fmt"
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

// Datetime is the format to use anywhere we need to output a date+time to an user.
func Datetime(iface interface{}) string {
	var t time.Time
	switch iface := iface.(type) {
	case time.Time:
		t = iface
	case TimeAsDateTimeTZ:
		t = iface.Time()
	default:
		panic(fmt.Errorf("unexpected type %T", iface))
	}

	return t.Format("2006-01-02 15h04 MST")
}
