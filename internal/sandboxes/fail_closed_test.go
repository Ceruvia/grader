package sandboxes_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestSandboxRejectsLaunchFailureWithStaleMeta(t *testing.T) {
	meta := filepath.Join(t.TempDir(), "run.meta")
	if err := os.WriteFile(meta, []byte("time:0.001\ntime-wall:0.001\nexitcode:0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, executable := range []string{filepath.Join(t.TempDir(), "missing-isolate"), "/bin/false"} {
		t.Run(filepath.Base(executable), func(t *testing.T) {
			sbx := &sandboxes.IsolateSandbox{IsolatePath: executable, BoxId: 831}
			got := sbx.Execute(*command.GetCommandBuilder("/bin/true"), sandboxes.RedirectionFiles{MetaFilename: meta})
			if got.Status != sandboxes.INTERNAL_ERROR {
				t.Fatalf("launch failure must be internal error, got %+v", got)
			}
		})
	}
}

func TestParserRejectsIncompleteMeta(t *testing.T) {
	for _, content := range []string{"", "message:incomplete\n", "time:0.001\n", "exitcode:0\n", "time:bad\ntime-wall:0.001\nexitcode:0\n", "time:0.001\ntime-wall:0.001\nexitcode:1\n", "status:unknown\n"} {
		t.Run(content, func(t *testing.T) {
			meta := filepath.Join(t.TempDir(), "run.meta")
			if err := os.WriteFile(meta, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			if got, err := sandboxes.ParseMetaResult(meta); err == nil {
				t.Fatalf("incomplete/invalid meta must fail closed, got %+v", got)
			}
		})
	}
}
