package http

import (
	"context"
	"math/rand"
	"net/http"
	"strings"
	"time"
)
import "github.com/hashicorp/go-retryablehttp"

//NewThrottlingAwareClient Wraps given client and handles retries on HTTP 429.
func NewThrottlingAwareClient(httpClient *http.Client, requestLogger func(format string, args ...interface{})) *http.Client {
	client := retryablehttp.NewClient()
	client.HTTPClient = httpClient
	client.Logger = debugLogger{inner: requestLogger}

	client.RetryMax = 15
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		shouldRetry := resp == nil || resp.StatusCode == 429

		return shouldRetry, nil
	}
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler
	client.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		if resp == nil {
			requestLogger("Error while parsing retry-after header: response is nil!")
			return 0
		}

		retryAfter, err := time.Parse(time.RFC1123, resp.Header.Get("retry-after"))
		if err != nil {
			requestLogger("Error while parsing retry-after header: %v", err)
			return 0
		}

		// Add random 5-10s delay to reduce the contention
		retryAfter = retryAfter.Add(time.Second * time.Duration(5+rand.Intn(5)))

		var duration time.Duration = 0

		now := time.Now()
		if retryAfter.After(now) {
			duration = retryAfter.Sub(now)
		}

		return duration
	}

	return client.StandardClient()
}

type debugLogger struct {
	inner func(format string, args ...interface{})
}

func (l debugLogger) Printf(format string, args ...interface{}) {
	// Fix weird format of inner logging...
	format = strings.ReplaceAll(format, "[DEBUG] ", "")
	format = strings.ReplaceAll(format, "%s", "%v")
	l.inner(format, args...)
}
