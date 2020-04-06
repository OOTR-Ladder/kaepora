package util

import (
	"database/sql"
)

// NullString is a shorthand to create a valid sql.NullString or an invalid one
// if the given string is empty.
func NullString(v string) sql.NullString {
	return sql.NullString{
		String: v,
		Valid:  v != "",
	}
}
