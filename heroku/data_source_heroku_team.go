package heroku

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceHerokuTeam() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceHerokuTeamRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"default": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"membership_limit": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"provisioned_licenses": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceHerokuTeamRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api

	name := d.Get("name").(string)

	team, err := client.TeamInfo(context.TODO(), name)
	if err != nil {
		return err
	}

	d.SetId(team.ID)

	var setErr error
	setErr = d.Set("name", team.Name)
	setErr = d.Set("default", team.Default)
	setErr = d.Set("membership_limit", team.MembershipLimit)
	setErr = d.Set("provisioned_licenses", team.ProvisionedLicenses)
	setErr = d.Set("type", team.Type)

	return setErr
}
