package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type spacePeerInfo struct {
	heroku.Peering
}

func resourceHerokuSpacePeeringConnectionAccepter() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpacePeeringConnectionAccepterCreate,
		Read:   resourceHerokuSpacePeeringConnectionAccepterRead,
		Delete: resourceHerokuSpacePeeringConnectionAccepterDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"space": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_peering_connection_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceHerokuSpacePeeringConnectionAccepterCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceIdentity := d.Get("space").(string)
	pcxID := d.Get("vpc_peering_connection_id").(string)

	_, err := client.PeeringAccept(context.TODO(), spaceIdentity, pcxID)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Space ID: %s, Peering Connection ID: %s", spaceIdentity, pcxID)

	d.SetId(pcxID)

	log.Printf("[DEBUG] Waiting for connection (%s) to be accepted", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"initiating-request", "pending-acceptance", "provisioning"},
		Target:  []string{"active"},
		Refresh: SpacePeeringConnAccepterStateRefreshFunc(client, spaceIdentity, d.Id()),
		Timeout: 20 * time.Minute,
	}

	finalPeerConn, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Space (%s) to become available: %s", d.Id(), err)
	}

	p := finalPeerConn.(*heroku.Peering)

	d.Set("status", p.Status)
	d.Set("type", p.Type)

	return nil
}

func resourceHerokuSpacePeeringConnectionAccepterRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceIdentity := d.Get("space").(string)

	peeringConn, err := client.PeeringInfo(context.TODO(), spaceIdentity, d.Id())
	if err != nil {
		return err
	}

	d.SetId(peeringConn.PcxID)
	d.Set("status", peeringConn.Status)
	d.Set("type", peeringConn.Type)
	d.Set("vpc_peering_connection_id", peeringConn.AwsVpcID)

	return nil
}

func resourceHerokuSpacePeeringConnectionAccepterDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	log.Printf("[INFO] Deleting space peering connection: %s", d.Id())

	_, err := client.PeeringDestroy(context.TODO(), d.Get("space").(string), d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// SpaceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Space peering connection. Connections go through a provisioning process.
func SpacePeeringConnAccepterStateRefreshFunc(client *heroku.Service, spaceIdentity string, pcxID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		peeringConn, err := client.PeeringInfo(context.TODO(), spaceIdentity, pcxID)
		if err != nil {
			return nil, "", err
		}

		pcx := spacePeerInfo{
			Peering: *peeringConn,
		}

		return &pcx, peeringConn.Status, nil
	}
}
