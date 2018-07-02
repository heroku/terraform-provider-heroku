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
//the existing member into the state and update if needed.
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
	log.Printf("[DEBUG] lifecycle `Create` current permissions %s", spew.Sdump(d.Get("permissions").(*schema.Set)))
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
	log.Printf("[DEBUG] lifecycle `Read` current permissions %s", spew.Sdump(d.Get("permissions").(*schema.Set)))
	d.Set("permissions", spaceAppAccessPermissionsToSchemaSet(spaceAppAccess))
	log.Printf("[DEBUG] lifecycle `Read` altered permissions %s", spew.Sdump(d.Get("permissions").(*schema.Set)))

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
			log.Printf("[DEBUG] lifecycle `Update` API call failed: %s", spew.Sdump(err))
			return err
		}
		log.Printf("[DEBUG] lifecycle `Update` API call done")
	} else {
		log.Printf("[DEBUG] lifecycle `Update` permissions NOT CHANGED")
	}
	return nil
}

//Members cannot be deleted from a space with this resource, they are removed from the state file
//and their permissions are cleared out via an update.
func resourceHerokuSpaceMemberDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] lifecycle `Delete`")
	currentSet := d.Get("permissions").(*schema.Set)
	log.Printf("[DEBUG] lifecycle `Delete` current permissions %s", spew.Sdump(currentSet))
	emptyItems := make([]interface{}, 0)
	emptySet := schema.NewSet(currentSet.F, emptyItems)
	d.Set("permissions", emptySet)
	return resourceHerokuSpaceMemberUpdate(d, meta)
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

//The choice of using anonymous structs in heroku-go should be revisited per
//https://github.com/interagent/schematic/issues/17
func permissionsSchemaSetToSpaceAppAccessUpdateOpts(permSet *schema.Set) heroku.SpaceAppAccessUpdateOpts {
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
