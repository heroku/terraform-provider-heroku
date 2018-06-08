package heroku

import (
	"log"
	"net/http"

	heroku "github.com/cyberdelia/heroku-go/v3"
)

type Config struct {
	Email   string
	APIKey  string
	Headers http.Header
}

// Client() returns a new Service for accessing Heroku.
//
func (c *Config) Client() (*heroku.Service, error) {
	service := heroku.NewService(&http.Client{
		Transport: &heroku.Transport{
			Username:          c.Email,
			Password:          c.APIKey,
			UserAgent:         heroku.DefaultUserAgent,
			AdditionalHeaders: c.Headers,
		},
	})

	log.Printf("[INFO] Heroku Client configured for user: %s", c.Email)

	return service, nil
}
