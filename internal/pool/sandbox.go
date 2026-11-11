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

	// A crash leaves boxes behind, maybe with student programs still running.
	// Kill them all first, so none runs on while the boxes before it are
	// cleaned. A failed kill is retried by CleanBox below.
	for i := 1; i <= size; i++ {
		if err := sandboxes.KillBoxProcesses(i); err != nil {
			log.WithField("sandboxId", i).WithError(err).Error("Stale sandbox kill failed")
		}
	}
	ch := make(chan sandboxSlot, size)
	for i := 1; i <= size; i++ {
		// Clean every box before it serves a job. One that cannot be cleaned
		// stays quarantined until Acquire retries it.
		if err := sandboxes.CleanBox(isolatePath, i); err != nil {
			log.WithField("sandboxId", i).WithError(err).Error("Stale sandbox cleanup failed, quarantining box")
			ch <- sandboxSlot{sb: &sandboxes.IsolateSandbox{IsolatePath: isolatePath, BoxId: i}, cleanupRequired: true}
			continue
		}
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
		// CleanBox, not Cleanup: a box quarantined at start was never
		// initialized by this process, and a live process may still hold it.
		if err := sandboxes.CleanBox(slot.sb.IsolatePath, slot.sb.BoxId); err != nil {
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
