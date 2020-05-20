package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"log"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
)

func resourceHerokuPipeline() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuPipelineCreate,
		Update: resourceHerokuPipelineUpdate,
		Read:   resourceHerokuPipelineRead,
		Delete: resourceHerokuPipelineDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuPipelineImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[a-z][a-z0-9-]{2,29}$`),
					"invalid pipeline name"),
			},

			"owner": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.IsUUID,
						},

						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{"team", "user"}, false),
						},
					},
				},
			},
		},
	}
}

func resourceHerokuPipelineImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api

	p, err := client.PipelineInfo(context.TODO(), d.Id())
	if err != nil {
		return nil, err
	}

	d.SetId(p.ID)
	setPipelineAttributes(d, p)

	return []*schema.ResourceData{d}, nil
}

func resourceHerokuPipelineCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	opts := heroku.PipelineCreateOpts{}

	if v, ok := d.GetOk("name"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] New pipeline name: %s", vs)
		opts.Name = vs
	}

	// If the owner is set, use it. Otherwise, pipeline ownership will default
	// to the authenticated user for this provider.
	opts.Owner = (*struct {
		ID   string `json:"id" url:"id,key"`
		Type string `json:"type" url:"type,key"`
	})(&struct {
		ID   string
		Type string
	}{ID: "", Type: ""})
	if v, ok := d.GetOk("owner"); ok {
		vi := v.([]interface{})
		ownerInfo := vi[0].(map[string]interface{})

		ownerID := ownerInfo["id"].(string)
		ownerType := ownerInfo["type"].(string)

		opts.Owner.ID = ownerID
		opts.Owner.Type = ownerType
	} else {
		authUser, authGetUserErr := client.AccountInfo(context.TODO())
		if authGetUserErr != nil {
			return authGetUserErr
		}

		opts.Owner.ID = authUser.ID
		opts.Owner.Type = "user"
	}

	log.Printf("[DEBUG] New pipeline owner id: %s", opts.Owner.ID)
	log.Printf("[DEBUG] New pipeline owner type: %s", opts.Owner.Type)

	log.Printf("[DEBUG] Pipeline create configuration: %#v", opts)

	p, err := client.PipelineCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("Error creating pipeline: %s", err)
	}

	d.SetId(p.ID)

	log.Printf("[INFO] Pipeline ID: %s", d.Id())

	return resourceHerokuPipelineRead(d, meta)
}

func resourceHerokuPipelineUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	if d.HasChange("name") {
		name := d.Get("name").(string)
		opts := heroku.PipelineUpdateOpts{
			Name: &name,
		}

		_, err := client.PipelineUpdate(context.TODO(), d.Id(), opts)
		if err != nil {
			return err
		}
	}

	return resourceHerokuPipelineRead(d, meta)
}

func resourceHerokuPipelineDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	log.Printf("[INFO] Deleting pipeline: %s", d.Id())

	_, err := client.PipelineDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting pipeline: %s", err)
	}

	d.SetId("")

	return nil
}

func resourceHerokuPipelineRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	p, err := client.PipelineInfo(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving pipeline: %s", err)
	}

	setPipelineAttributes(d, p)

	return nil
}

func setPipelineAttributes(d *schema.ResourceData, p *heroku.Pipeline) {
	d.Set("name", p.Name)

	ownerInfo := make(map[string]string)
	ownerInfo["id"] = p.Owner.ID
	ownerInfo["type"] = p.Owner.Type
	d.Set("owner", []interface{}{ownerInfo})
}
