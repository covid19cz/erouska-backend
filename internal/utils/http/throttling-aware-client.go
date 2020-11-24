package http

import (
	"context"
	"fmt"
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

	client.RetryMax = 5
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		return resp.StatusCode == 429, nil
	}
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler
	client.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		retryAfter, err := time.Parse(time.RFC1123, resp.Header.Get("retry-after"))
		if err != nil {
			fmt.Printf("Error while parsing retry-after header: %v", err)
			return 0
		}

		// This is needed because the 'Retry-After' header is rounded to whole second so we need to prolong the waiting period to prevent
		// premature retry.
		retryAfter = retryAfter.Add(time.Millisecond * 750)

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
