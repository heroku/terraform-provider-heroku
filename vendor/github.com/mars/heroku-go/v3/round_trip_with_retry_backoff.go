package heroku

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
)

// net/http RoundTripper interface, a.k.a. Transport
// https://godoc.org/net/http#RoundTripper
type RoundTripWithRetryBackoff struct{}

func (_ RoundTripWithRetryBackoff) RoundTrip(req *http.Request) (*http.Response, error) {
	var lastResponse *http.Response
	var lastError error

	retryableRoundTrip := func() error {
		lastResponse, lastError = http.DefaultTransport.RoundTrip(req)
		// Detect Heroku API rate limiting
		// https://devcenter.heroku.com/articles/platform-api-reference#client-error-responses
		if lastResponse.StatusCode == 429 {
			return fmt.Errorf("Heroku API rate limited: 429 Too Many Requests")
		}
		return nil
	}

	rateLimitRetryConfig := &backoff.ExponentialBackOff{
		Clock:               backoff.SystemClock,
		InitialInterval:     30 * time.Second,
		RandomizationFactor: 0.25,
		Multiplier:          2,
		MaxInterval:         15 * time.Minute,
		// After MaxElapsedTime the ExponentialBackOff stops.
		// It never stops if MaxElapsedTime == 0.
		MaxElapsedTime: 0,
	}
	rateLimitRetryConfig.Reset()

	err := backoff.RetryNotify(retryableRoundTrip, rateLimitRetryConfig, notifyLog)
	// Propagate the rate limit error when retries eventually fail.
	if err != nil {
		if lastResponse != nil {
			lastResponse.Body.Close()
		}
		return nil, err
	}
	// Propagate all other response errors.
	if lastError != nil {
		if lastResponse != nil {
			lastResponse.Body.Close()
		}
		return nil, lastError
	}

	return lastResponse, nil
}

func notifyLog(err error, waitDuration time.Duration) {
	log.Printf("Will retry Heroku API request in %s, because %s", waitDuration, err)
}
