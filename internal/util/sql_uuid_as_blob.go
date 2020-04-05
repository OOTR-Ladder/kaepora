package util

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// UUIDAsBlob is stored as blob(16) but used as a uuid.UUID
type UUIDAsBlob uuid.UUID

func NewUUIDAsBlob() UUIDAsBlob {
	return UUIDAsBlob(uuid.New())
}

func (t UUIDAsBlob) Value() (driver.Value, error) {
	buf := [16]byte(t)
	return driver.Value(buf[:]), nil
}

func (t UUIDAsBlob) UUID() uuid.UUID {
	return uuid.UUID(t)
}

func (t *UUIDAsBlob) Scan(src interface{}) error {
	slice, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", src)
	}

	var buf [16]byte

	copy(buf[:], slice)
	*t = UUIDAsBlob(buf)

	return nil
}

type NullUUIDAsBlob struct {
	UUID  UUIDAsBlob
	Valid bool // Valid is true if UUIDAsBlob is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullUUIDAsBlob) Scan(value interface{}) error {
	if value == nil {
		ns.UUID, ns.Valid = UUIDAsBlob{}, false
		return nil
	}

	ns.Valid = true

	return ns.UUID.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullUUIDAsBlob) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}

	return ns.UUID, nil
}
