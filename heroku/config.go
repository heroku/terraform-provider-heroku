package heroku

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bgentry/go-netrc/netrc"
	heroku "github.com/heroku/heroku-go/v6"
	"github.com/heroku/terraform-provider-heroku/v5/version"
	homedir "github.com/mitchellh/go-homedir"
)

const (
	DefaultPostAppCreateDelay    = int64(10)
	DefaultPostSpaceCreateDelay  = int64(5)
	DefaultPostDomainCreateDelay = int64(5)

	// Default custom timeouts
	DefaultAddonCreateTimeout         = int64(20)
	DefaultSetAddonConfigVarsInState  = true
	DefaultSetAppAllConfigVarsInState = true
)

type Config struct {
	Api       *heroku.Service
	APIKey    string
	DebugHTTP bool
	Email     string
	Headers   http.Header
	URL       string

	// Delays
	PostAppCreateDelay    int64
	PostDomainCreateDelay int64
	PostSpaceCreateDelay  int64

	// Timeouts
	AddonCreateTimeout int64

	// Customization
	SetAddonConfigVarsInState  bool
	SetAppAllConfigVarsInState bool
}

func (c Config) String() string {
	return fmt.Sprintf("{APIKey:xxx Email:%s URL:%s Headers:xxx DebugHTTP:%t PostAppCreateDelay:%d PostDomainCreateDelay:%d PostSpaceCreateDelay:%d}",
		c.Email, c.URL, c.DebugHTTP, c.PostAppCreateDelay, c.PostDomainCreateDelay, c.PostSpaceCreateDelay)
}

func NewConfig() *Config {
	config := &Config{
		Headers:                    make(http.Header),
		PostAppCreateDelay:         DefaultPostAppCreateDelay,
		PostDomainCreateDelay:      DefaultPostDomainCreateDelay,
		PostSpaceCreateDelay:       DefaultPostSpaceCreateDelay,
		AddonCreateTimeout:         DefaultAddonCreateTimeout,
		SetAddonConfigVarsInState:  DefaultSetAddonConfigVarsInState,
		SetAppAllConfigVarsInState: DefaultSetAppAllConfigVarsInState,
	}
	if isDebugOrHigher() {
		config.DebugHTTP = true
	}
	return config
}

// isDebugOrHigher reports whether the TF_LOG environment variable requests
// DEBUG- or TRACE-level logging. It replaces the SDKv2
// helper/logging.IsDebugOrHigher used before the migration to
// terraform-plugin-framework.
func isDebugOrHigher() bool {
	level := strings.ToUpper(os.Getenv("TF_LOG"))
	return level == "DEBUG" || level == "TRACE"
}

func (c *Config) initializeAPI() (err error) {
	c.Api = heroku.NewService(&http.Client{
		Transport: &heroku.Transport{
			Username: c.Email,
			Password: c.APIKey,
			UserAgent: fmt.Sprintf("%s terraform-provider-heroku/%s",
				heroku.DefaultUserAgent, version.ProviderVersion),
			AdditionalHeaders: c.Headers,
			Debug:             c.DebugHTTP,
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

	return
}

func (c *Config) applyNetrcFile() error {
	// Get the netrc file path. If path not shown, then fall back to default netrc path value
	path := os.Getenv("NETRC_PATH")

	if path == "" {
		dir := os.Getenv("NETRC")
		if dir == "" {
			dir = "~"
		}

		filename := ".netrc"
		if runtime.GOOS == "windows" {
			filename = "_netrc"
		}

		var err error
		path, err = homedir.Expand(filepath.Join(dir, filename))
		if err != nil {
			return err
		}
	}

	// If the file is not a file, then do nothing
	if fi, err := os.Stat(path); err != nil {
		// File doesn't exist, do nothing
		if os.IsNotExist(err) {
			return nil
		}

		// Some other error!
		return err
	} else if fi.IsDir() {
		// File is directory, ignore
		return nil
	}

	// Load up the netrc file
	net, err := netrc.ParseFile(path)
	if err != nil {
		return fmt.Errorf("error parsing netrc file at %q: %s", path, err)
	}

	u, err := url.Parse(c.URL)
	if err != nil {
		return err
	}

	machine := net.FindMachine(u.Host)
	if machine == nil {
		// Machine not found, no problem
		return nil
	}

	// Set the user/api key/headers
	c.Email = machine.Login
	c.APIKey = machine.Password

	return nil
}
