package heroku

import (
	"context"
	"log"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuSpaceAppAccess() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceAppAccessCreate,
		Read:   resourceHerokuSpaceAppAccessRead,
		Update: resourceHerokuSpaceAppAccessUpdate,
		Delete: resourceHerokuSpaceAppAccessDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
				Optional: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

//callback for schema Resource.Create
//There's no actual method to create a space member, so we just need to import
//the existing member into the state and update if needed.
func resourceHerokuSpaceAppAccessCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] resourceHerokuSpaceAppAccessCreate")
	err := resourceHerokuSpaceAppAccessUpdate(d, meta)
	if err != nil {
		return err
	}
	return resourceHerokuSpaceAppAccessRead(d, meta)
}

//callback for schema Resource.Read
func resourceHerokuSpaceAppAccessRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)
	space := d.Get("space").(string)
	email := d.Get("email").(string)
	spaceAppAccess, err := client.SpaceAppAccessInfo(context.TODO(), space, email)
	if err != nil {
		return err
	}
	d.SetId(spaceAppAccess.User.ID)
	d.Set("space", spaceAppAccess.Space.Name)
	d.Set("email", spaceAppAccess.User.Email)
	d.Set("permissions", spaceAppAccessPermissionsToList(spaceAppAccess))
	return err
}

//callback for schema Resource.Update
func resourceHerokuSpaceAppAccessUpdate(d *schema.ResourceData, meta interface{}) error {
	currentPermissionsSet := d.Get("permissions").(*schema.Set)
	email := d.Get("email").(string)
	space := d.Get("space").(string)
	opts := createSpaceAppAccessUpdateOpts(currentPermissionsSet)
	client := meta.(*heroku.Service)
	_, err := client.SpaceAppAccessUpdate(context.TODO(), space, email, opts)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] updated permissions (%v) for %s", currentPermissionsSet.List(), email)
	return nil
}

//callback for schema Resource.Delete
//Members cannot be deleted from a space with this resource, they are removed
//from the state file and their permissions are cleared out.
func resourceHerokuSpaceAppAccessDelete(d *schema.ResourceData, meta interface{}) error {
	currentPermissionsSet := d.Get("permissions").(*schema.Set)
	log.Printf("[DEBUG] removing current permissions (%v) for %s", currentPermissionsSet.List(), d.Get("email").(string))
	d.Set("permissions", make([]string, 0))
	return resourceHerokuSpaceAppAccessUpdate(d, meta)
}

//utility method to convert SpaceAppAccess to a simple string array of
//permission names.
func spaceAppAccessPermissionsToList(spaceAppAccess *heroku.SpaceAppAccess) []string {
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
	for _, perm := range permSet.List() {
		permName := perm.(string)
		permOpt := struct {
			Name *string `json:"name,omitempty" url:"name,omitempty,key"`
		}{&permName}
		opts.Permissions = append(opts.Permissions, permOpt)
	}
	return opts
}
