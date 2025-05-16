package javalang_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages/javalang"
)

var (
	JavaGradingLanguage = javalang.JavaLanguage{}
)

func TestGetName(t *testing.T) {
	got := JavaGradingLanguage.GetName()
	want := "Java"
	tester.AssertDeep(t, got, want)
}

func TestGetAllowedExtention(t *testing.T) {
	got := JavaGradingLanguage.GetAllowedExtention()
	want := []string{"java"}
	tester.AssertDeep(t, got, want)
}

func TestGetCompilationCommand(t *testing.T) {
	Tests := []struct {
		mainSourceFilename string
		sourceFilenames    []string
		want               string
	}{
		{"Main.java", []string{"Main.java", "Gajah.java", "Bebek.java", "Animal.java"}, `/bin/bash -c /usr/bin/javac "Main.java" "Gajah.java" "Bebek.java" "Animal.java" && /usr/bin/jar cfe "Main.jar" "Main" *.class`},
		{"Grader.java", []string{"Grader.java", "Gajah.java", "Bebek.java", "Animal.java"}, `/bin/bash -c /usr/bin/javac "Grader.java" "Gajah.java" "Bebek.java" "Animal.java" && /usr/bin/jar cfe "Grader.jar" "Grader" *.class`},
	}

	for _, test := range Tests {
		t.Run("it should be able to give the compilation command", func(t *testing.T) {
			got := JavaGradingLanguage.GetCompilationCommand(test.mainSourceFilename, test.sourceFilenames...)
			tester.AssertDeep(t, got.BuildFullCommand(), test.want)
		})
	}
}

func TestGetExecutionCommand(t *testing.T) {
	got := JavaGradingLanguage.GetExecutionCommand("Main.java")
	want := "/usr/bin/java -jar Main.jar"
	tester.AssertDeep(t, got.BuildFullCommand(), want)
}

func TestGetExecutableFilename(t *testing.T) {
	got := JavaGradingLanguage.GetExecutableFilename("Main.java")
	want := "Main.jar"
	tester.AssertDeep(t, got, want)
}
