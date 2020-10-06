package schedule_test

import (
	"encoding/json"
	"kaepora/internal/back/schedule"
	"testing"
	"time"
)

func TestRollingScheduler(t *testing.T) {
	conf, err := json.Marshal(schedule.RollingScheduler{
		Period:     4 * time.Hour,
		Modulus:    4,
		Remainders: []int64{1, 3},
	})
	if err != nil {
		t.Fatal(err)
	}

	s := schedule.New(schedule.Config{
		Type:    schedule.TypeRolling,
		Payload: conf,
	})

	testScheduler(t, s, []scheduleTestData{
		{
			now:      "2020-10-03 09:00:00+00:00",
			expected: "2020-10-03 12:00:00+00:00",
		},
		{
			now:      "2020-10-03 12:00:00+00:00",
			expected: "2020-10-03 20:00:00+00:00",
		},
		{
			now:      "2020-10-03 13:00:00+00:00",
			expected: "2020-10-03 20:00:00+00:00",
		},
		{
			now:      "2020-10-03 21:00:00+00:00",
			expected: "2020-10-04 04:00:00+00:00",
		},
	})
}

func TestRollingSchedulerEmptyPeriod(t *testing.T) {
	s := schedule.RollingScheduler{
		Period:     0,
		Modulus:    4,
		Remainders: []int64{1, 3},
	}

	if actual := s.Next(); actual != (time.Time{}) {
		t.Error("expected zero time.Time")
	}
}
