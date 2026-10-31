//go:build linux

package tasks

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/pool"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestPoolRecoversAfterProcessRestart(t *testing.T) {
	if mode := os.Getenv("CERUVIA_POOL_RESTART_HELPER"); mode != "" {
		if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
			t.Fatal(err)
		}
		defer pool.Pool.Close()
		sbx, err := pool.Pool.Acquire()
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Pool.Release(sbx)
		canary := filepath.Join(sbx.GetBoxdir(), "previous-process")
		if mode == "initial" {
			if err := os.WriteFile(canary, []byte("stale student state"), 0600); err != nil {
				t.Fatal(err)
			}
			fmt.Println("READY " + sbx.GetBoxdir())
			var input [1]byte
			_, _ = os.Stdin.Read(input[:]) // parent kills this process without running cleanup
			return
		}
		if _, err := os.Lstat(canary); !os.IsNotExist(err) {
			t.Fatalf("stale file survived process restart: %v", err)
		}
		red := sandboxes.CreateRedirectionFiles(sbx)
		if err := red.CreateNewMetaFileAndRedirect("restart.meta"); err != nil {
			t.Fatal(err)
		}
		if result := sbx.Execute(*command.GetCommandBuilder("/bin/true"), red); result.Status != sandboxes.ZERO_EXIT_CODE {
			t.Fatalf("replacement process cannot execute: %+v", result)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	first := exec.CommandContext(ctx, binary, "-test.run=^TestPoolRecoversAfterProcessRestart$")
	first.Env = append(os.Environ(), "CERUVIA_POOL_RESTART_HELPER=initial")
	stdin, err := first.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := first.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	first.Stderr = os.Stderr
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	defer first.Process.Kill()
	ready, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || !strings.HasPrefix(ready, "READY ") {
		t.Fatalf("initial process did not initialize: %q, %v", ready, err)
	}
	boxDir := strings.TrimSpace(strings.TrimPrefix(ready, "READY "))
	t.Cleanup(func() { repairTestBox(boxDir) })
	if err := first.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := first.Wait(); err == nil {
		t.Fatal("expected abrupt process termination")
	}
	second := exec.CommandContext(ctx, binary, "-test.run=^TestPoolRecoversAfterProcessRestart$")
	second.Env = append(os.Environ(), "CERUVIA_POOL_RESTART_HELPER=restarted")
	if output, err := second.CombinedOutput(); err != nil {
		t.Fatalf("replacement grader process failed: %v\n%s", err, output)
	}
}

func TestPoolRecoversActualPartialInitialization(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	sbx, ok := pool.Pool.TryAcquire()
	if !ok {
		t.Fatal("no sandbox")
	}
	boxDir := sbx.GetBoxdir()
	pool.Pool.Release(sbx)
	fault := filepath.Join(filepath.Dir(boxDir), "block-initialization")
	if err := os.MkdirAll(fault, 0755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("mount", "-t", "tmpfs", "-o", "size=1m", "tmpfs", fault).CombinedOutput(); err != nil {
		t.Fatalf("inject filesystem fault: %v %s", err, output)
	}
	t.Cleanup(func() {
		_ = exec.Command("umount", fault).Run()
		repairTestBox(boxDir)
	})
	if unexpected, ok := pool.Pool.TryAcquire(); ok {
		pool.Pool.Release(unexpected)
		t.Fatal("isolate initialization unexpectedly succeeded through a nested mount")
	}
	output, err := exec.Command("/usr/local/bin/isolate", "-b", "1", "--cg", "--cleanup").CombinedOutput()
	if err == nil || !strings.Contains(string(output), "not initialized properly") {
		t.Fatalf("fault did not create a real partial isolate lock: %v %s", err, output)
	}
	if output, err := exec.Command("umount", fault).CombinedOutput(); err != nil {
		t.Fatalf("clear filesystem fault: %v %s", err, output)
	}
	recovered, ok := pool.Pool.TryAcquire()
	if !ok {
		t.Fatal("pool remains quarantined after the real initialization fault cleared")
	}
	defer pool.Pool.Release(recovered)
}

// Restore actual isolate state even after a deliberately broken regression run.
func repairTestBox(boxDir string) {
	id := filepath.Base(filepath.Dir(boxDir))
	_ = exec.Command("umount", boxDir).Run()
	_ = exec.Command("/usr/local/bin/isolate", "-b", id, "--cg", "--init").Run()
	_ = exec.Command("/usr/local/bin/isolate", "-b", id, "--cg", "--cleanup").Run()
}
