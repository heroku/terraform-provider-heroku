package heroku

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHerokuSpace() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceHerokuSpaceRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"organization": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cidr": {
				Type:     schema.TypeString,
				Computed: true,
				Default:  nil,
				Optional: true,
			},

			"data_cidr": {
				Type:     schema.TypeString,
				Computed: true,
				Default:  nil,
				Optional: true,
			},

			"outbound_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
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
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataSourceHerokuSpaceRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api

	name := d.Get("name").(string)
	spaceRaw, _, err := SpaceStateRefreshFunc(client, name)()
	if err != nil {
		return err
	}

	space := spaceRaw.(*spaceWithNAT)

	d.SetId(name)
	d.Set("state", space.State)
	d.Set("shield", space.Shield)

	return resourceHerokuSpaceRead(d, m)
}
