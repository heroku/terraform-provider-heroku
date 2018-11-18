package heroku

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"os"
	"runtime"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/hashicorp/terraform/terraform"
	heroku "github.com/heroku/heroku-go/v3"
	"github.com/mitchellh/go-homedir"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_EMAIL", nil),
			},

			"api_key": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_API_KEY", nil),
			},
			"headers": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_HEADERS", nil),
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_API_URL", heroku.DefaultURL),
			},
			"delays": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"post_app_create_delay": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      DefaultPostAppCreateDelay,
							ValidateFunc: validation.IntAtLeast(0),
						},
						"post_space_create_delay": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      DefaultPostSpaceCreateDelay,
							ValidateFunc: validation.IntAtLeast(0),
						},
						"post_domain_create_delay": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      DefaultPostDomainCreateDelay,
							ValidateFunc: validation.IntAtLeast(0),
						},
					},
				},
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"heroku_account_feature":                   resourceHerokuAccountFeature(),
			"heroku_addon":                             resourceHerokuAddon(),
			"heroku_addon_attachment":                  resourceHerokuAddonAttachment(),
			"heroku_app":                               resourceHerokuApp(),
			"heroku_app_feature":                       resourceHerokuAppFeature(),
			"heroku_app_release":                       resourceHerokuAppRelease(),
			"heroku_build":                             resourceHerokuBuild(),
			"heroku_cert":                              resourceHerokuCert(),
			"heroku_domain":                            resourceHerokuDomain(),
			"heroku_drain":                             resourceHerokuDrain(),
			"heroku_formation":                         resourceHerokuFormation(),
			"heroku_pipeline":                          resourceHerokuPipeline(),
			"heroku_pipeline_coupling":                 resourceHerokuPipelineCoupling(),
			"heroku_slug":                              resourceHerokuSlug(),
			"heroku_space":                             resourceHerokuSpace(),
			"heroku_space_inbound_ruleset":             resourceHerokuSpaceInboundRuleset(),
			"heroku_space_app_access":                  resourceHerokuSpaceAppAccess(),
			"heroku_space_peering_connection_accepter": resourceHerokuSpacePeeringConnectionAccepter(),
			"heroku_space_vpn_connection":              resourceHerokuSpaceVPNConnection(),
			"heroku_team_collaborator":                 resourceHerokuTeamCollaborator(),
			"heroku_team_member":                       resourceHerokuTeamMember(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"heroku_addon":              dataSourceHerokuAddon(),
			"heroku_app":                dataSourceHerokuApp(),
			"heroku_space":              dataSourceHerokuSpace(),
			"heroku_space_peering_info": dataSourceHerokuSpacePeeringInfo(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{}

	headers := make(map[string]string)
	if h := d.Get("headers").(string); h != "" {
		if err := json.Unmarshal([]byte(h), &headers); err != nil {
			return nil, err
		}
	}

	h := make(http.Header)
	for k, v := range headers {
		h.Set(k, v)
	}

	config.Headers = h

	if url, ok := d.GetOk("url"); ok {
		config.URL = url.(string)
	}

	err := readNetrcFile(&config)
	if err != nil {
		return nil, err
	}

	if email, ok := d.GetOk("email"); ok {
		config.Email = email.(string)
	}

	if apiKey, ok := d.GetOk("api_key"); ok {
		config.APIKey = apiKey.(string)
	}

	err = applyDelayConfig(d, &config)

	log.Println("[INFO] Initializing Heroku client")
	if err := config.loadAndInitialize(); err != nil {
		return nil, err
	}

	return &config, nil
}

func applyDelayConfig(d *schema.ResourceData, config *Config) error {
	if v, ok := d.GetOk("delays"); ok {
		vL := v.([]interface{})
		if len(vL) > 1 {
			return fmt.Errorf("Provider configuration error: only 1 api config is permitted")
		}
		for _, v := range vL {
			apiConfig := v.(map[string]interface{})
			if v, ok := apiConfig["post_app_create_delay"].(int); ok && v != 0 {
				config.PostAppCreateDelay = int64(v)
			}
			if v, ok := apiConfig["post_space_create_delay"].(int); ok && v != 0 {
				config.PostSpaceCreateDelay = int64(v)
			}
			if v, ok := apiConfig["post_domain_create_delay"].(int); ok && v != 0 {
				config.PostDomainCreateDelay = int64(v)
			}
		}
	}
	return nil
}

func buildCompositeID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func parseCompositeID(id string) (p1 string, p2 string, err error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) == 2 {
		p1 = parts[0]
		p2 = parts[1]
	} else {
		err = fmt.Errorf("error: Import composite ID requires two parts separated by colon, eg x:y")
	}
	return
}

// Credit of this method is from https://github.com/Yelp/terraform-provider-signalform
func readNetrcFile(config *Config) error {
	// Get the netrc file path. If path not shown, then fall back to default netrc path value
	path := os.Getenv("NETRC_PATH")

	if path == "" {
		filename := ".netrc"
		if runtime.GOOS == "windows" {
			filename = "_netrc"
		}

		var err error
		path, err = homedir.Expand("~/" + filename)
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

	u, err := url.Parse(config.URL)
	if err != nil {
		return err
	}

	machine := net.FindMachine(u.Host)
	if machine == nil {
		// Machine not found, no problem
		return nil
	}

	// Set the user/api key/headers
	config.Email = machine.Login
	config.APIKey = machine.Password

	return nil
}
