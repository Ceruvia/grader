package sandboxes_test

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

// Isolate 2.1 deletes symlinks and FIFOs from the box after each run. These tests plant
// them from the host so GetFile stays safe without relying on that cleanup.
func TestGetFileReadsOnlyRegularBoxFiles(t *testing.T) {
	for _, test := range []struct {
		name    string
		plant   func(t *testing.T, path, hostFile string)
		wantErr bool
	}{
		{"regular file", func(t *testing.T, path, _ string) {
			if err := os.WriteFile(path, []byte("main.c:1: error\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}, false},
		{"symlink to host file", func(t *testing.T, path, hostFile string) {
			if err := os.Symlink(hostFile, path); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"fifo", func(t *testing.T, path, _ string) {
			if err := syscall.Mkfifo(path, 0666); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"directory", func(t *testing.T, path, _ string) {
			if err := os.Mkdir(path, 0700); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"larger than the box file size limit", func(t *testing.T, path, _ string) {
			if err := os.WriteFile(path, bytes.Repeat([]byte("x"), 2048), 0600); err != nil {
				t.Fatal(err)
			}
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			hostFile := filepath.Join(dir, "host-env")
			if err := os.WriteFile(hostFile, []byte("BACKEND_API_TOKEN=host-file-canary\n"), 0600); err != nil {
				t.Fatal(err)
			}
			boxDir := filepath.Join(dir, "box")
			if err := os.Mkdir(boxDir, 0700); err != nil {
				t.Fatal(err)
			}
			test.plant(t, filepath.Join(boxDir, "_compile.out"), hostFile)
			// FileSizeLimit is in KB, like isolate -f, so the cap is 1024 bytes.
			sbx := &sandboxes.IsolateSandbox{BoxDir: boxDir, FileSizeLimit: 1}
			sbx.AddFileWithoutMove("_compile.out")

			type read struct {
				data []byte
				err  error
			}
			done := make(chan read, 1)
			go func() {
				data, err := sbx.GetFile("_compile.out")
				done <- read{data, err}
			}()
			var got read
			select {
			case got = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("GetFile blocked")
			}
			if bytes.Contains(got.data, []byte("canary")) {
				t.Fatalf("GetFile returned host file content %q", got.data)
			}
			if (got.err != nil) != test.wantErr {
				t.Fatalf("expected error %v, got %v (data %q)", test.wantErr, got.err, got.data)
			}
			if !test.wantErr && string(got.data) != "main.c:1: error\n" {
				t.Fatalf("GetFile = %q", got.data)
			}
		})
	}
}
