package httputil

import (
	"fmt"
	"net/http"
	"time"
)

// DefaultClient is a shared http.Client with a 30s timeout.
var DefaultClient = &http.Client{Timeout: 30 * time.Second}

// Do performs an HTTP request with exponential backoff retry.
// Retries on: network errors, 429, 5xx responses.
// Max 3 attempts: delays 1s, 2s before retries.
func Do(req *http.Request) (*http.Response, error) {
	delays := []time.Duration{1 * time.Second, 2 * time.Second}
	var lastErr error
	for attempt := range 3 {
		resp, err := DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < 2 {
				time.Sleep(delays[attempt])
			}
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, req.URL)
			if attempt < 2 {
				time.Sleep(delays[attempt])
			}
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("request failed after 3 attempts: %w", lastErr)
}
