package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

/**
Heroku's collaborator & team collaborator overlap in several of their CRUD endpoints.
However, these two resources have minute differences similar to the square/rectangle analogy.
Given that is likely a heroku provider user will likely be using teams, I'm implementing this
resource first. So if you have a team/org, please use this resource.
*/

/**
herokuTeamCollaborator is a value type used to hold the details of a
team collaborator. We use this for common storage of values needed for the
heroku.TeamCollaborator types
*/
type herokuTeamCollaborator struct {
	Email string
}

// type teamCollaborator is used to store all the details of a heroku team collaborator
type teamCollaborator struct {
	Id string // Id of the resource

	AppName          string // the app the collaborator belongs to
	AppID            string // the app the collaborator belongs to
	TeamCollaborator *herokuTeamCollaborator
	Client           *heroku.Service
	Permissions      []string // can be a combo or all of ["view", "deploy", "operate", "manage"]
}

func resourceHerokuTeamCollaborator() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuTeamCollaboratorCreate,
		Read:   resourceHerokuTeamCollaboratorRead,
		Update: resourceHerokuTeamCollaboratorUpdate,
		Delete: resourceHerokuTeamCollaboratorDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuTeamCollaboratorImport,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},

			"email": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"permissions": {
				Type:     schema.TypeSet, // We are using TypeSet type here as the order for permissions is not important.
				Required: true,
				MinItems: 1,
				MaxItems: 4,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuTeamCollaboratorV0().CoreConfigSchema().ImpliedType(),
				Upgrade: upgradeAppToAppID,
				Version: 0,
			},
		},
	}
}

func resourceHerokuTeamCollaboratorCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.TeamAppCollaboratorCreateOpts{}

	appID := getAppId(d)

	opts.User = getEmail(d)

	/**
	Setting the silent parameter to true by default. It is really an optional parameter that doesn't
	belong in the resource's state, especially since it's not part of the collaborator GET endpoint.
	After several iterations to keep it as part of schema but ignoring a state diff, nothing worked out well.
	*/
	vs := true
	opts.Silent = &vs

	if v, ok := d.GetOk("permissions"); ok {
		permsSet := v.(*schema.Set)
		perms := make([]*string, permsSet.Len())

		for i, perm := range permsSet.List() {
			p := perm.(string)
			perms[i] = &p
		}

		log.Printf("[DEBUG] Permissions: %v", perms)
		opts.Permissions = perms
	}

	log.Printf("[DEBUG] Creating Heroku Team Collaborator: [%s]", opts.User)

	var resourceID string
	collaborator, createErr := client.TeamAppCollaboratorCreate(context.TODO(), appID, opts)
	if createErr != nil {
		// Handle scenario when user has already been granted access to the app.
		if strings.Contains(strings.ToLower(createErr.Error()), "is already a collaborator on app") {
			// Loop through all collaborators on the app to get the collaborator ID
			collaborators, listErr := client.TeamAppCollaboratorList(context.TODO(), appID,
				&heroku.ListRange{Max: 1000, Descending: true})
			if listErr != nil {
				return listErr
			}

			for _, c := range collaborators {
				if c.User.Email == opts.User {
					resourceID = c.ID
					break
				}
			}
		} else {
			return createErr
		}
	} else {
		resourceID = collaborator.ID
	}

	d.SetId(resourceID)
	log.Printf("[INFO] New Collaborator ID: %s", d.Id())

	return resourceHerokuTeamCollaboratorRead(d, meta)
}

func resourceHerokuTeamCollaboratorRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	teamCollaborator, err := resourceHerokuTeamCollaboratorRetrieve(d.Id(), d.Get("app_id").(string), client)

	if err != nil {
		if strings.Contains(err.Error(), "Couldn't find that user") {
			// If user cannot be found, remove the resource from state.
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("app_id", teamCollaborator.AppID)
	d.Set("email", teamCollaborator.TeamCollaborator.Email)
	d.Set("permissions", teamCollaborator.Permissions)

	return nil
}

func resourceHerokuTeamCollaboratorUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	opts := heroku.TeamAppCollaboratorUpdateOpts{}

	if d.HasChange("permissions") {
		permsSet := d.Get("permissions").(*schema.Set)
		perms := make([]string, permsSet.Len())

		for i, perm := range permsSet.List() {
			perms[i] = perm.(string)
		}

		log.Printf("[DEBUG] Permissions: %s", perms)
		opts.Permissions = perms
	}

	appID := getAppId(d)
	email := getEmail(d)

	log.Printf("[DEBUG] Updating Heroku Team Collaborator: [%s]", email)
	updatedTeamCollaborator, err := client.TeamAppCollaboratorUpdate(context.TODO(), appID, email, opts)
	if err != nil {
		return err
	}

	d.SetId(updatedTeamCollaborator.ID)

	return resourceHerokuTeamCollaboratorRead(d, meta)
}

func resourceHerokuTeamCollaboratorDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting Heroku Team Collaborator: [%s]", d.Id())
	_, err := client.TeamAppCollaboratorDelete(context.TODO(), getAppId(d), getEmail(d))

	if err != nil {
		return fmt.Errorf("error deleting Team Collaborator: %s", err)
	}

	// So long as the DELETE succeeded, remove the resource from state
	d.SetId("")

	/**
	There's a edge scenario where if you immediately delete a team collaborator and recreate it, the Heroku api
	will complain the team collaborator still exists. So to remedy this, we will do a GET for the collaborator
	until it 404s before proceeding further.
	*/
	log.Printf("[INFO] Begin checking if [%s] has been deleted", getEmail(d))
	retryError := resource.Retry(10*time.Second, func() *resource.RetryError {
		_, err := client.TeamAppCollaboratorInfo(context.TODO(), getAppId(d), d.Id())

		// Debug log to check
		log.Printf("[INFO] Is error nil when GET#show team collaborator? %t", err == nil)

		// If err is nil, then that means the GET was successful and the collaborator still exists on the team app
		if err == nil {
			// fmt.ErrorF does not output to log when TF_LOG=DEBUG is set to true, hence the need to execute log.PrintF for
			// debugging purpose and fmt.ErrorF so the retry func loops
			log.Printf("[WARNING] Team collaborator [%s] exists after deletion. Checking again", getEmail(d))
			return resource.RetryableError(err)
		} else {
			// if there is an error in the GET, the collaborator no longer exists.
			return nil
		}
	})

	if retryError != nil {
		return fmt.Errorf("[ERROR] Team collaborator [%s] still exists on [%s] after checking several times", getEmail(d), getAppId(d))
	}

	return nil
}

func resourceHerokuTeamCollaboratorRetrieve(id string, appID string, client *heroku.Service) (*teamCollaborator, error) {
	teamCollaborator := teamCollaborator{Id: id, AppID: appID, Client: client}

	err := teamCollaborator.Update()

	if err != nil {
		return nil, fmt.Errorf("[ERROR] Error retrieving team collaborator: %s", err)
	}

	return &teamCollaborator, nil
}

func (tc *teamCollaborator) Update() error {
	var errs []error

	log.Printf("[INFO] tc.Id is %s", tc.Id)

	teamCollaborator, err := tc.Client.TeamAppCollaboratorInfo(context.TODO(), tc.AppID, tc.Id)

	if err != nil {
		errs = append(errs, err)
	} else {
		tc.TeamCollaborator = &herokuTeamCollaborator{}
		tc.TeamCollaborator.Email = teamCollaborator.User.Email
		tc.AppName = teamCollaborator.App.Name
		tc.AppID = teamCollaborator.App.ID
	}

	// The underlying go client does not return permission info on the collaborator when calling
	// 'TeamAppCollaboratorInfo'. Instead that is returned via calling 'CollaboratorInfo'
	collaborator, collaboratorErr := tc.Client.CollaboratorInfo(context.TODO(), tc.AppID, tc.Id)
	if collaboratorErr != nil {
		errs = append(errs, collaboratorErr)
	} else {
		// build the slice of perms
		perms := make([]string, 0, len(collaborator.Permissions))
		for _, perm := range collaborator.Permissions {
			perms = append(perms, perm.Name)
		}

		tc.Permissions = perms
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func resourceHerokuTeamCollaboratorImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, email, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	collaborator, err := client.CollaboratorInfo(context.Background(), app, email)
	if err != nil {
		return nil, err
	}

	d.SetId(collaborator.ID)
	d.Set("app_id", collaborator.App.ID)
	d.Set("email", collaborator.User.Email)

	var perms []string
	for _, p := range collaborator.Permissions {
		perms = append(perms, p.Name)
	}

	d.Set("permissions", perms)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuTeamCollaboratorV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"email": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"permissions": {
				Type:     schema.TypeSet, // We are using TypeSet type here as the order for permissions is not important.
				Required: true,
				MinItems: 1,
				MaxItems: 4,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}
