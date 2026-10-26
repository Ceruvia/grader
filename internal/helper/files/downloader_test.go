package files_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ceruvia/grader/internal/helper/files"
)

func assertNoFile(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("expected no file at %s, stat error = %v", path, err)
	}
}

func TestDownloadFile(t *testing.T) {
	t.Run("hanging server stops at the context deadline", func(t *testing.T) {
		for _, test := range []struct {
			name        string
			sendHeaders bool
		}{
			{"before headers", false},
			{"during body", true},
		} {
			t.Run(test.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if test.sendHeaders {
						_, _ = w.Write([]byte("partial"))
						w.(http.Flusher).Flush()
					}
					<-r.Context().Done()
				}))
				t.Cleanup(server.Close)
				path := filepath.Join(t.TempDir(), "submission.zip")
				ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				defer cancel()

				start := time.Now()
				err := files.DownloadFile(ctx, path, server.URL)
				if err == nil {
					t.Fatal("expected an error from a hanging server")
				}
				if elapsed := time.Since(start); elapsed > time.Second {
					t.Fatalf("download returned after %s, want under 1s", elapsed)
				}
				assertNoFile(t, path)
			})
		}
	})

	t.Run("body over 64 MiB is rejected", func(t *testing.T) {
		chunk := make([]byte, 1<<20)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for range 65 {
				if _, err := w.Write(chunk); err != nil {
					return
				}
			}
		}))
		t.Cleanup(server.Close)
		path := filepath.Join(t.TempDir(), "submission.zip")
		if err := files.DownloadFile(context.Background(), path, server.URL); err == nil {
			t.Fatal("expected an error for a 65 MiB body")
		}
		assertNoFile(t, path)
	})

	t.Run("error status leaves no file", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		t.Cleanup(server.Close)
		path := filepath.Join(t.TempDir(), "submission.zip")
		if err := files.DownloadFile(context.Background(), path, server.URL); err == nil {
			t.Fatal("expected an error for 404")
		}
		assertNoFile(t, path)
	})

	t.Run("body at the cap is saved", func(t *testing.T) {
		want := bytes.Repeat([]byte{'x'}, 64<<20)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(want)
		}))
		t.Cleanup(server.Close)
		path := filepath.Join(t.TempDir(), "submission.zip")
		if err := files.DownloadFile(context.Background(), path, server.URL); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("saved %d bytes, error = %v, want %d bytes", len(got), err, len(want))
		}
	})
}
