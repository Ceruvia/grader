package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/pool"
)

func TestPoolQuarantinesFailedCleanup(t *testing.T) {
	const isolate = "/usr/local/bin/isolate"
	if err := pool.NewSandboxPool(isolate, 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	sbx, ok := pool.Pool.TryAcquire()
	if !ok {
		t.Fatal("no initial sandbox")
	}
	canary := filepath.Join(sbx.GetBoxdir(), "stale-file")
	if err := os.WriteFile(canary, []byte("previous student"), 0600); err != nil {
		t.Fatal(err)
	}
	sbx.IsolatePath = filepath.Join(t.TempDir(), "missing-isolate")
	pool.Pool.Release(sbx)
	if unsafe, ok := pool.Pool.TryAcquire(); ok {
		unsafe.IsolatePath = isolate
		pool.Pool.Release(unsafe)
		t.Fatal("pool reused a box after failed cleanup without a successful re-init")
	}
	// Restoring the executable simulates recovery from the process-launch failure.
	sbx.IsolatePath = isolate
	recovered, ok := pool.Pool.TryAcquire()
	if !ok {
		t.Fatal("pool did not recover after isolate returned")
	}
	defer pool.Pool.Release(recovered)
	if _, err := os.Lstat(canary); !os.IsNotExist(err) {
		t.Fatalf("previous student's file survived recovery: %v", err)
	}
}
