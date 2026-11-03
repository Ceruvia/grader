package machinery

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

var testBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond}

func callbackServer(t *testing.T, statuses ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(requests.Add(1))
		if r.Header.Get("X-API-Key") != "token" || r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		status := statuses[len(statuses)-1]
		if n <= len(statuses) {
			status = statuses[n-1]
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func TestPostCallback(t *testing.T) {
	body := []byte(`{"task_id":"task-1"}`)
	for _, test := range []struct {
		name     string
		statuses []int
		wantErr  bool
		requests int32
	}{
		{"retries 5xx until success", []int{503, 503, 200}, false, 3},
		{"gives up after four attempts", []int{503}, true, 4},
		{"retries rate limit", []int{429, 204}, false, 2},
		{"stops on client error", []int{400}, true, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, requests := callbackServer(t, test.statuses...)
			err := postCallback(context.Background(), server.Client(), server.URL, "token", body, testBackoff)
			if (err != nil) != test.wantErr {
				t.Fatalf("expected error %v, got %v", test.wantErr, err)
			}
			if got := requests.Load(); got != test.requests {
				t.Fatalf("expected %d requests, got %d", test.requests, got)
			}
		})
	}

	t.Run("closed port returns an error", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		url := "http://" + listener.Addr().String()
		if err := listener.Close(); err != nil {
			t.Fatal(err)
		}
		if err := postCallback(context.Background(), &http.Client{Timeout: time.Second}, url, "token", body, testBackoff); err == nil {
			t.Fatal("expected an error for a closed port")
		}
	})

	t.Run("cancelled context stops retries", func(t *testing.T) {
		server, requests := callbackServer(t, 503)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := postCallback(ctx, server.Client(), server.URL, "token", body, []time.Duration{time.Hour}); err == nil {
			t.Fatal("expected an error for a cancelled context")
		}
		if got := requests.Load(); got > 1 {
			t.Fatalf("expected at most 1 request, got %d", got)
		}
	})
}
