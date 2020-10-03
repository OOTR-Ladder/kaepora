package schedule

import (
	"log"
	"math"
	"time"
)

// The RollingScheduler schedules dates based on the number of elapsed periods.
type RollingScheduler struct {
	// Period between races, and offset to set the 0h race.
	Period, Offset time.Duration

	Modulus    int64
	Remainders []int64 // when the remainders of (period number % Modulus) is in this list, the date is chosen.
}

func NewRollingScheduler() RollingScheduler {
	return RollingScheduler{}
}

func (s *RollingScheduler) NextBetween(t time.Time, max time.Time) time.Time {
	period := int64(math.Floor(s.Period.Seconds()))
	if period <= 0 {
		log.Printf("error: period is too short: %v", s.Period)
		return time.Time{}
	}

	periodID := t.Unix() / period

	for {
		periodID++
		next := time.Unix(period*periodID, 0).Add(s.Offset).UTC()

		if next.Before(t) {
			continue
		}

		if next.After(max) {
			return time.Time{}
		}

		for _, v := range s.Remainders {
			if periodID%s.Modulus == v {
				return next
			}
		}
	}
}

func (s *RollingScheduler) Next() time.Time {
	now := time.Now()
	return s.NextBetween(now, now.AddDate(0, 0, 7))
}
