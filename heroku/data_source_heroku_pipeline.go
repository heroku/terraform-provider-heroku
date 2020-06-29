package heroku

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceHerokuPipeline() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceHerokuPipelineRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"owner_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceHerokuPipelineRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api

	name := d.Get("name").(string)

	pipeline, getErr := client.PipelineInfo(context.TODO(), name)
	if getErr != nil {
		return getErr
	}

	d.SetId(pipeline.ID)
	d.Set("name", pipeline.Name)
	d.Set("owner_id", pipeline.Owner.ID)
	d.Set("owner_type", pipeline.Owner.Type)

	return nil
}
