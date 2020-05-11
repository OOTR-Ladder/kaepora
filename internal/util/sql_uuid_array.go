package util

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// UUIDArrayAsJSON is stored as an UNIX timestamp but used as a time.Time.
type UUIDArrayAsJSON []uuid.UUID

func (a UUIDArrayAsJSON) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a UUIDArrayAsJSON) Slice() []uuid.UUID {
	return []uuid.UUID(a)
}

func (a *UUIDArrayAsJSON) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, &a)
	case string:
		return json.Unmarshal([]byte(src), &a)
	default:
		return fmt.Errorf("expected []byte or string, got %T", src)
	}
}

type SortByUUID []uuid.UUID

func (a SortByUUID) Len() int {
	return len([]uuid.UUID(a))
}

func (a SortByUUID) Less(i, j int) bool {
	return a[i].String() < a[j].String()
}

func (a SortByUUID) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
