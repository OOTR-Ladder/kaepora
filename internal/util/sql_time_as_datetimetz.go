package util

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// TimeAsDateTimeTZ is stored as an UNIX timestamp but used as a time.Time
type TimeAsDateTimeTZ time.Time

func (t TimeAsDateTimeTZ) Value() (driver.Value, error) {
	return driver.Value(time.Time(t).Format(time.RFC3339)), nil
}

func (t TimeAsDateTimeTZ) Time() time.Time {
	return time.Time(t)
}

func (t TimeAsDateTimeTZ) String() string {
	return t.Time().String()
}

func (t *TimeAsDateTimeTZ) Scan(src interface{}) error {
	var str string
	switch src := src.(type) {
	case []byte:
		str = string(src)
	case string:
		str = src
	default:
		return fmt.Errorf("expected []byte or string, got %T", src)
	}

	tmp, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	*t = TimeAsDateTimeTZ(tmp)
	return nil
}

type NullTimeAsDateTimeTZ struct {
	Time  TimeAsDateTimeTZ
	Valid bool // Valid is true if TimeAsDateTimeTZ is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullTimeAsDateTimeTZ) Scan(value interface{}) error {
	if value == nil {
		ns.Time, ns.Valid = TimeAsDateTimeTZ{}, false
		return nil
	}

	ns.Valid = true

	return ns.Time.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullTimeAsDateTimeTZ) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}

	return ns.Time.Value()
}
