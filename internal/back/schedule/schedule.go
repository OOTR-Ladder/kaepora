package schedule

import (
	"encoding/json"
	"log"
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

func New(conf Config) Scheduler { // TODO, error instead of panic
	switch conf.Type {
	case TypeDayOfWeek:
		s := NewDayOfWeekScheduler()
		if err := json.Unmarshal(conf.Payload, &s); err != nil {
			panic(err)
		}
		return &s
	case TypeRolling:
		s := NewRollingScheduler()
		if err := json.Unmarshal(conf.Payload, &s); err != nil {
			panic(err)
		}
		return &s
	}

	log.Printf("warning: invalid scheduler type '%s'", conf.Type)
	return &VoidScheduler{}
}

type VoidScheduler struct{}

func (s *VoidScheduler) Next() time.Time {
	return time.Time{}
}

func (s *VoidScheduler) NextBetween(start, end time.Time) time.Time {
	return time.Time{}
}
