package sandboxes_test

import (
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestSandboxBoundsTemporaryScratch(t *testing.T) {
	for i, tc := range []struct{ name, setup, body string }{
		{"tmp bytes", "paths = ['/tmp/a', '/tmp/b', '/tmp/c']", "for name in paths:\n with open(name, 'wb') as f:\n  for _ in range(90): f.write(chunk)"},
		{"tmp entries", "", "for i in range(10001): open('/tmp/file' + str(i), 'w').close()"},
		{"shared bytes", "paths = ['/box/a', '/box/b', '/tmp/c']", "for name in paths:\n with open(name, 'wb') as f:\n  for _ in range(90): f.write(chunk)"},
		{"shared entries", "", "for i in range(5001):\n open('/box/box-file' + str(i), 'w').close()\n open('/tmp/tmp-file' + str(i), 'w').close()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 834+i)
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
			if err := red.CreateNewMetaFileAndRedirect("scratch.meta"); err != nil {
				t.Fatal(err)
			}
			script := "chunk = b'x' * (1024 * 1024)\n" + tc.setup + "\ntry:\n"
			script += " " + strings.ReplaceAll(tc.body, "\n", "\n ") + "\nexcept OSError as exc:\n if exc.errno == 28: raise SystemExit(42)\n raise\n"
			got := sbx.Execute(*command.GetCommandBuilder("/usr/bin/python3").AddArgs("-c", script), red)
			if got.Status != sandboxes.NONZERO_EXIT_CODE || got.ExitCode != 42 {
				t.Fatalf("scratch must share the kernel byte/inode bound (ENOSPC exit 42), got %+v", got)
			}
		})
	}
}
