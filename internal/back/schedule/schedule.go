package schedule

import (
	"encoding/json"
	"log"
	"time"
)

type Scheduler interface {
	Next() time.Time
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
