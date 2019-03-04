package heroku

import "github.com/hashicorp/terraform/helper/schema"

func resourceHerokuConfigAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuConfigAssociationCreate,
		Read:   resourceHerokuConfigAssociationRead,
		Update: resourceHerokuConfigAssociationUpdate,
		Delete: resourceHerokuConfigAssociationDelete,

		//Importer: &schema.ResourceImporter{
		//	State: resourceHerokuConfigImport,
		//},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"config_vars": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"sensitive_config_vars": {
				Type:      schema.TypeMap,
				Sensitive: true,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
				},
			},
		},
	}
}

func resourceHerokuConfigAssociationCreate(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}

func resourceHerokuConfigAssociationRead(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}

func resourceHerokuConfigAssociationUpdate(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}

func resourceHerokuConfigAssociationDelete(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}
