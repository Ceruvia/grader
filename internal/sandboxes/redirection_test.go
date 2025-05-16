package sandboxes_test

import (
	"os"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestSetters(t *testing.T) {
	t.Run("it should be able to set meta file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/copy/source",
		}
		err := red.RedirectMeta("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:       "../../tests/copy/source",
			MetaFilename: "../../tests/copy/source/file.c",
		}

		tester.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard input file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/copy/source",
		}
		err := red.RedirectStandardInput("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "../../tests/copy/source",
			StandardInputFilename: "file.c",
		}

		tester.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard output file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/copy/source",
		}
		err := red.RedirectStandardOutput("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                 "../../tests/copy/source",
			StandardOutputFilename: "file.c",
		}

		tester.AssertDeep(t, red, want)
	})

	t.Run("it should be able to set standard error file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/copy/source",
		}
		err := red.RedirectStandardError("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "../../tests/copy/source",
			StandardErrorFilename: "file.c",
		}

		tester.AssertDeep(t, red, want)
	})

	t.Run("it should be able to reset all redirections", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			StandardInputFilename:  "1.in",
			StandardOutputFilename: "1.out.expected",
			StandardErrorFilename:  "1.err",
		}

		red.ResetRedirection()

		want := sandboxes.RedirectionFiles{}

		tester.AssertDeep(t, red, want)
	})
}

func TestCreation(t *testing.T) {
	t.Run("it should be able to create and redirect meta file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/sandbox",
		}
		err := red.CreateNewMetaFileAndRedirect("_isolate.meta")
		defer deleteFile("../../tests/sandbox/_isolate.meta")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("../../tests/sandbox/_isolate.meta"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:       "../../tests/sandbox",
			MetaFilename: "../../tests/sandbox/_isolate.meta",
		}

		tester.AssertDeep(t, red, want)
	})

	t.Run("it should be able to create and redirect standard input file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/sandbox",
		}
		err := red.CreateNewStandardInputFileAndRedirect("input.in")
		defer deleteFile("../../tests/sandbox/input.in")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("../../tests/sandbox/input.in"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "../../tests/sandbox",
			StandardInputFilename: "input.in",
		}

		tester.AssertDeep(t, red, want)
	})

	t.Run("it should be able to create and redirect standard output file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/sandbox",
		}
		err := red.CreateNewStandardOutputFileAndRedirect("output.out")
		defer deleteFile("../../tests/sandbox/output.out")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("../../tests/sandbox/output.out"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                 "../../tests/sandbox",
			StandardOutputFilename: "output.out",
		}

		tester.AssertDeep(t, red, want)
	})

	t.Run("it should be able to create and redirect standard error file", func(t *testing.T) {
		red := sandboxes.RedirectionFiles{
			Boxdir: "../../tests/sandbox",
		}
		err := red.CreateNewStandardErrorFileAndRedirect("error.err")
		defer deleteFile("../../tests/sandbox/error.err")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("../../tests/sandbox/error.err"); err != nil {
			t.Fatalf("file was not created: %q", err.Error())
		}

		want := sandboxes.RedirectionFiles{
			Boxdir:                "../../tests/sandbox",
			StandardErrorFilename: "error.err",
		}

		tester.AssertDeep(t, red, want)
	})
}

func deleteFile(filepath string) error {
	return os.Remove(filepath)
}
