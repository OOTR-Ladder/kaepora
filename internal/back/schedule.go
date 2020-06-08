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
	t, max = t.UTC(), max.UTC()
	if t.After(max) {
		return time.Time{}
	}

	min := max
	for location, perLoc := range s.prepareTzMap() {
		local := t.In(location)
		dow := local.Format("Mon")

		for _, nextHourStr := range perLoc[dow] {
			nextHour, err := time.ParseInLocation("15:04", nextHourStr, location)
			if err != nil {
				log.Printf("error: ignored schedule, can't parse time '%s': %v", nextHourStr, err)
				continue // HACK, silently ignore misconfiguration
			}

			localNext := time.Date(
				local.Year(), local.Month(), local.Day(),
				nextHour.Hour(), nextHour.Minute(),
				0, 0, location,
			)

			if localNext.After(t) && localNext.Before(min) {
				min = localNext
			}
		}
	}

	if min != max {
		return min
	}

	// Found nothing, roll over to next day at midnight until max is reached
	next := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
	return s.NextBetween(next, max)
}

// prepareTzMap parses the per-day schedule into a per-location map.
func (s *Schedule) prepareTzMap() map[*time.Location]map[string][]string {
	// tzMap[location][dow] => 15:04
	tzMap := map[*time.Location]map[string][]string{}

	for _, wd := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"} {
		for _, v := range s.hoursForWeekday(wd) {
			parts := strings.SplitN(v, " ", 2)
			if len(parts) < 2 {
				log.Printf("error: ignored schedule, bad format: '%s'", v)
				continue // HACK, silently ignore misconfiguration
			}

			location, err := time.LoadLocation(parts[1])
			if err != nil {
				log.Printf("error: ignored schedule, bad location '%s': %v", parts[1], err)
				continue // HACK, silently ignore misconfiguration
			}

			if _, ok := tzMap[location]; !ok {
				tzMap[location] = map[string][]string{}
			}

			tzMap[location][wd] = append(tzMap[location][wd], parts[0])
			sort.Strings(tzMap[location][wd])
		}
	}

	return tzMap
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
