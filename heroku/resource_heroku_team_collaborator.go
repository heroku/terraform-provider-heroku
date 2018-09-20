package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/heroku/heroku-go/v3"
	"log"
	"time"
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
	TeamCollaborator *herokuTeamCollaborator
	Client           *Config
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

func resourceHerokuTeamCollaboratorCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	opts := heroku.TeamAppCollaboratorCreateOpts{}

	appName := getAppName(d)

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
	collaborator, err := config.Api.TeamAppCollaboratorCreate(context.TODO(), appName, opts)
	if err != nil {
		return err
	}

	d.SetId(collaborator.ID)
	log.Printf("[INFO] New Collaborator ID: %s", d.Id())

	return resourceHerokuTeamCollaboratorRead(d, meta)
}

func resourceHerokuTeamCollaboratorRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	teamCollaborator, err := resourceHerokuTeamCollaboratorRetrieve(d.Id(), d.Get("app").(string), config)

	if err != nil {
		return err
	}

	d.Set("app", teamCollaborator.AppName)
	d.Set("email", teamCollaborator.TeamCollaborator.Email)
	d.Set("permissions", teamCollaborator.Permissions)

	return nil
}

func resourceHerokuTeamCollaboratorUpdate(d *schema.ResourceData, meta interface{}) error {
	// Enable Partial state mode to track what was successfully committed
	d.Partial(true)

	config := meta.(*Config)
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

	appName := getAppName(d)
	email := getEmail(d)

	log.Printf("[DEBUG] Updating Heroku Team Collaborator: [%s]", email)
	updatedTeamCollaborator, err := config.Api.TeamAppCollaboratorUpdate(context.TODO(), appName, email, opts)
	if err != nil {
		return err
	}

	d.SetPartial("permissions")

	d.SetId(updatedTeamCollaborator.ID)

	d.Partial(false)

	return resourceHerokuTeamCollaboratorRead(d, meta)
}

func resourceHerokuTeamCollaboratorDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	log.Printf("[INFO] Deleting Heroku Team Collaborator: [%s]", d.Id())
	_, err := config.Api.TeamAppCollaboratorDelete(context.TODO(), getAppName(d), getEmail(d))

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
		_, err := config.Api.TeamAppCollaboratorInfo(context.TODO(), getAppName(d), d.Id())

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
		return fmt.Errorf("[ERROR] Team collaborator [%s] still exists on [%s] after checking several times", getEmail(d), getAppName(d))
	}

	return nil
}

func resourceHerokuTeamCollaboratorRetrieve(id string, appName string, config *Config) (*teamCollaborator, error) {
	teamCollaborator := teamCollaborator{Id: id, AppName: appName, Client: config}

	err := teamCollaborator.Update()

	if err != nil {
		return nil, fmt.Errorf("[ERROR] Error retrieving team collaborator: %s", err)
	}

	return &teamCollaborator, nil
}

func (tc *teamCollaborator) Update() error {
	var errs []error

	log.Printf("[INFO] tc.Id is %s", tc.Id)

	teamCollaborator, err := tc.Client.Api.TeamAppCollaboratorInfo(context.TODO(), tc.AppName, tc.Id)

	if err != nil {
		errs = append(errs, err)
	} else {
		tc.TeamCollaborator = &herokuTeamCollaborator{}
		tc.TeamCollaborator.Email = teamCollaborator.User.Email
		tc.AppName = teamCollaborator.App.Name
	}

	// The underlying go client does not return permission info on the collaborator when calling
	// 'TeamAppCollaboratorInfo'. Instead that is returned via calling 'CollaboratorInfo'
	collaborator, collaboratorErr := tc.Client.Api.CollaboratorInfo(context.TODO(), tc.AppName, tc.Id)
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
	config := meta.(*Config)

	app, email := parseCompositeID(d.Id())

	collaborator, err := config.Api.CollaboratorInfo(context.Background(), app, email)
	if err != nil {
		return nil, err
	}

	d.SetId(collaborator.ID)
	d.Set("app", collaborator.App.Name)
	d.Set("email", collaborator.User.Email)
	d.Set("permissions", collaborator.Permissions)

	return []*schema.ResourceData{d}, nil
}
