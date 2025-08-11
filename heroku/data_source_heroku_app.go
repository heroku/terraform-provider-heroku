package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v6"
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

			"last_release_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"last_slug_id": {
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

			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceHerokuAppRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api

	name := d.Get("name").(string)
	app, err := resourceHerokuAppRetrieve(name, client)
	if err != nil {
		return err
	}

	d.SetId(app.App.ID)

	if app.IsTeamApp {
		setErr := setTeamDetails(d, app)
		if setErr != nil {
			return setErr
		}
	}

	setErr := setAppDetails(d, app)
	if setErr != nil {
		return setErr
	}

	d.Set("buildpacks", app.Buildpacks)
	d.Set("config_vars", app.Vars)

	releaseRange := heroku.ListRange{
		Field:      "version",
		Max:        200,
		Descending: true,
	}
	releases, err := client.ReleaseList(context.Background(), app.App.ID, &releaseRange)
	if err != nil {
		return fmt.Errorf("Failed to fetch releases for app '%s': %s", name, err)
	}
	for _, r := range releases {
		if r.Status == "succeeded" {
			d.Set("last_release_id", r.ID)
			if r.Slug != nil {
				d.Set("last_slug_id", r.Slug.ID)
			}
			break
		}
	}

	return nil
}
