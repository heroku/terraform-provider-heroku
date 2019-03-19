package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
	"log"
)

func resourceHerokuTeam() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuTeamCreate,
		Read:   resourceHerokuTeamRead,
		Update: resourceHerokuTeamUpdate,
		Delete: resourceHerokuTeamDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuTeamImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"default": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"credit_card_collections": {
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
		},
	}
}

func resourceHerokuTeamImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	team, err := client.TeamInfo(context.Background(), d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(team.ID)
	var setErr error
	setErr = d.Set("name", team.Name)
	setErr = d.Set("credit_card_collections", team.CreditCardCollections)
	setErr = d.Set("default", team.Default)
	setErr = d.Set("membership_limit", team.MembershipLimit)
	setErr = d.Set("provisioned_licenses", team.ProvisionedLicenses)

	return []*schema.ResourceData{d}, setErr
}

func resourceHerokuTeamCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.TeamCreateOpts{}

	if v, ok := d.GetOk("name"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] name: %s", vs)
		opts.Name = vs
	}

	log.Printf(fmt.Sprintf("[DEBUG] Creating new team named %v...", opts.Name))

	team, createErr := client.TeamCreate(context.TODO(), opts)
	if createErr != nil {
		return createErr
	}

	d.SetId(team.ID)
	log.Printf("[INFO] New Team ID: %s", d.Id())

	return resourceHerokuTeamRead(d, meta)
}

func resourceHerokuTeamRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[DEBUG] Reading Heroku team...")
	team, readErr := client.TeamInfo(context.TODO(), d.Id())
	if readErr != nil {
		return readErr
	}

	var setErr error
	setErr = d.Set("name", team.Name)
	setErr = d.Set("credit_card_collections", team.CreditCardCollections)
	setErr = d.Set("default", team.Default)
	setErr = d.Set("membership_limit", team.MembershipLimit)
	setErr = d.Set("provisioned_licenses", team.ProvisionedLicenses)

	return setErr
}

func resourceHerokuTeamUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.TeamUpdateOpts{}

	if d.HasChange("name") {
		v := d.Get("name").(string)
		log.Printf("[DEBUG] New name: %s", v)
		opts.Name = &v
	}

	log.Printf("[DEBUG] Updating Heroku team...")
	_, updateErr := client.TeamUpdate(context.TODO(), d.Id(), opts)
	if updateErr != nil {
		return updateErr
	}

	return resourceHerokuTeamRead(d, meta)
}

func resourceHerokuTeamDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[DEBUG] Deleting Heroku team...")
	_, deleteErr := client.TeamDelete(context.TODO(), d.Id())
	if deleteErr != nil {
		return deleteErr
	}

	d.SetId("")

	return nil
}
