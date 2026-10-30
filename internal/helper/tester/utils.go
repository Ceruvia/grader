package tester

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
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

// GraderEnvPath is the env dump that dump-env.service writes and grader-docker.service reads.
const GraderEnvPath = "/run/grader/env"

// PlantGraderEnv replaces the real grader env dump with content for the rest of the test and
// restores it afterwards. go test runs packages as parallel processes that share this file,
// so an exclusive flock is held from the write until the restore is done.
func PlantGraderEnv(t testing.TB, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(GraderEnvPath), 0700); err != nil {
		t.Fatal(err)
	}
	lock, err := os.OpenFile(GraderEnvPath+".test-lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		_ = lock.Close()
		t.Fatal(err)
	}
	// Cleanups run last-in first-out: this one runs after the restore below.
	t.Cleanup(func() { _ = lock.Close() })
	original, originalErr := os.ReadFile(GraderEnvPath)
	if originalErr != nil && !os.IsNotExist(originalErr) {
		t.Fatal(originalErr)
	}
	t.Cleanup(func() {
		// Another writer during the test means the canary check may have probed the wrong content.
		if planted, err := os.ReadFile(GraderEnvPath); err != nil || string(planted) != content {
			t.Errorf("%s changed during the test: %q, %v", GraderEnvPath, planted, err)
		}
		if originalErr == nil {
			if err := os.WriteFile(GraderEnvPath, original, 0600); err != nil {
				t.Error(err)
			}
		} else if err := os.Remove(GraderEnvPath); err != nil {
			t.Error(err)
		}
	})
	if err := os.WriteFile(GraderEnvPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
