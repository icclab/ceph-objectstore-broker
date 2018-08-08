package testutils

import (
	"fmt"
	"testing"
)

func CheckErrs(t *testing.T, args []interface{}, fatal bool, errors ...error) func(t *testing.T) {
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
			if fatal {
				t.Fatal(args, failed)
			} else {
				t.Error(args, failed)
			}
		}
	}
}

func Equals(expected interface{}, actual interface{}, msg string) error {
	if expected != actual {
		return fmt.Errorf("%s. Expected: '%+v', got: '%+v'", msg, expected, actual)
	}
	return nil
}
