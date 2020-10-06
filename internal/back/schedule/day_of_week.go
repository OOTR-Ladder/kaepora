package schedule

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// A DayOfWeekScheduler scheduler is a per-weekday set hour local times at which
// sessions can occur.
type DayOfWeekScheduler struct {
	Mon []string
	Tue []string
	Wed []string
	Thu []string
	Fri []string
	Sat []string
	Sun []string
}

func NewDayOfWeekScheduler() DayOfWeekScheduler {
	return DayOfWeekScheduler{ // ensure we store "[]" and not "null"
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
// For debugging purposes.
func (s *DayOfWeekScheduler) SetAll(hours []string) {
	s.Mon = hours
	s.Tue = hours
	s.Wed = hours
	s.Thu = hours
	s.Fri = hours
	s.Sat = hours
	s.Sun = hours
}

func (s *DayOfWeekScheduler) NextBetween(t time.Time, max time.Time) time.Time {
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
	nextDay := time.Date(
		t.Year(), t.Month(), t.Day(),
		0, 0, 0, 0, t.Location(),
	).AddDate(0, 0, 1)
	return s.NextBetween(nextDay, max)
}

// flattenHours get all hours in the week of the given time.
func (s *DayOfWeekScheduler) flattenHours(t time.Time) []time.Time {
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
func (s *DayOfWeekScheduler) totalHoursCount() int {
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

func (s *DayOfWeekScheduler) Next() time.Time {
	now := time.Now()
	return s.NextBetween(now, now.AddDate(0, 0, 7))
}

func (s *DayOfWeekScheduler) hoursForDow(dow int) []string {
	return [][]string{s.Sun, s.Mon, s.Tue, s.Wed, s.Thu, s.Fri, s.Sat}[dow]
}
