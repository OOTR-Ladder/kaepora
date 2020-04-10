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

	tests := []struct {
		now      string
		expected string
	}{
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
	}

	format := "2006-01-02 15:04:05-07:00"
	for _, v := range tests {
		now, err := time.Parse(format, v.now)
		if err != nil {
			t.Fatal(err)
		}

		actual := s.NextBetween(now, now.AddDate(0, 0, 7)).Format(format)
		if actual != v.expected {
			t.Errorf("expected %s, got %s", v.expected, actual)
		}
	}
}
