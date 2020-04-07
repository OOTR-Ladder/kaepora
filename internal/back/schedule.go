package back

import (
	"database/sql/driver"
	"encoding/json"
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

	for _, v := range s.hoursForWeekday(t.Format("Mon")) {
		parts := strings.SplitN(v, " ", 2)
		if len(parts) < 2 {
			continue // HACK, silently ignore misconfiguration
		}

		location, _ := time.LoadLocation(parts[1])
		hour, _ := time.ParseInLocation("15:04", parts[0], location)
		next := time.Date(t.Year(), t.Month(), t.Day(), hour.Hour(), hour.Minute(), 0, 0, location)

		if next.After(t) {
			return next
		}
	}

	// Found nothing, roll over to next day at midnight until max is reached
	next := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
	return s.NextBetween(next, max)
}

// Returns the next scheduled date in a week span
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
	return json.Unmarshal(src.([]byte), s)
}

func (s Schedule) Value() (driver.Value, error) {
	str, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return driver.Value(str), nil
}
