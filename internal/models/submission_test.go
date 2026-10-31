package models_test

import (
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/models"
)

// A builder run target is the artifact the makefile leaves, run as named. A
// Java target names the main class, and the Java makefile leaves <class>.jar.
func TestBuilderRunTargetExecutionCommand(t *testing.T) {
	tests := []struct {
		language, runScript, want string
	}{
		{"Java", "Main", "/usr/bin/java -jar Main.jar"},
		{"C", "program.out", "./program.out"},
		{"C", "prog", "./prog"},
		{"Haskell", "Main", "./Main"},
	}
	for _, tt := range tests {
		s := models.SubmissionWithBuilder{Core: models.Core{Language: tt.language}, Builder: "Makefile", RunScript: tt.runScript}
		cmd := s.GetLanguage().GetExecutionCommand(s.GetExecFilenameOrScript())
		got := strings.TrimSpace(cmd.BuildFullCommand())
		if got != tt.want {
			t.Errorf("%s run target %q: want %q, got %q", tt.language, tt.runScript, tt.want, got)
		}
	}
}
