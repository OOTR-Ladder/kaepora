package back // nolint:testpackage

import (
	"testing"
	"time"
)

func TestPeriodCompute(t *testing.T) {
	type entry struct {
		fn                         func(time.Time) time.Time
		input, inputZone, expected string
	}

	// false positive for some reason
	// nolint:gofmt
	cases := []entry{
		entry{currentPeriodStart, "2020-05-15 02:00", "Europe/Paris", "2020-05-11 00:00 UTC"},
		entry{currentPeriodStart, "2020-05-11 02:00", "Europe/Paris", "2020-05-11 00:00 UTC"},
		entry{currentPeriodStart, "2020-05-10 02:00", "Europe/Paris", "2020-05-04 00:00 UTC"},

		entry{previousPeriodStart, "2020-05-15 02:00", "Europe/Paris", "2020-05-04 00:00 UTC"},
		entry{previousPeriodStart, "2020-05-11 02:00", "Europe/Paris", "2020-05-04 00:00 UTC"},
		entry{previousPeriodStart, "2020-05-10 02:00", "Europe/Paris", "2020-04-27 00:00 UTC"},

		entry{nextPeriodStart, "2020-05-15 02:00", "Europe/Paris", "2020-05-18 00:00 UTC"},
		entry{nextPeriodStart, "2020-05-11 02:00", "Europe/Paris", "2020-05-18 00:00 UTC"},
		entry{nextPeriodStart, "2020-05-10 02:00", "Europe/Paris", "2020-05-11 00:00 UTC"},

		// The tricky cases, where intepreting dow in the wrong TZ could mess
		// up the results.
		entry{currentPeriodStart, "2020-05-15 00:00", "Europe/Paris", "2020-05-11 00:00 UTC"},
		entry{currentPeriodStart, "2020-05-11 00:00", "Europe/Paris", "2020-05-04 00:00 UTC"},
		entry{currentPeriodStart, "2020-05-10 00:00", "Europe/Paris", "2020-05-04 00:00 UTC"},
	}

	for k, v := range cases {
		zone, err := time.LoadLocation(v.inputZone)
		if err != nil {
			t.Fatal(err)
		}

		input, err := time.ParseInLocation("2006-01-02 15:04", v.input, zone)
		if err != nil {
			t.Fatal(err)
		}

		actual := v.fn(input).Format("2006-01-02 15:04 MST")
		if actual != v.expected {
			t.Errorf("case #%d: expected %s got %s", k, v.expected, actual)
		}
	}
}
