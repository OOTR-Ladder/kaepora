package back // nolint:testpackage

import (
	"testing"
	"time"
)

func TestPeriodCompute(t *testing.T) {
	type entry struct {
		fn              func(time.Time) time.Time
		input, expected string
	}

	// false positive for some reason
	// nolint:gofmt
	cases := []entry{
		entry{currentPeriodStart, "2020-05-15 02:00 CET", "2020-05-11"},
		entry{currentPeriodStart, "2020-05-11 02:00 CET", "2020-05-11"},
		entry{currentPeriodStart, "2020-05-10 02:00 CET", "2020-05-04"},

		entry{previousPeriodStart, "2020-05-15 02:00 CET", "2020-05-04"},
		entry{previousPeriodStart, "2020-05-11 02:00 CET", "2020-05-04"},
		entry{previousPeriodStart, "2020-05-10 02:00 CET", "2020-04-27"},

		entry{nextPeriodStart, "2020-05-15 02:00 CET", "2020-05-18"},
		entry{nextPeriodStart, "2020-05-11 02:00 CET", "2020-05-18"},
		entry{nextPeriodStart, "2020-05-10 02:00 CET", "2020-05-11"},

		// The tricky cases, where intepreting dow in the wrong TZ could mess
		// up the results.
		entry{currentPeriodStart, "2020-05-15 00:00 CET", "2020-05-11"},
		entry{currentPeriodStart, "2020-05-11 00:00 CET", "2020-05-04"},
		entry{currentPeriodStart, "2020-05-10 00:00 CET", "2020-05-04"},
	}

	for k, v := range cases {
		input, err := time.Parse("2006-01-02 15:04 MST", v.input)
		if err != nil {
			t.Fatal(err)
		}

		actual := v.fn(input).Format("2006-01-02")
		if actual != v.expected {
			t.Errorf("case #%d: expected %s got %s", k, v.expected, actual)
		}
	}
}
