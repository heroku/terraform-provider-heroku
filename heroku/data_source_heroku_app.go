package heroku

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceHerokuApp() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceHerokuAppRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"space": {
				Type:     schema.TypeString,
				Computed: true,
				Default:  nil,
			},

			"region": {
				Type:     schema.TypeString,
				Computed: true,
				Default:  nil,
			},

			"stack": {
				Type:     schema.TypeString,
				Computed: true,
				Default:  nil,
			},

			"internal_routing": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"buildpacks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"config_vars": {
				Type:     schema.TypeMap,
				Computed: true,
			},

			"git_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"web_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"acm": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"heroku_hostname": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"organization": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"locked": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"personal": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceHerokuAppRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api

	name := d.Get("name").(string)
	app, err := resourceHerokuAppRetrieve(name, true, client)
	if err != nil {
		return err
	}

	d.SetId(time.Now().UTC().String())

	if app.Organization {
		err := setOrganizationDetails(d, app)
		if err != nil {
			return err
		}
	}

	setAppDetails(d, app)

	d.Set("buildpacks", app.Buildpacks)
	d.Set("config_vars", app.Vars)

	return nil
}
