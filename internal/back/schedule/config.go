package schedule

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Config struct {
	Type    Type
	Payload json.RawMessage
}

type Type string

const (
	TypeDayOfWeek Type = "day-of-week"
	TypeRolling   Type = "rolling"
)

func (c *Config) Scan(src interface{}) error {
	switch src := src.(type) {
	case string:
		return json.Unmarshal([]byte(src), c)
	case []byte:
		return json.Unmarshal(src, c)
	default:
		return fmt.Errorf("expected []byte or string, got %T", src)
	}
}

func (c Config) Value() (driver.Value, error) {
	str, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return driver.Value(str), nil
}
