package isolate_test

import (
	"errors"
	"os"
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes/isolate"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestCreateIsolateSandbox(t *testing.T) {
	t.Run("it should succesfully create an Isolate sandbox", func(t *testing.T) {
		sbx, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", 990)
		defer sbx.Cleanup()

		want := isolate.IsolateSandbox{
			IsolatePath:   "/usr/local/bin/isolate",
			BoxId:         990,
			AllowedDirs:   []string{},
			Filenames:     []string{},
			FileSizeLimit: 100 * 1024,
			MaxProcesses:  50,
			BoxDir:        "/var/local/lib/isolate/990",
		}

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		utils.AssertDeep(t, sbx, want)
	})
}

func TestAddFile(t *testing.T) {
	t.Run("it should add the file to sbx.Filenames", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{
			Filenames: []string{},
			BoxDir:    "tests/fake/destination",
		}

		err := sbx.AddFile("tests/fake/source/file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		got := sbx.Filenames
		want := []string{"file.c"}

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should copy the file to sbx.Boxdir", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{
			Filenames: []string{},
			BoxDir:    "tests/fake/destination",
		}

		err := sbx.AddFile("tests/fake/source/file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("tests/fake/destination/file.c"); err != nil {
			t.Errorf("file was not moved to Boxdir: %q", err)
		}

		// cleanup
		os.Remove("tests/fake/destination/file.c")
	})

	t.Run("it should return error when file doesn't exist", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{
			Filenames: []string{},
			BoxDir:    "tests/fake/destination",
		}

		err := sbx.AddFile("tests/fake/source/gaada.c")

		if err == nil {
			t.Fatalf("didn't get an error when expecting: %q", err)
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf(`should've gotten "no such file or directory", instead got %q`, err)
		}
	})
}

func TestContainsFile(t *testing.T) {
	sbx := isolate.IsolateSandbox{
		Filenames: []string{"iexists.c"},
	}
	t.Run("it should return True when file is in sbx.Filenames", func(t *testing.T) {
		got := sbx.ContainsFile("iexists.c")
		want := true

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should return False when file doesn't exists in sbx.Filenames", func(t *testing.T) {
		got := sbx.ContainsFile("idontexists.c")
		want := false

		utils.AssertDeep(t, got, want)
	})
}

func TestGetFile(t *testing.T) {
	sbx := isolate.IsolateSandbox{
		BoxDir:    "tests/fake/source",
		Filenames: []string{"file.c"},
	}

	t.Run("it should be able to get a file", func(t *testing.T) {
		file, err := sbx.GetFile("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		data := make([]byte, 4)
		file.Read(data)

		got := string(data)
		want := "smth"

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should return error when not in sbx.Filenames", func(t *testing.T) {
		_, err := sbx.GetFile("nada.c")

		if err == nil {
			t.Fatalf("didn't get an error when expecting: %q", err)
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf(`should've gotten "no such file or directory", instead got %q`, err)
		}
	})
}
