package machinery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// callbackClient bounds every callback attempt. The backend waits up to 8 s for the
// machinery result, so 10 s leaves room without hanging a worker.
var callbackClient = &http.Client{Timeout: 10 * time.Second}

// callbackBackoff is the wait before each retry, so a callback makes up to 4 attempts.
var callbackBackoff = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

// postCallback POSTs body to url. It retries on transport errors, 5xx and 429,
// stops on any other non-2xx status, and stops when ctx ends.
func postCallback(ctx context.Context, client *http.Client, url, token string, body []byte, backoff []time.Duration) error {
	var lastErr error
	for attempt := 0; attempt <= len(backoff); attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(backoff[attempt-1])
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("callback cancelled after %d attempts: %w (last error: %v)", attempt, ctx.Err(), lastErr)
			case <-timer.C:
			}
		}

		retry, err := sendCallback(ctx, client, url, token, body)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retry {
			return err
		}
	}
	return fmt.Errorf("callback failed after %d attempts: %w", len(backoff)+1, lastErr)
}

// sendCallback makes one attempt and reports whether a failure is worth retrying.
func sendCallback(ctx context.Context, client *http.Client, url, token string, body []byte) (retry bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("build callback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", token)

	res, err := client.Do(req)
	if err != nil {
		return ctx.Err() == nil, fmt.Errorf("send callback: %w", err)
	}
	defer res.Body.Close()
	// Drain a bounded amount so the connection can be reused.
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))

	switch {
	case res.StatusCode >= 200 && res.StatusCode < 300:
		return false, nil
	case res.StatusCode >= 500 || res.StatusCode == http.StatusTooManyRequests:
		return true, fmt.Errorf("callback status %d", res.StatusCode)
	default:
		return false, fmt.Errorf("callback status %d", res.StatusCode)
	}
}
