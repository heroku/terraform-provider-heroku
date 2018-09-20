package heroku

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	heroku "github.com/heroku/heroku-go/v3"
)

func resourceHerokuTeamMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuTeamMemberSet,
		Read:   resourceHerokuTeamMemberRead,
		Update: resourceHerokuTeamMemberSet,
		Delete: resourceHerokuTeamMemberDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuTeamMemberImport,
		},

		Schema: map[string]*schema.Schema{
			"team": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"email": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"role": {
				Type:     schema.TypeString,
				Required: true,
			},

			"federated": {
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
			},
		},
	}
}

// Callback for schema.ResourceImporter
func resourceHerokuTeamMemberImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	team, email := parseCompositeID(d.Id())
	d.Set("team", team)
	d.Set("email", email)
	resourceHerokuTeamMemberRead(d, meta)
	return []*schema.ResourceData{d}, nil
}

// Callback for schema Resource.Create and schema Resource.Update
func resourceHerokuTeamMemberSet(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	email := d.Get("email").(string)
	federated := d.Get("federated").(bool)
	role := d.Get("role").(string)
	team := d.Get("team").(string)

	opts := heroku.TeamMemberCreateOrUpdateOpts{
		Email:     email,
		Role:      &role,
		Federated: &federated,
	}

	_, err := config.Api.TeamMemberCreateOrUpdate(context.TODO(), team, opts)
	if err != nil {
		return err
	}

	d.SetId(buildCompositeID(team, email))
	return resourceHerokuTeamMemberRead(d, meta)
}

// Callback for schema Resource.Read
func resourceHerokuTeamMemberRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	team, email := parseCompositeID(d.Id())

	members, err := config.Api.TeamMemberList(context.TODO(), team, &heroku.ListRange{Field: "email"})
	if err != nil {
		return err
	}

	var found heroku.TeamMember
	for _, member := range members {
		if member.Email == email {
			found = member
			break
		}
	}

	if found.ID == "" {
		return fmt.Errorf("Could not find member record for %s on team %s", email, team)
	}

	d.Set("team", team)
	d.Set("email", found.Email)
	d.Set("role", found.Role)
	d.Set("federated", found.Federated)

	return nil
}

// Callback for schema Resource.Delete
func resourceHerokuTeamMemberDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	team, email := parseCompositeID(d.Id())

	_, err := config.Api.TeamMemberDelete(context.TODO(), team, email)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
