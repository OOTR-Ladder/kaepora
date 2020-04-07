package util

import (
	"errors"
	"strings"
)

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
