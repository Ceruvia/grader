package utils

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func AssertDeep[T any](t testing.TB, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got error %+v, when expected %+v", got, want)
	}
}

func AssertError(t testing.TB, got, want error) {
	t.Helper()
	if got != want {
		t.Errorf("got error %+v, when expected %+v", got, want)
	}
}

func AssertNotError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("got an error when expected none: %+v", err)
	}
}

func AssertFileCreated(t testing.TB, path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
