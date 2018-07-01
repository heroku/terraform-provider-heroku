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

//There's no actual method to create a space member, so we actually just need to import
//the member into the state and update if needed.
func resourceHerokuSpaceMemberCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] lifecycle `Create`")
	client := meta.(*heroku.Service)
	space := d.Get("space").(string)
	email := d.Get("email").(string)
	spaceAppAccess, err := client.SpaceAppAccessInfo(context.TODO(), space, email)
	log.Printf("[DEBUG] lifecycle `Create` retrieved data %s", spew.Sdump(spaceAppAccess))
	if err != nil {
		return err
	}
	d.SetId(spaceAppAccess.User.ID)
	d.Set("space", spaceAppAccess.Space.Name)
	d.Set("email", spaceAppAccess.User.Email)
	d.Set("permissions", spaceAppAccessPermissionsToSchemaSet(spaceAppAccess))

	return resourceHerokuSpaceMemberUpdate(d, meta)
}

func resourceHerokuSpaceMemberRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] lifecycle `Read`")
	client := meta.(*heroku.Service)
	space := d.Get("space").(string)
	//Can take the email or an ID. Use the ID to help make sure we keep consistent with the state file
	spaceAppAccess, err := client.SpaceAppAccessInfo(context.TODO(), space, d.Id())
	log.Printf("[DEBUG] lifecycle `Read` retrieved data %s", spew.Sdump(spaceAppAccess))
	if err != nil {
		return err
	}
	d.Set("space", spaceAppAccess.Space.Name)
	d.Set("email", spaceAppAccess.User.Email)
	d.Set("permissions", spaceAppAccessPermissionsToSchemaSet(spaceAppAccess))

	return nil
}

func resourceHerokuSpaceMemberUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] lifecycle `Update`")
	log.Printf("[DEBUG] lifecycle `Update` current permissions %s", spew.Sdump(d.Get("permissions").(*schema.Set)))
	if d.HasChange("permissions") {
		log.Printf("[DEBUG] lifecycle `Update` permissions CHANGED")
		client := meta.(*heroku.Service)
		opts := permissionsSchemaSetToSpaceAppAccessUpdateOpts(d.Get("permissions").(*schema.Set))
		space := d.Get("space").(string)
		log.Printf("[DEBUG] lifecycle `Update` API opts: %s", spew.Sdump(opts))
		_, err := client.SpaceAppAccessUpdate(context.TODO(), space, d.Id(), opts)
		if err != nil {
			return err
		}
	} else {
		log.Printf("[DEBUG] lifecycle `Update` permissions NOT CHANGED")
	}
	return nil
}

//return permissions to none?
func resourceHerokuSpaceMemberDelete(d *schema.ResourceData, meta interface{}) error {
	// client := meta.(*heroku.Service)
	log.Printf("[DEBUG] lifecycle `Delete`")
	return nil
}

func spaceAppAccessPermissionsToSchemaSet(spaceAppAccess *heroku.SpaceAppAccess) []string {
	perms := make([]string, 0)
	if spaceAppAccess != nil {
		for _, perm := range spaceAppAccess.Permissions {
			perms = append(perms, perm.Name)
		}
	}
	log.Printf("[DEBUG] permissions set to %s", spew.Sdump(perms))
	return perms
}

type permissionUpdateItem = struct {
	Name *string `json:"name,omitempty" url:"name,omitempty,key"`
}

func permissionsSchemaSetToSpaceAppAccessUpdateOpts(permSet *schema.Set) heroku.SpaceAppAccessUpdateOpts {
	permissions := make([]*permissionUpdateItem, 0)
	opts := heroku.SpaceAppAccessUpdateOpts{Permissions: permissions}
	for _, perm := range permSet.List() {
		permName := perm.(string)
		permOpt := permissionUpdateItem{Name: &permName}
		opts.Permissions = append(opts.Permissions, &permOpt)
	}
	return opts
}
