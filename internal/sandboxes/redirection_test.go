package sandboxes_test

import (
	"os"
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

func TestCreation(t *testing.T) {
	t.Run("it should be able to create and redirect meta file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake",
		}
		err := red.CreateNewMetaFileAndRedirect("_isolate.meta")
		defer deleteFile("isolate/tests/fake/_isolate.meta")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("isolate/tests/fake/_isolate.meta"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:       "isolate/tests/fake",
			MetaFilename: "isolate/tests/fake/_isolate.meta",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to create and redirect standard input file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake",
		}
		err := red.CreateNewStandardInputFileAndRedirect("input.in")
		defer deleteFile("isolate/tests/fake/input.in")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("isolate/tests/fake/input.in"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "isolate/tests/fake",
			StandardInputFilename: "input.in",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to create and redirect standard output file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake",
		}
		err := red.CreateNewStandardOutputFileAndRedirect("output.out")
		defer deleteFile("isolate/tests/fake/output.out")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("isolate/tests/fake/output.out"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                 "isolate/tests/fake",
			StandardOutputFilename: "output.out",
		}

		utils.AssertDeep(t, red, want)
	})

	t.Run("it should be able to create and redirect standard error file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "isolate/tests/fake",
		}
		err := red.CreateNewStandardErrorFileAndRedirect("error.err")
		defer deleteFile("isolate/tests/fake/error.err")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("isolate/tests/fake/error.err"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "isolate/tests/fake",
			StandardErrorFilename: "error.err",
		}

		utils.AssertDeep(t, red, want)
	})
}

func deleteFile(filepath string) error {
	return os.Remove(filepath)
}
