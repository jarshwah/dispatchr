package dispatchr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestDispatcher_RetryAfter(t *testing.T) {
	now := time.Date(2024, 3, 14, 15, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		retryAfter    string
		expectedDelay time.Duration
		shouldRetry   bool
	}{
		{
			name:          "seconds delay",
			retryAfter:    "30",
			expectedDelay: 30 * time.Second,
			shouldRetry:   true,
		},
		{
			name:          "http date delay",
			retryAfter:    now.Add(1 * time.Minute).Format("Mon, 02 Jan 2006 15:04:05 GMT"),
			expectedDelay: 1 * time.Minute,
			shouldRetry:   true,
		},
		{
			name:          "invalid format",
			retryAfter:    "invalid",
			expectedDelay: 0,
			shouldRetry:   true,
		},
		{
			name:          "empty header",
			retryAfter:    "",
			expectedDelay: 0,
			shouldRetry:   true,
		},
		{
			name:          "negative seconds",
			retryAfter:    "-30",
			expectedDelay: 0,
			shouldRetry:   true,
		},
		{
			name:          "past date",
			retryAfter:    now.Add(-1 * time.Minute).Format("Mon, 02 Jan 2006 15:04:05 GMT"),
			expectedDelay: 0,
			shouldRetry:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Retry-After header value: %q", tt.retryAfter)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", tt.retryAfter)
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			defer server.Close()

			targetURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatalf("Failed to parse server URL: %v", err)
			}

			d := NewDispatcher()
			d.nowFn = func() time.Time { return now }
			result := d.Dispatch(context.Background(), targetURL, "test-task", []byte("{}"))

			if got := result.RetryAfter; !approximatelyEqual(got, tt.expectedDelay) {
				t.Errorf("RetryAfter = %v, want %v", got, tt.expectedDelay)
			}

			if got := result.ShouldRetry; got != tt.shouldRetry {
				t.Errorf("ShouldRetry = %v, want %v", got, tt.shouldRetry)
			}

			if result.Success {
				t.Error("Success = true, want false")
			}

			expectedMsg := fmt.Sprintf("rate limited with retry after %v", tt.expectedDelay)
			if tt.expectedDelay == 0 {
				expectedMsg = "rate limited: 429"
			}
			if got := result.ErrorMessage; got != expectedMsg {
				t.Errorf("ErrorMessage = %q, want %q", got, expectedMsg)
			}
		})
	}
}

// approximatelyEqual handles small timing differences that might occur with HTTP date parsing
func approximatelyEqual(a, b time.Duration) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < time.Second
}
