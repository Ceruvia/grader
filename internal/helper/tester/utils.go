package tester

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func AssertDeep[T any](t testing.TB, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, when expected %+v", got, want)
	}
}

func AssertError(t testing.TB, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("got error %q, when expected %q", got, want)
	}
}

func AssertIncludesError(t testing.TB, got error, wantToContain string) {
	t.Helper()
	if !strings.Contains(got.Error(), wantToContain) {
		t.Errorf("got error %q, when expected to contain %q", got, wantToContain)
	}
}

func AssertCustomError(t testing.TB, got, want error) {
	t.Helper()
	if got.Error() != want.Error() {
		t.Errorf("got error %q, when expected %q", got, want)
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
