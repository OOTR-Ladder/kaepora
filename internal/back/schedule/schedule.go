package schedule

import (
	"encoding/json"
	"fmt"
	"time"
)

type Scheduler interface {
	// Next returns the next scheduled date in a week span or a zero time is
	// none is found.
	Next() time.Time

	// NextBetween returns the first event occurring between two point in time
	// or a zero time if none is found.
	NextBetween(start, end time.Time) time.Time
}

func New(conf Config) (Scheduler, error) {
	switch conf.Type {
	case TypeDayOfWeek:
		s := NewDayOfWeekScheduler()
		if err := json.Unmarshal(conf.Payload, &s); err != nil {
			return nil, err
		}
		return &s, nil

	case TypeRolling:
		s := NewRollingScheduler()
		if err := json.Unmarshal(conf.Payload, &s); err != nil {
			return nil, err
		}
		return &s, nil
	}

	return &VoidScheduler{}, fmt.Errorf("invalid scheduler type '%s'", conf.Type)
}

type VoidScheduler struct{}

func (s *VoidScheduler) Next() time.Time {
	return time.Time{}
}

func (s *VoidScheduler) NextBetween(start, end time.Time) time.Time {
	return time.Time{}
}
