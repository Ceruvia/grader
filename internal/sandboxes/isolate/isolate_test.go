package isolate_test

import (
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

		sbx.AddFile("tests/fake/source/file.c")

		got := sbx.Filenames
		want := []string{"file.c"}

		utils.AssertDeep(t, got, want)
	})
}
