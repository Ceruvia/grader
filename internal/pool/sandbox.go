package pool

import (
	"errors"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

var Pool *SandboxPool

type SandboxPool struct {
	pool chan *sandboxes.IsolateSandbox
}

func NewSandboxPool(isolatePath string, size int) error {
	if size <= 0 {
		return errors.New("pool size must be > 0")
	}

	ch := make(chan *sandboxes.IsolateSandbox, size)
	for i := 1; i <= size; i++ {
		sb, err := sandboxes.CreateIsolateSandbox(isolatePath, i)
		if err != nil {
			return err
		}
		ch <- sb
	}

	Pool = &SandboxPool{pool: ch}
	return nil
}

// Acquire blocks until one sandbox is available, then returns it.
// While a sandbox is "checked out," no other goroutine can acquire that same instance.
func (p *SandboxPool) Acquire() *sandboxes.IsolateSandbox {
	return <-p.pool
}

// TryAcquire attempts to get a sandbox without blocking. If none is available,
// it returns (nil, false).
func (p *SandboxPool) TryAcquire() (*sandboxes.IsolateSandbox, bool) {
	select {
	case sb := <-p.pool:
		return sb, true
	default:
		return nil, false
	}
}

func (p *SandboxPool) Release(sb *sandboxes.IsolateSandbox) {
	p.pool <- sb
}

func (p *SandboxPool) Close() {
	close(p.pool)
	for sb := range p.pool {
		sb.Cleanup()
	}
}

func (p *SandboxPool) IdleCount() int {
	return len(p.pool)
}

func (p *SandboxPool) BusyCount() int {
	return cap(p.pool) - len(p.pool)
}