package utils

import (
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
