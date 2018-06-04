package heroku

import (
	"context"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceHerokuSpacePeeringInfo() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceHerokuSpacePeeringInfoRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"aws_account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"aws_region": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_cidr": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"dyno_cidr_blocks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"unavailable_cidr_blocks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceHerokuSpacePeeringInfoRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*heroku.Service)

	name := d.Get("name").(string)
	d.SetId(name)

	peeringInfo, err := client.PeeringInfoInfo(context.TODO(), name)
	if err != nil {
		return err
	}

	d.Set("aws_account_id", peeringInfo.AwsAccountID)
	d.Set("aws_region", peeringInfo.AwsRegion)
	d.Set("vpc_id", peeringInfo.VpcID)
	d.Set("vpc_cidr", peeringInfo.VpcCidr)
	d.Set("dyno_cidr_blocks", peeringInfo.DynoCidrBlocks)
	d.Set("unavailable_cidr_blocks", peeringInfo.UnavailableCidrBlocks)

	return nil
}
