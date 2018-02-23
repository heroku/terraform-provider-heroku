package heroku

import (
	"context"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceHerokuSpace() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceHerokuSpaceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"region": {
				Type:     schema.TypeString,
				Computed: true,
				Default:  nil,
			},

			"state": {
				Type:     schema.TypeString,
				Computed: true,
				Default:  nil,
			},

			"shield": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"organization": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceHerokuSpaceRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*heroku.Service)

	name := d.Get("name").(string)
	space, err := client.SpaceInfo(context.TODO(), name)
	if err != nil {
		return err
	}

	d.SetId(name)
	d.Set("region", space.Region.Name)
	d.Set("name", name)
	d.Set("state", space.State)
	d.Set("organization", space.Organization.Name)

	if space.Shield {
		d.Set("shield", "on")
	} else {
		d.Set("shield", "off")
	}

	return nil
}
