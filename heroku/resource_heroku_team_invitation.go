package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
	"log"
)

func resourceHerokuTeamInvitation() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuTeamInvitationCreate,
		Read:   resourceHerokuTeamInvitationRead,
		Update: resourceHerokuTeamInvitationUpdate,
		Delete: resourceHerokuTeamInvitationDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuTeamInvitationImport,
		},

		Schema: map[string]*schema.Schema{
			"team_id": {
				Type:     schema.TypeString,
				Required: true,
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
		},
	}
}

func resourceHerokuTeamInvitationImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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

func resourceHerokuTeamInvitationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.TeamInvitationCreateOpts{}

	teamId := getTeamId(d)

	if v, ok := d.GetOk("email"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] email: %s", vs)
		opts.Email = vs
	}

	if v, ok := d.GetOk("role"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] role: %s", vs)
		opts.Role = &vs
	}

	log.Printf(fmt.Sprintf("[DEBUG] Creating new team invitation for %v...", opts.Email))

	teamInvitation, createErr := client.TeamInvitationCreate(context.TODO(), teamId, opts)
	if createErr != nil {
		return createErr
	}

	d.SetId(teamInvitation.ID)
	log.Printf("[INFO] New Team invitation ID: %s", d.Id())

	return resourceHerokuTeamInvitationRead(d, meta)
}

func resourceHerokuTeamInvitationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	teamId := getTeamId(d)
	email := getEmail(d)

	log.Printf("[DEBUG] Reading Heroku team invitation...")
	readOpts := &heroku.ListRange{
		Descending: true,
		FirstID:    d.Id(),
		LastID:     d.Id(),
	}

	teamInvitations, readErr := client.TeamInvitationList(context.TODO(), teamId, readOpts)
	if readErr != nil {
		return readErr
	}

	var setErr error
	readSuccess := false
	for _, invitation := range teamInvitations {
		if invitation.User.Email == email {
			setErr = d.Set("team_id", invitation.Team.ID)
			setErr = d.Set("email", invitation.User.Email)
			setErr = d.Set("role", invitation.Role)
			readSuccess = true
		}
	}

	if !readSuccess {
		return fmt.Errorf("[ERROR] Didn't properly read the resource from remote.")
	}

	return setErr
}

// resourceHerokuTeamInvitationUpdate will update the invitation's role using the
// same API Endpoint as resourceHerokuTeamInvitationCreate.
func resourceHerokuTeamInvitationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	teamId := getTeamId(d)
	opts := heroku.TeamInvitationCreateOpts{}

	// Automatically set the opts's email value to the associated resource value
	opts.Email = getEmail(d)

	if d.HasChange("email") {
		v := d.Get("email").(string)
		log.Printf("[DEBUG] New name: %s", v)
		opts.Email = v
	}

	log.Printf("[DEBUG] Updating Heroku team invitation...")
	_, updateErr := client.TeamInvitationCreate(context.TODO(), teamId, opts)
	if updateErr != nil {
		return updateErr
	}

	// Although we are using the same heroku client function to update the resource,
	// the resource's remote id does not change. Therefore it is not necessary to set the ID again.

	return resourceHerokuTeamInvitationRead(d, meta)
}

// resourceHerokuTeamInvitationDelete will revoke the invitation
func resourceHerokuTeamInvitationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	teamId := getTeamId(d)

	log.Printf("[DEBUG] Revoking Heroku team invitation...")
	_, deleteErr := client.TeamInvitationRevoke(context.TODO(), teamId, d.Id())
	if deleteErr != nil {
		return deleteErr
	}

	d.SetId("")

	return nil
}
