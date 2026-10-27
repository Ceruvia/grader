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

func TestSandboxBoundsAggregateDisk(t *testing.T) {
	for i, script := range []string{
		"chunk = b'x' * (1024 * 1024)\ntry:\n for name in ['a', 'b', 'c']:\n  with open(name, 'wb') as f:\n   for _ in range(90): f.write(chunk)\nexcept OSError as exc:\n if exc.errno == 28: raise SystemExit(42)\n raise",
		"try:\n for i in range(10001): open(str(i), 'w').close()\nexcept OSError as exc:\n if exc.errno == 28: raise SystemExit(42)\n raise",
	} {
		t.Run([]string{"bytes", "entries"}[i], func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 832+i)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := sbx.Cleanup(); err != nil {
					t.Error(err)
				}
			})
			sbx.SetWallTimeLimitInMiliseconds(10000)
			red := sandboxes.CreateRedirectionFiles(sbx)
			if err := red.CreateNewMetaFileAndRedirect("disk.meta"); err != nil {
				t.Fatal(err)
			}
			got := sbx.Execute(*command.GetCommandBuilder("/usr/bin/python3").AddArgs("-c", script), red)
			if got.Status != sandboxes.NONZERO_EXIT_CODE || got.ExitCode != 42 {
				t.Fatalf("kernel aggregate limit must return ENOSPC (exit 42), got %+v", got)
			}
		})
	}
}
