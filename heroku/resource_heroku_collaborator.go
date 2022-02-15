package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

/**
herokuCollaborator is a value type used to hold the details of a
collaborator. We use this for common storage of values needed for the
heroku.Collaborator types
*/
type herokuCollaborator struct {
	Email string
}

// type collaborator is used to store all the details of a heroku collaborator
type collaborator struct {
	Id string // Id of the resource

	AppName      string // the app the collaborator belongs to
	AppID        string // the app the collaborator belongs to
	Collaborator *herokuCollaborator
	Client       *heroku.Service
}

func resourceHerokuCollaborator() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuCollaboratorCreate,
		Read:   resourceHerokuCollaboratorRead,
		Delete: resourceHerokuCollaboratorDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuCollaboratorImport,
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
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceHerokuCollaboratorV0().CoreConfigSchema().ImpliedType(),
				Upgrade: upgradeAppToAppID,
				Version: 0,
			},
		},
	}
}

func resourceHerokuCollaboratorCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.CollaboratorCreateOpts{}

	appID := getAppId(d)

	opts.User = getEmail(d)

	/**
	Setting the silent parameter to true by default. It is really an optional parameter that doesn't
	belong in the resource's state, especially since it's not part of the collaborator GET endpoint.
	*/
	vs := true
	opts.Silent = &vs

	log.Printf("[DEBUG] Creating Heroku Collaborator: [%s]", opts.User)
	createdCollaborator, err := client.CollaboratorCreate(context.TODO(), appID, opts)
	if err != nil {
		return err
	}

	d.SetId(createdCollaborator.ID)
	log.Printf("[INFO] New Collaborator ID: %s", d.Id())

	return resourceHerokuCollaboratorRead(d, meta)
}

func resourceHerokuCollaboratorRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	collaborator, err := resourceHerokuCollaboratorRetrieve(d.Id(), d.Get("app_id").(string), client)

	if err != nil {
		return err
	}

	d.Set("app_id", collaborator.AppID)
	d.Set("email", collaborator.Collaborator.Email)

	return nil
}

func resourceHerokuCollaboratorDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting Heroku Collaborator: [%s]", d.Id())
	_, err := client.CollaboratorDelete(context.TODO(), getAppId(d), getEmail(d))

	if err != nil {
		return fmt.Errorf("error deleting Collaborator: %s", err)
	}

	// So long as the DELETE succeeded, remove the resource from state
	d.SetId("")

	/**
	This is borrowed from the heroku team collaborator resource where it
	was noted that there's an edge scenario where if you immediately delete
	a team collaborator and recreate it, the Heroku api will complain the
	team collaborator still exists. Since these APIs are so similiar, I am
	following a similar pattern here.
	*/
	log.Printf("[INFO] Begin checking if [%s] has been deleted", getEmail(d))
	retryError := resource.Retry(10*time.Second, func() *resource.RetryError {
		_, err := client.CollaboratorInfo(context.TODO(), getAppId(d), d.Id())

		// Debug log to check
		log.Printf("[INFO] Is error nil when GET#show collaborator? %t", err == nil)

		// If err is nil, then that means the GET was successful and the collaborator still exists on the app
		if err == nil {
			// fmt.ErrorF does not output to log when TF_LOG=DEBUG is set to true, hence the need to execute log.PrintF for
			// debugging purpose and fmt.ErrorF so the retry func loops
			log.Printf("[WARNING] Collaborator [%s] exists after deletion. Checking again", getEmail(d))
			return resource.RetryableError(err)
		} else {
			// if there is an error in the GET, the collaborator no longer exists.
			return nil
		}
	})

	if retryError != nil {
		return fmt.Errorf("[ERROR] Collaborator [%s] still exists on [%s] after checking several times", getEmail(d), getAppId(d))
	}

	return nil
}

func resourceHerokuCollaboratorRetrieve(id string, appID string, client *heroku.Service) (*collaborator, error) {
	collaborator := collaborator{Id: id, AppID: appID, Client: client}

	err := collaborator.Update()

	if err != nil {
		return nil, fmt.Errorf("[ERROR] Error retrieving collaborator: %s", err)
	}

	return &collaborator, nil
}

func (c *collaborator) Update() error {
	var errs []error

	log.Printf("[INFO] c.Id is %s", c.Id)

	collaboratorInfo, err := c.Client.CollaboratorInfo(context.TODO(), c.AppID, c.Id)

	if err != nil {
		errs = append(errs, err)
	} else {
		c.Collaborator = &herokuCollaborator{}
		c.Collaborator.Email = collaboratorInfo.User.Email
		c.AppName = collaboratorInfo.App.Name
		c.AppID = collaboratorInfo.App.ID
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func resourceHerokuCollaboratorImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	app, email, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	collaboratorInfo, err := client.CollaboratorInfo(context.Background(), app, email)
	if err != nil {
		return nil, err
	}

	d.SetId(collaboratorInfo.ID)
	d.Set("app_id", collaboratorInfo.App.ID)
	d.Set("email", collaboratorInfo.User.Email)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuCollaboratorV0() *schema.Resource {
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
		},
	}
}
