package heroku

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

// Provider returns a terraform.ResourceProvider.
func Provider() *schema.Provider {
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

			"customizations": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"set_app_all_config_vars_in_state": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  DefaultSetAppAllConfigVarsInState,
						},
					},
				},
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

			"timeouts": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"addon_create_timeout": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      DefaultAddonCreateTimeout,
							ValidateFunc: validation.IntAtLeast(10),
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
			"heroku_app_config_association":            resourceHerokuAppConfigAssociation(),
			"heroku_app_feature":                       resourceHerokuAppFeature(),
			"heroku_app_release":                       resourceHerokuAppRelease(),
			"heroku_app_webhook":                       resourceHerokuAppWebhook(),
			"heroku_build":                             resourceHerokuBuild(),
			"heroku_cert":                              resourceHerokuCert(),
			"heroku_collaborator":                      resourceHerokuCollaborator(),
			"heroku_config":                            resourceHerokuConfig(),
			"heroku_domain":                            resourceHerokuDomain(),
			"heroku_drain":                             resourceHerokuDrain(),
			"heroku_formation":                         resourceHerokuFormation(),
			"heroku_pipeline":                          resourceHerokuPipeline(),
			"heroku_pipeline_config_var":               resourceHerokuPipelineConfigVar(),
			"heroku_pipeline_coupling":                 resourceHerokuPipelineCoupling(),
			"heroku_review_app_config":                 resourceHerokuReviewAppConfig(),
			"heroku_slug":                              resourceHerokuSlug(),
			"heroku_space":                             resourceHerokuSpace(),
			"heroku_space_inbound_ruleset":             resourceHerokuSpaceInboundRuleset(),
			"heroku_space_app_access":                  resourceHerokuSpaceAppAccess(),
			"heroku_space_peering_connection_accepter": resourceHerokuSpacePeeringConnectionAccepter(),
			"heroku_space_vpn_connection":              resourceHerokuSpaceVPNConnection(),
			"heroku_ssl":                               resourceHerokuSSL(),
			"heroku_team_collaborator":                 resourceHerokuTeamCollaborator(),
			"heroku_team_member":                       resourceHerokuTeamMember(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"heroku_addon":              dataSourceHerokuAddon(),
			"heroku_app":                dataSourceHerokuApp(),
			"heroku_pipeline":           dataSourceHerokuPipeline(),
			"heroku_space":              dataSourceHerokuSpace(),
			"heroku_space_peering_info": dataSourceHerokuSpacePeeringInfo(),
			"heroku_team":               dataSourceHerokuTeam(),
			"heroku_team_members":       dataSourceHerokuTeamMembers(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	log.Println("[INFO] Initializing Heroku provider")
	config := NewConfig()

	if err := config.applySchema(d); err != nil {
		return nil, err
	}

	if err := config.applyNetrcFile(); err != nil {
		return nil, err
	}

	//the provider resource schema takes precedence over Netrc
	if email, ok := d.GetOk("email"); ok {
		config.Email = email.(string)
	}

	if apiKey, ok := d.GetOk("api_key"); ok {
		config.APIKey = apiKey.(string)
	}

	if err := config.initializeAPI(); err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Heroku provider initialized: %s\n", config)

	return config, nil
}
