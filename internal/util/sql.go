package util

import (
	"gopkg.in/guregu/null.v4"
)

// NullString is a shorthand to create a valid null.String or an invalid one if
// the given string is empty.
func NullString(v string) null.String {
	return null.NewString(v, v != "")
}
