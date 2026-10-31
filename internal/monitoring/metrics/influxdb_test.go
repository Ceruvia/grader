package metrics

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/pool"
)

// Metrics report the pool they are given. They must not read the global pool,
// which stays nil while isolate boxes are still initializing.
func TestMetricsPusherReportsGivenPoolWhileGlobalIsUnset(t *testing.T) {
	writes := make(chan string, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		select {
		case writes <- string(b):
		default:
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	previous := pool.Pool
	pool.Pool = nil
	t.Cleanup(func() { pool.Pool = previous })

	InitMetricPusher(&config.ServerConfig{MonitoringCfg: &config.MonitoringConfig{InfluxURL: srv.URL, InfluxOrganization: "o", InfluxBucket: "b"}})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunMetricsPusher(ctx, &pool.SandboxPool{})
	}()

	select {
	case point := <-writes:
		// An empty pool has no idle and no busy sandbox.
		if !strings.Contains(point, "idle_workers=0") || !strings.Contains(point, "busy_workers=0") {
			t.Errorf("want idle_workers=0 and busy_workers=0, got %q", point)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no metrics point pushed")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pusher did not stop after its context ended")
	}
}
