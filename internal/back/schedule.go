package back

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

type Schedule struct {
	Mon []string
	Tue []string
	Wed []string
	Thu []string
	Fri []string
	Sat []string
	Sun []string
}

func NewSchedule() Schedule {
	return Schedule{ // ensure we store "[]" and not "null"
		Mon: []string{},
		Tue: []string{},
		Wed: []string{},
		Thu: []string{},
		Fri: []string{},
		Sat: []string{},
		Sun: []string{},
	}
}

func (s *Schedule) SetAll(hours []string) {
	s.Mon = hours
	s.Tue = hours
	s.Wed = hours
	s.Thu = hours
	s.Fri = hours
	s.Sat = hours
	s.Sun = hours
}

func (s *Schedule) NextBetween(t time.Time, max time.Time) time.Time {
	if t.After(max) {
		return time.Time{}
	}

	min := max
	for _, v := range s.hoursForWeekday(t.Format("Mon")) {
		parts := strings.SplitN(v, " ", 2)
		if len(parts) < 2 {
			log.Printf("error: ignored schedule, bad format: '%s'", v)
			continue // HACK, silently ignore misconfiguration
		}

		location, err := time.LoadLocation(parts[1])
		if err != nil {
			log.Printf("error: ignored schedule, bad location: '%s'", parts[1])
			continue // HACK, silently ignore misconfiguration
		}

		hour, _ := time.ParseInLocation("15:04", parts[0], location)
		next := time.Date(t.Year(), t.Month(), t.Day(), hour.Hour(), hour.Minute(), 0, 0, location)

		if next.After(t) && next.Before(min) {
			min = next
		}
	}

	// As schedule data may not be ordered, we have to ensure we take the
	// lowest date, not the first match.
	if min != max {
		return min
	}

	// Found nothing, roll over to next day at midnight until max is reached
	next := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
	return s.NextBetween(next, max)
}

// Returns the next scheduled date in a week span.
func (s *Schedule) Next() time.Time {
	return s.NextBetween(time.Now(), time.Now().AddDate(0, 0, 7))
}

func (s *Schedule) hoursForWeekday(day string) []string {
	switch day {
	case "Mon":
		return s.Mon
	case "Tue":
		return s.Tue
	case "Wed":
		return s.Wed
	case "Thu":
		return s.Thu
	case "Fri":
		return s.Fri
	case "Sat":
		return s.Sat
	case "Sun":
		return s.Sun
	default:
		panic("invalid day: " + day)
	}
}

func (s *Schedule) Scan(src interface{}) error {
	switch src := src.(type) {
	case string:
		return json.Unmarshal([]byte(src), s)
	case []byte:
		return json.Unmarshal(src, s)
	default:
		return fmt.Errorf("expected []byte or string, got %T", src)
	}
}

func (s Schedule) Value() (driver.Value, error) {
	str, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return driver.Value(str), nil
}
