package testutils

import (
	"errors"
	"testing"
)

func Check(pass bool, msg string) error {
	if !pass {
		return errors.New(msg)
	}
	return nil
}

func CheckErr(t *testing.T, e error, fatal bool, args ...interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		if e != nil {
			if fatal {
				t.Fatal(args, e)
			} else {
				t.Error(args, e)
			}
		}
	}
}

func CheckErrs(t *testing.T, e []error, fatal bool, args ...interface{}) func(t *testing.T) {
	return func(t *testing.T) {

		var failed []error
		for _, err := range e {
			if err != nil {
				failed = append(failed, err)
			}
		}

		if len(failed) != 0 {
			if fatal {
				t.Fatal(args, failed)
			} else {
				t.Error(args, failed)
			}
		}
	}
}
