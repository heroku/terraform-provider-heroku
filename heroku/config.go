package heroku

import (
	"log"
	"net/http"
	"os"

	heroku "github.com/heroku/heroku-go/v3"
)

type Config struct {
	Email   string
	APIKey  string
	Headers http.Header
}

// Client returns a new Service for accessing Heroku.
func (c *Config) Client() (*heroku.Service, error) {
	var debugHTTP = false
	if os.Getenv("TF_LOG") == "TRACE" || os.Getenv("TF_LOG") == "DEBUG" {
		debugHTTP = true
	}
	service := heroku.NewService(&http.Client{
		Transport: &heroku.Transport{
			Username:          c.Email,
			Password:          c.APIKey,
			UserAgent:         heroku.DefaultUserAgent,
			AdditionalHeaders: c.Headers,
			Debug:             debugHTTP,
		},
	})

	log.Printf("[INFO] Heroku Client configured for user: %s", c.Email)

	return service, nil
}
