package heroku

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuSpaceAppAccess() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceAppAccessSet,
		Read:   resourceHerokuSpaceAppAccessRead,
		Update: resourceHerokuSpaceAppAccessSet,
		Delete: resourceHerokuSpaceAppAccessDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuSpaceAppAccessImport,
		},

		Schema: map[string]*schema.Schema{
			"space": {
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
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

//callback for schema.ResourceImporter
func resourceHerokuSpaceAppAccessImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	space, email, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}
	d.Set("space", space)
	d.Set("email", email)
	readErr := resourceHerokuSpaceAppAccessRead(d, meta)
	if readErr != nil {
		return nil, readErr
	}
	return []*schema.ResourceData{d}, nil
}

//callback for schema Resource.Create and schema Resource.Update
func resourceHerokuSpaceAppAccessSet(d *schema.ResourceData, meta interface{}) error {
	_, err := updateSpaceAppAccess(d.Get("permissions").(*schema.Set), d, meta)
	if err != nil {
		return err
	}
	return resourceHerokuSpaceAppAccessRead(d, meta)
}

//callback for schema Resource.Read
func resourceHerokuSpaceAppAccessRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api
	space := d.Get("space").(string)
	email := d.Get("email").(string)
	spaceAppAccess, err := client.SpaceAppAccessInfo(context.TODO(), space, email)
	if err != nil {
		return err
	}
	d.SetId(spaceAppAccess.User.ID)
	d.Set("space", spaceAppAccess.Space.Name)
	d.Set("email", spaceAppAccess.User.Email)
	d.Set("permissions", createPermissionsList(spaceAppAccess))
	return nil
}

//callback for schema Resource.Delete
//Members cannot be deleted from a space with this resource, they are removed
//from the state file and their permissions are cleared out.
func resourceHerokuSpaceAppAccessDelete(d *schema.ResourceData, meta interface{}) error {
	_, err := updateSpaceAppAccess(nil, d, meta)
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

//utility method to call heroku.SpaceAppAccessUpdate
func updateSpaceAppAccess(permissions *schema.Set, d *schema.ResourceData, meta interface{}) (*heroku.SpaceAppAccess, error) {
	email := d.Get("email").(string)
	space := d.Get("space").(string)
	opts := createSpaceAppAccessUpdateOpts(permissions)
	client := meta.(*Config).Api
	spaceAppAccess, err := client.SpaceAppAccessUpdate(context.TODO(), space, email, opts)
	if err != nil {
		return nil, err
	}
	return spaceAppAccess, nil
}

//utility method to convert SpaceAppAccess to a simple string array of
//permission names.
func createPermissionsList(spaceAppAccess *heroku.SpaceAppAccess) []string {
	perms := make([]string, 0)
	if spaceAppAccess != nil {
		for _, perm := range spaceAppAccess.Permissions {
			perms = append(perms, perm.Name)
		}
	}
	return perms
}

//utility method to convert a schema.Set of simple permission names to
//SpaceAppAccessUpdateOpts suitable as input into the the heroku API.
func createSpaceAppAccessUpdateOpts(permSet *schema.Set) heroku.SpaceAppAccessUpdateOpts {
	//The choice of using anonymous structs in heroku-go should be revisited per
	//https://github.com/interagent/schematic/issues/17
	permissions := make([]struct {
		Name *string `json:"name,omitempty" url:"name,omitempty,key"`
	}, 0)
	opts := heroku.SpaceAppAccessUpdateOpts{Permissions: permissions}
	if permSet != nil {
		for _, perm := range permSet.List() {
			permName := perm.(string)
			permOpt := struct {
				Name *string `json:"name,omitempty" url:"name,omitempty,key"`
			}{&permName}
			opts.Permissions = append(opts.Permissions, permOpt)
		}
	}
	return opts
}
