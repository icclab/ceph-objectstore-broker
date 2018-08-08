package testutils

import (
	"fmt"
	"testing"
)

func CheckErrs(t *testing.T, args []interface{}, errors ...error) func(t *testing.T) {
	return func(t *testing.T) {
		//Get non-nil errors
		var failed []error
		gotError := false
		for _, err := range errors {
			if err != nil {
				failed = append(failed, err)
				gotError = true
			}
		}

		if gotError {
			t.Error(args, failed)
		}
	}
}

func Equals(expected interface{}, actual interface{}, msg string) error {
	if expected != actual {
		return fmt.Errorf("%s. Expected: '%+v', got: '%+v'", msg, expected, actual)
	}
	return nil
}

func Atleast(min float64, actual float64, msg string) error {
	if min > actual {
		return fmt.Errorf("%s. Expected at least: '%+v', got: '%+v'", msg, min, actual)
	}
	return nil
}
