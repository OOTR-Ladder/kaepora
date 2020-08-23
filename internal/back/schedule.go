package back

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// A Schedule is a per-weekday set hour local times at which sessions can occur.
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

// SetAll sets the same to of hours to all weekdays (the slice is not copied).
func (s *Schedule) SetAll(hours []string) {
	s.Mon = hours
	s.Tue = hours
	s.Wed = hours
	s.Thu = hours
	s.Fri = hours
	s.Sat = hours
	s.Sun = hours
}

// NextBetween returns the first event occurring between two point in time or a
// zero time if none is found.
func (s *Schedule) NextBetween(t time.Time, max time.Time) time.Time {
	t, max = t.UTC(), max.UTC()
	if t.After(max) {
		return time.Time{}
	}

	for _, next := range s.flattenHours(t) {
		if next.Before(t) || next.Sub(t) == 0 {
			continue
		}

		if next.After(max) {
			return time.Time{}
		}

		return next
	}

	// No match, attempt starting at next day.
	return s.NextBetween(t.AddDate(0, 0, 1), max)
}

// flattenHours get all hours in the week of the given time.
func (s *Schedule) flattenHours(t time.Time) []time.Time {
	ret := make([]time.Time, 0, s.totalHoursCount())

	start := t.AddDate(0, 0, -int(t.Weekday()))
	for dow := 0; dow < 7; dow++ {
		day := start.AddDate(0, 0, dow)

		for _, rawHour := range s.hoursForDow(dow) {
			hourStr, location, err := hourLocation(rawHour)
			if err != nil {
				log.Printf("error: %s", err)
				continue
			}

			hour, err := time.ParseInLocation("15:04", hourStr, location)
			if err != nil {
				log.Printf("error: %s", err)
				continue
			}

			ret = append(ret, time.Date(
				day.Year(), day.Month(), day.Day(),
				hour.Hour(), hour.Minute(),
				0, 0, location,
			))
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Before(ret[j])
	})

	return ret
}

// totalHoursCount returns the number of hours set across all days.
func (s *Schedule) totalHoursCount() int {
	return len(s.Mon) + len(s.Tue) + len(s.Wed) + len(s.Thu) + len(s.Fri) +
		len(s.Sat) + len(s.Sun)
}

func hourLocation(str string) (string, *time.Location, error) {
	parts := strings.SplitN(str, " ", 2)
	if len(parts) < 2 {
		return "", nil, fmt.Errorf("bad format: '%s'", str)
	}

	location, err := time.LoadLocation(parts[1])
	if err != nil {
		return "", nil, fmt.Errorf("bad location '%s': %v", parts[1], err)
	}

	return parts[0], location, nil
}

// Next returns the next scheduled date in a week span.
func (s *Schedule) Next() time.Time {
	now := time.Now()
	return s.NextBetween(now, now.AddDate(0, 0, 7))
}

func (s *Schedule) hoursForDow(dow int) []string {
	return [][]string{s.Sun, s.Mon, s.Tue, s.Wed, s.Thu, s.Fri, s.Sat}[dow]
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
