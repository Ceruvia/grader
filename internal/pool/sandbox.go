package pool

import (
	"errors"

	"github.com/Ceruvia/grader/internal/sandboxes"
	log "github.com/sirupsen/logrus"
)

var Pool *SandboxPool

type sandboxSlot struct {
	sb              *sandboxes.IsolateSandbox
	initialized     bool
	cleanupRequired bool
}

type SandboxPool struct {
	pool chan sandboxSlot
}

func NewSandboxPool(isolatePath string, size int) error {
	if size <= 0 {
		return errors.New("pool size must be > 0")
	}

	ch := make(chan sandboxSlot, size)
	for i := 1; i <= size; i++ {
		sb, err := sandboxes.CreateIsolateSandbox(isolatePath, i)
		if err != nil {
			close(ch)
			for slot := range ch {
				err = errors.Join(err, slot.sb.Cleanup())
			}
			return err
		}
		ch <- sandboxSlot{sb: sb, initialized: true}
	}

	Pool = &SandboxPool{pool: ch}
	return nil
}

// Acquire blocks until one sandbox is available, then returns it.
// While a sandbox is "checked out," no other goroutine can acquire that same instance.
func (p *SandboxPool) Acquire() (*sandboxes.IsolateSandbox, error) {
	return p.initialize(<-p.pool)
}

// TryAcquire does not wait for a lease. A quarantined slot is returned only
// after cleanup and re-init, and recovery failure is treated as unavailable.
func (p *SandboxPool) TryAcquire() (*sandboxes.IsolateSandbox, bool) {
	select {
	case slot := <-p.pool:
		sb, err := p.initialize(slot)
		return sb, err == nil
	default:
		return nil, false
	}
}

// Release quarantines the id until cleanup and reinitialization have succeeded.
func (p *SandboxPool) Release(sb *sandboxes.IsolateSandbox) {
	err := sb.Cleanup()
	if err != nil {
		log.WithField("sandboxId", sb.BoxId).WithError(err).Error("Sandbox cleanup failed, quarantining box")
	}
	p.pool <- sandboxSlot{sb: sb, cleanupRequired: err != nil}
}

func (p *SandboxPool) initialize(slot sandboxSlot) (*sandboxes.IsolateSandbox, error) {
	if slot.initialized {
		return slot.sb, nil
	}
	if slot.cleanupRequired {
		if err := slot.sb.Cleanup(); err != nil {
			p.pool <- slot
			return nil, err
		}
		slot.cleanupRequired = false
	}
	sb, err := sandboxes.CreateIsolateSandbox(slot.sb.IsolatePath, slot.sb.BoxId)
	if err != nil {
		// A failed --init can leave isolate's lock marked uninitialized, which
		// --cleanup rejects. Retry --init itself: it owns recovery of partial
		// initialization. Usable boxes with failed cleanup still take the
		// mandatory Cleanup path above before any initialization is attempted.
		p.pool <- slot
		return nil, err
	}
	return sb, nil
}

func (p *SandboxPool) Close() {
	close(p.pool)
	for sb := range p.pool {
		if err := sb.sb.Cleanup(); err != nil {
			log.WithField("sandboxId", sb.sb.BoxId).WithError(err).Error("Sandbox cleanup failed at pool close")
		}
	}
}

func (p *SandboxPool) IdleCount() int {
	return len(p.pool)
}

func (p *SandboxPool) BusyCount() int {
	return cap(p.pool) - len(p.pool)
}
