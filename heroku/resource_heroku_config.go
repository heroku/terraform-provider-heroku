package heroku

import "github.com/hashicorp/terraform/helper/schema"

func resourceHerokuConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuConfigCreate,
		Read:   resourceHerokuConfigRead,
		Update: resourceHerokuConfigUpdate,
		Delete: resourceHerokuConfigDelete,

		//Importer: &schema.ResourceImporter{
		//	State: resourceHerokuConfigImport,
		//},

		Schema: map[string]*schema.Schema{
			"vars": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"sensitive_vars": {
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

func resourceHerokuConfigCreate(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}

func resourceHerokuConfigRead(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}

func resourceHerokuConfigUpdate(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}

func resourceHerokuConfigDelete(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {

}
