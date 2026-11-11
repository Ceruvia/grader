package machinery

import (
	"testing"
	"time"
)

// A consumer that never reports stopped (a stop during Machinery's retry
// sleep) cannot hold the stop past shutdownGrace plus the release grace.
func TestWaitStoppedIsBounded(t *testing.T) {
	savedShutdown, savedRelease := shutdownGrace, releaseGrace
	shutdownGrace, releaseGrace = 100*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() { shutdownGrace, releaseGrace = savedShutdown, savedRelease })

	s := newStopState()
	returned := make(chan struct{})
	go func() { s.waitStopped(); close(returned) }()
	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("waitStopped still blocked on a consumer that never stopped")
	}
	if s.released.Err() == nil {
		t.Fatal("waitStopped returned without releasing the running jobs")
	}
}
