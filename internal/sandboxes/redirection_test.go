package sandboxes_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestSetters(t *testing.T) {
	t.Run("it should be able to set meta file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake/source",
		}
		err := red.RedirectMeta("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:       "isolate/tests/fake/source",
			MetaFilename: "isolate/tests/fake/source/file.c",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard input file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake/source",
		}
		err := red.RedirectStandardInput("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "isolate/tests/fake/source",
			StandardInputFilename: "file.c",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard output file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake/source",
		}
		err := red.RedirectStandardOutput("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                 "isolate/tests/fake/source",
			StandardOutputFilename: "file.c",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard error file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake/source",
		}
		err := red.RedirectStandardError("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "isolate/tests/fake/source",
			StandardErrorFilename: "file.c",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to reset all redirections", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			StandardInputFilename:  "1.in",
			StandardOutputFilename: "1.out.expected",
			StandardErrorFilename:  "1.err",
		}

		red.ResetRedirection()

		want := sandboxes.RedirectionFiles{}

		utils.AssertDeep(t, red, want)
	})
}
