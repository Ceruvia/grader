package sandboxes_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestSetters(t *testing.T) {
	t.Run("it should be able to set meta file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{}
		err := red.RedirectMeta("isolate/tests/fake/source", "file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			MetaFilename: "isolate/tests/fake/source/file.c",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard input file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{}
		err := red.RedirectStandardInput("isolate/tests/fake/source", "file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			StandardInputFilename: "isolate/tests/fake/source/file.c",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard output file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{}
		err := red.RedirectStandardOutput("isolate/tests/fake/source", "file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			StandardOutputFilename: "isolate/tests/fake/source/file.c",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard error file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{}
		err := red.RedirectStandardError("isolate/tests/fake/source", "file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			StandardErrorFilename: "isolate/tests/fake/source/file.c",
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
