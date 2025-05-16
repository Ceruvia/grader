package pylang_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages/pylang"
)

var (
	PythonGradingLanguage = pylang.Python3Language{}
)

func TestGetName(t *testing.T) {
	got := PythonGradingLanguage.GetName()
	want := "Python 3"
	tester.AssertDeep(t, got, want)
}

func TestGetAllowedExtention(t *testing.T) {
	got := PythonGradingLanguage.GetAllowedExtention()
	want := []string{"py"}
	tester.AssertDeep(t, got, want)
}

func TestGetCompilationCommand(t *testing.T) {

	got := PythonGradingLanguage.GetCompilationCommand("", "")
	want := "/bin/true "
	tester.AssertDeep(t, got.BuildFullCommand(), want)
}

func TestGetExecutionCommand(t *testing.T) {
	got := PythonGradingLanguage.GetExecutionCommand("main.py")
	want := "/usr/bin/python3 main.py"
	tester.AssertDeep(t, got.BuildFullCommand(), want)
}

func TestGetExecutableFilename(t *testing.T) {
	got := PythonGradingLanguage.GetExecutableFilename("main.py")
	want := "main.py"
	tester.AssertDeep(t, got, want)
}
