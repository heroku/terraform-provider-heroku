package heroku

import (
	"github.com/hashicorp/terraform/helper/logging"
	heroku "github.com/heroku/heroku-go/v3"
	"log"
	"net/http"
)

type Config struct {
	Email   string
	APIKey  string
	Headers http.Header
	URL     string

	Api *heroku.Service
}

// Client returns a new Config for accessing Heroku.
func (c *Config) loadAndInitialize() error {
	var debugHTTP = false
	if logging.IsDebugOrHigher() {
		debugHTTP = true
	}
	c.Api = heroku.NewService(&http.Client{
		Transport: &heroku.Transport{
			Username:          c.Email,
			Password:          c.APIKey,
			UserAgent:         heroku.DefaultUserAgent,
			AdditionalHeaders: c.Headers,
			Debug:             debugHTTP,
			Transport:         heroku.RoundTripWithRetryBackoff{
				// Configuration fields for ExponentialBackOff
				// InitialIntervalSeconds: 30,
				// RandomizationFactor:    0.25,
				// Multiplier:             2,
				// MaxIntervalSeconds:     900,
				// MaxElapsedTimeSeconds:  0,
			},
		},
	})

	c.Api.URL = c.URL

	log.Printf("[INFO] Heroku Client configured for user: %s", c.Email)

	return nil
}
