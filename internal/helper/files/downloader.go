package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// MaxDownloadBytes caps one grader or submission archive.
const MaxDownloadBytes = 64 << 20

var ErrDownloadTooLarge = errors.New("download exceeds 64 MiB")

var downloadClient = &http.Client{Timeout: 30 * time.Second}

// DownloadFile saves url to filepath. It fails on a non-2xx status, after 30 s,
// when ctx ends, or when the body exceeds MaxDownloadBytes. On any error no file is left behind.
func DownloadFile(ctx context.Context, filepath, url string) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(filepath)
		}
	}()

	n, err := io.Copy(out, io.LimitReader(resp.Body, MaxDownloadBytes+1))
	if err != nil {
		return err
	}
	if n > MaxDownloadBytes {
		return ErrDownloadTooLarge
	}
	return nil
}
