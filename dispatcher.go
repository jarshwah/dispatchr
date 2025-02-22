package dispatchr

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type DispatchResult struct {
	Success      bool
	ShouldRetry  bool
	RetryAfter   time.Duration
	ErrorMessage string
}

type Dispatcher struct {
	httpClient *http.Client
	nowFn      func() time.Time // for testing
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext,
			},
		},
		nowFn: time.Now,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, target *url.URL, taskName string, payload []byte) DispatchResult {
	req, err := d.prepareRequest(ctx, target, taskName, payload)
	if err != nil {
		return DispatchResult{
			Success:      false,
			ShouldRetry:  true,
			ErrorMessage: fmt.Sprintf("failed to create request: %v", err),
		}
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return DispatchResult{
			Success:      false,
			ShouldRetry:  true,
			ErrorMessage: fmt.Sprintf("failed to execute request: %v", err),
		}
	}
	defer resp.Body.Close()

	return d.handleResponse(resp)
}

func (d *Dispatcher) prepareRequest(ctx context.Context, target *url.URL, taskName string, payload []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", target.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Task-Name", taskName)
	return req, nil
}

func (d *Dispatcher) handleResponse(resp *http.Response) DispatchResult {
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return DispatchResult{Success: true}

	case resp.StatusCode == 429:
		if duration := parseRetryAfter(resp.Header.Get("Retry-After"), d.nowFn()); duration > 0 {
			return DispatchResult{
				Success:      false,
				ShouldRetry:  true,
				RetryAfter:   duration,
				ErrorMessage: fmt.Sprintf("rate limited with retry after %v", duration),
			}
		}
		return DispatchResult{
			Success:      false,
			ShouldRetry:  true,
			ErrorMessage: fmt.Sprintf("rate limited: %d", resp.StatusCode),
		}

	case resp.StatusCode >= 400 && resp.StatusCode < 600:
		return DispatchResult{
			Success:      false,
			ShouldRetry:  true,
			ErrorMessage: fmt.Sprintf("request failed with status: %d", resp.StatusCode),
		}

	default:
		return DispatchResult{
			Success:      false,
			ShouldRetry:  false,
			ErrorMessage: fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
		}
	}
}

func parseRetryAfter(retryAfter string, now time.Time) time.Duration {
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds first
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(retryAfter); err == nil {
		// Convert both times to UTC before comparing
		if seconds := int(t.UTC().Sub(now.UTC()).Seconds()); seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}

	return 0
}
