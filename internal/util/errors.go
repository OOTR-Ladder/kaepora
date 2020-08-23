package util

import (
	"errors"
	"strings"
)

// ErrPublic is an error we are allowed to show to the end-user.
type ErrPublic string

func (e ErrPublic) Error() string {
	return string(e)
}

// Is implements the error interface.
func (e ErrPublic) Is(v error) bool {
	_, ok := v.(ErrPublic)
	return ok
}

// ConcatErrors returns an error if errs contains a single non-nil error.
func ConcatErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err.Error())
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	return errors.New(strings.Join(filtered, "; "))
}
