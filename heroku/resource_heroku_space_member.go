package heroku

import (
	"context"
	"log"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuSpaceMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceMemberCreate,
		Read:   resourceHerokuSpaceMemberRead,
		Update: resourceHerokuSpaceMemberUpdate,
		Delete: resourceHerokuSpaceMemberDelete,

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
func resourceHerokuSpaceMemberCreate(d *schema.ResourceData, meta interface{}) error {
	_, err := syncSpaceAppAccessInfo(d.Get("email").(string), d, meta)
	if err != nil {
		return err
	}
	return resourceHerokuSpaceMemberUpdate(d, meta)
}

//callback for schema Resource.Read
func resourceHerokuSpaceMemberRead(d *schema.ResourceData, meta interface{}) error {
	_, err := syncSpaceAppAccessInfo(d.Id(), d, meta)
	return err
}

//callback for schema Resource.Update
func resourceHerokuSpaceMemberUpdate(d *schema.ResourceData, meta interface{}) error {
	currentPermissionsSet := d.Get("permissions").(*schema.Set)
	memberEmail := d.Get("email").(string)
	if d.HasChange("permissions") {
		log.Printf("[DEBUG] update current permissions (%s) for %s", spew.Sdump(currentPermissionsSet), memberEmail)
		client := meta.(*heroku.Service)
		opts := permissionsSchemaSetToSpaceAppAccessUpdateOpts(d.Get("permissions").(*schema.Set))
		space := d.Get("space").(string)
		_, err := client.SpaceAppAccessUpdate(context.TODO(), space, d.Id(), opts)
		if err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] current permissions (%s) for %s not changed", spew.Sdump(currentPermissionsSet), memberEmail)
	}
	return nil
}

//callback for schema Resource.Delete
//Members cannot be deleted from a space with this resource, they are removed
//from the state file and their permissions are cleared out via an update.
func resourceHerokuSpaceMemberDelete(d *schema.ResourceData, meta interface{}) error {
	currentPermissionsSet := d.Get("permissions").(*schema.Set)
	log.Printf("[DEBUG] removing current permissions (%s) for %s", spew.Sdump(currentPermissionsSet), d.Get("email").(string))
	emptyItems := make([]interface{}, 0)
	emptySet := schema.NewSet(currentPermissionsSet.F, emptyItems)
	d.Set("permissions", emptySet)
	return resourceHerokuSpaceMemberUpdate(d, meta)
}

//utility method to retrieve and sync the schema with the SpaceAppAccessInfo
//from the Heroku API using email or ID
func syncSpaceAppAccessInfo(emailOrID string, d *schema.ResourceData, meta interface{}) (*heroku.SpaceAppAccess, error) {
	client := meta.(*heroku.Service)
	space := d.Get("space").(string)
	spaceAppAccess, err := client.SpaceAppAccessInfo(context.TODO(), space, emailOrID)
	if err != nil {
		return nil, err
	}
	if g.Id() == "" {
		d.SetId(spaceAppAccess.User.ID)
	}
	d.Set("space", spaceAppAccess.Space.Name)
	d.Set("email", spaceAppAccess.User.Email)
	d.Set("permissions", spaceAppAccessPermissionsToSchemaSet(spaceAppAccess))
	log.Printf("[DEBUG] set permissions from Heroku API to (%s) for %s", spew.Sdump(d.Get("permissions").(*schema.Set)), emailOrID)
	return spaceAppAccess, nil
}

//utility method to convert SpaceAppAccess to a simple string array of
//permission names.
func spaceAppAccessPermissionsToSchemaSet(spaceAppAccess *heroku.SpaceAppAccess) []string {
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
func permissionsSchemaSetToSpaceAppAccessUpdateOpts(permSet *schema.Set) heroku.SpaceAppAccessUpdateOpts {
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
