package builder_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages/builder"
)

var (
	MakefileBuilder = builder.MakefileBuilder{}
)

func TestGetName(t *testing.T) {
	got := MakefileBuilder.GetName()
	want := "Makefile"
	tester.AssertDeep(t, got, want)
}

func TestGetAllowedExtention(t *testing.T) {
	got := MakefileBuilder.GetAllowedExtention()
	want := []string{}
	tester.AssertDeep(t, got, want)
}

func TestGetCompilationCommand(t *testing.T) {
	Tests := []struct {
		compileScript string
		want          string
	}{
		{"", "/usr/bin/make "},
		{"all", "/usr/bin/make all"},
		{"compile INPUT_FILE=queuelist.c", "/usr/bin/make compile INPUT_FILE=queuelist.c"},
	}

	for _, test := range Tests {
		t.Run("it should be able to give the compilation command", func(t *testing.T) {
			got := MakefileBuilder.GetCompilationCommand(test.compileScript, []string{}...)
			tester.AssertDeep(t, got.BuildFullCommand(), test.want)
		})
	}
}

func TestGetExecutionCommand(t *testing.T) {
	got := MakefileBuilder.GetExecutionCommand("prog")
	want := "./prog "
	tester.AssertDeep(t, got.BuildFullCommand(), want)
}

func TestGetExecutableFilename(t *testing.T) {
	got := MakefileBuilder.GetExecutableFilename("Makefile")
	want := "Makefile"
	tester.AssertDeep(t, got, want)
}
