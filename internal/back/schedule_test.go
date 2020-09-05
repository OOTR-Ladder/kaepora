package back_test

import (
	"kaepora/internal/back"
	"testing"
	"time"
)

func TestScheduleNewIsNotNil(t *testing.T) {
	s := back.NewSchedule()
	if s.Mon == nil || s.Tue == nil || s.Wed == nil || s.Thu == nil ||
		s.Fri == nil || s.Sat == nil || s.Sun == nil {
		t.Error("A newly created schedule should have empty slices, not nil slices.")
	}
}

func TestScheduleNextBetween(t *testing.T) {
	s := back.NewSchedule()
	s.Mon = []string{"05:00 Europe/Paris", "05:00 Europe/Dublin"}
	s.Tue = []string{"15:00 Europe/Paris", "15:00 Europe/Dublin"}
	s.Fri = []string{"10:00 UTC"}

	testSchedule(t, s, []scheduleTestData{
		{
			now:      "2020-04-07 12:59:59+00:00",
			expected: "2020-04-07 15:00:00+02:00",
		},
		{
			now:      "2020-04-07 14:59:59+02:00",
			expected: "2020-04-07 15:00:00+02:00",
		},
		{
			now:      "2020-04-06 18:02:09+02:00",
			expected: "2020-04-07 15:00:00+02:00",
		},
		{
			now:      "2020-04-07 18:02:09+02:00",
			expected: "2020-04-10 10:00:00+00:00",
		},
		{
			now:      "2020-04-10 22:00:00-02:00",
			expected: "2020-04-13 05:00:00+02:00",
		},
		{
			now:      "2020-08-23 00:00:00+00:00",
			expected: "2020-08-24 05:00:00+02:00",
		},
		{
			now:      "2020-08-24 04:59:00+02:00",
			expected: "2020-08-24 05:00:00+02:00",
		},
		{
			now:      "2020-08-24 05:00:00+02:00",
			expected: "2020-08-24 05:00:00+01:00",
		},
	})
}

func TestScheduleNextMidnight(t *testing.T) {
	s := back.NewSchedule()
	s.Mon = []string{"00:00 UTC", "12:00 UTC"}

	testSchedule(t, s, []scheduleTestData{
		{
			now:      "2020-08-23 00:00:00+00:00",
			expected: "2020-08-24 00:00:00+00:00",
		},
	})
}

func TestScheduleStd(t *testing.T) {
	s := back.NewSchedule()
	s.SetAll([]string{
		"21:00 America/Los_Angeles",
		"21:00 America/New_York",
		"15:00 Europe/Paris",
		"21:00 Europe/Paris",
	})

	// -1h for the other days
	s.Tue = []string{"20:00 America/Los_Angeles", "20:00 America/New_York", "14:00 Europe/Paris", "20:00 Europe/Paris"}
	s.Thu = []string{"20:00 America/Los_Angeles", "20:00 America/New_York", "14:00 Europe/Paris", "20:00 Europe/Paris"}
	s.Sun = []string{"20:00 America/Los_Angeles", "20:00 America/New_York", "14:00 Europe/Paris", "20:00 Europe/Paris"}

	testSchedule(t, s, []scheduleTestData{
		{
			now:      "2020-04-15 10:00:00+00:00",
			expected: "2020-04-15 15:00:00+02:00",
		},
		{
			now:      "2020-04-15 00:00:00+02:00",
			expected: "2020-04-14 20:00:00-04:00",
		},
		{
			now:      "2020-04-15 01:00:00+00:00",
			expected: "2020-04-14 20:00:00-07:00",
		},
		{
			now:      "2020-05-29 08:00:00+00:00",
			expected: "2020-05-29 15:00:00+02:00",
		},
	})
}

func TestScheduleShu(t *testing.T) {
	s := back.NewSchedule()

	s.Mon = []string{"05:00 UTC", "21:00 Europe/Paris"}
	s.Tue = []string{"13:00 UTC", "22:00 UTC"}
	s.Wed = []string{"10:00 UTC", "14:00 UTC"}
	s.Thu = []string{"04:00 UTC", "20:00 Europe/Paris"}
	s.Fri = []string{"14:00 UTC", "23:00 UTC"}
	s.Sat = []string{"05:00 UTC", "10:00 UTC", "21:00 Europe/Paris", "23:00 UTC"}
	s.Sun = []string{"05:00 UTC", "10:00 UTC", "14:00 UTC", "21:00 Europe/Paris"}

	testSchedule(t, s, []scheduleTestData{
		{
			now:      "2020-09-05 20:30:00+02:00",
			expected: "2020-09-05 21:00:00+02:00",
		},
		{
			now:      "2020-09-05 22:00:00+02:00",
			expected: "2020-09-05 23:00:00+00:00",
		},
		{
			now:      "2020-09-05 23:00:00+00:00",
			expected: "2020-09-06 05:00:00+00:00",
		},
		{
			now:      "2020-09-06 06:00:00+00:00",
			expected: "2020-09-06 10:00:00+00:00",
		},
	})
}

type scheduleTestData struct {
	now      string
	expected string
}

func testSchedule(t *testing.T, s back.Schedule, tests []scheduleTestData) {
	t.Helper()

	format := "2006-01-02 15:04:05-07:00"
	for _, v := range tests {
		now, err := time.Parse(format, v.now)
		if err != nil {
			t.Fatal(err)
		}

		actual := s.NextBetween(now, now.AddDate(0, 0, 7)).Format(format)
		if actual != v.expected {
			t.Errorf("now: %s,\texpected %s, got %s", now, v.expected, actual)
		}
	}
}
