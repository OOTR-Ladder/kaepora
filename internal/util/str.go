package util

import (
	"fmt"
	"strings"
	"time"
)

// FormatDuration prettifies a duration by removing useless units.
// eg. 1h20m0s -> 1h20m
// It does not round/truncate the duration, it only works on the string.
// TODO: This should be localized.
func FormatDuration(d time.Duration) string {
	var prefix string
	if d > (24 * time.Hour) {
		prefix = fmt.Sprintf("%dd", d/(24*time.Hour))
		// Don't need minutes if its in more than a day
		d = (d % (24 * time.Hour)).Truncate(time.Hour)
	}

	ret := strings.TrimSuffix(d.Truncate(time.Second).String(), "0s")
	if strings.HasSuffix(ret, "h0m") {
		return prefix + strings.TrimSuffix(ret, "0m")
	}

	return prefix + ret
}

// Datetime is the format to use anywhere we need to output a date+time to an user.
func Datetime(iface interface{}) string {
	var t time.Time
	switch iface := iface.(type) {
	case time.Time:
		t = iface
	case TimeAsDateTimeTZ:
		t = iface.Time()
	case TimeAsTimestamp:
		t = iface.Time()
	default:
		panic(fmt.Errorf("unexpected type %T", iface))
	}

	return t.Format("2006-01-02 15h04 MST")
}

// Date is the format to use anywhere we need to output a date to an user.
func Date(iface interface{}) string {
	var t time.Time
	switch iface := iface.(type) {
	case time.Time:
		t = iface
	case TimeAsDateTimeTZ:
		t = iface.Time()
	case TimeAsTimestamp:
		t = iface.Time()
	default:
		panic(fmt.Errorf("unexpected type %T", iface))
	}

	return t.Format("2006-01-02")
}
