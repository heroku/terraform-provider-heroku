package heroku

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_EMAIL", nil),
			},

			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_API_KEY", nil),
			},
			"headers": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("HEROKU_HEADERS", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"heroku_addon":                             resourceHerokuAddon(),
			"heroku_addon_attachment":                  resourceHerokuAddonAttachment(),
			"heroku_app":                               resourceHerokuApp(),
			"heroku_app_feature":                       resourceHerokuAppFeature(),
			"heroku_app_release":                       resourceHerokuAppRelease(),
			"heroku_cert":                              resourceHerokuCert(),
			"heroku_domain":                            resourceHerokuDomain(),
			"heroku_drain":                             resourceHerokuDrain(),
			"heroku_formation":                         resourceHerokuFormation(),
			"heroku_pipeline":                          resourceHerokuPipeline(),
			"heroku_pipeline_coupling":                 resourceHerokuPipelineCoupling(),
			"heroku_space":                             resourceHerokuSpace(),
			"heroku_space_inbound_ruleset":             resourceHerokuSpaceInboundRuleset(),
			"heroku_space_member":                      resourceHerokuSpaceMember(),
			"heroku_space_peering_connection_accepter": resourceHerokuSpacePeeringConnectionAccepter(),
			"heroku_team_collaborator":                 resourceHerokuTeamCollaborator(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"heroku_space":              dataSourceHerokuSpace(),
			"heroku_space_peering_info": dataSourceHerokuSpacePeeringInfo(),
			"heroku_app":                dataSourceHerokuApp(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
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

	config := Config{
		Email:   d.Get("email").(string),
		APIKey:  d.Get("api_key").(string),
		Headers: h,
	}

	log.Println("[INFO] Initializing Heroku client")
	return config.Client()
}

func buildCompositeID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func parseCompositeID(id string) (string, string) {
	parts := strings.SplitN(id, ":", 2)
	return parts[0], parts[1]
}
