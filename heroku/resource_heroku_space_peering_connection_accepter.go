package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	heroku "github.com/heroku/heroku-go/v5"
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
			StateContext: resourceHerokuSpacePeeringConnectionAccepterImport,
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

func resourceHerokuSpacePeeringConnectionAccepterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*Config).Api
	spaceIdentity, peeringPcxID, err := parseCompositeID(d.Id())
	if err != nil {
		return nil, err
	}

	peeringConn, err := client.PeeringInfo(ctx, spaceIdentity, peeringPcxID)
	if err != nil {
		return nil, err
	}

	d.SetId(peeringConn.PcxID)
	d.Set("space", spaceIdentity)
	setPeeringConnectionAccepterProperties(d, peeringConn)
	return []*schema.ResourceData{d}, nil
}

func resourceHerokuSpacePeeringConnectionAccepterCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	spaceIdentity := d.Get("space").(string)
	pcxID := d.Get("vpc_peering_connection_id").(string)

	// There is a lag between when a peering request is initiated from AWS's end and when it
	// appears as an option in a space's list of peering connections. In testing, this is
	// usually in the 1-3 minute range. We retry for 5 minutes so plan/apply runs that
	// create the two resources at the same time don't result in an error.
	retryError := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := client.PeeringAccept(context.TODO(), spaceIdentity, pcxID)
		if err != nil {
			return resource.RetryableError(err)
		}

		log.Printf("[INFO] Peer connection %s to %s has been accepted", pcxID, spaceIdentity)
		return nil
	})

	if retryError != nil {
		return fmt.Errorf("[ERROR] Unable to accept peer connection %s to %s", pcxID, spaceIdentity)
	}

	log.Printf("[INFO] Space ID: %s, Peering Connection ID: %s", spaceIdentity, pcxID)

	d.SetId(pcxID)

	log.Printf("[DEBUG] Waiting for connection (%s) to be accepted", d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"initiating-request", "pending", "pending-acceptance", "provisioning"},
		Target:  []string{"active"},
		Refresh: SpacePeeringConnAccepterStateRefreshFunc(client, spaceIdentity, d.Id()),
		Timeout: 20 * time.Minute,
	}

	finalPeerConn, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Space (%s) to become available: %s", d.Id(), err)
	}

	p := finalPeerConn.(*spacePeerInfo)

	d.Set("status", p.Status)
	d.Set("type", p.Type)

	return nil
}

func setPeeringConnectionAccepterProperties(d *schema.ResourceData, peeringConn *heroku.Peering) {
	d.Set("status", peeringConn.Status)
	d.Set("type", peeringConn.Type)
	d.Set("vpc_peering_connection_id", peeringConn.PcxID)
}

func resourceHerokuSpacePeeringConnectionAccepterRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

	spaceIdentity := d.Get("space").(string)

	peeringConn, err := client.PeeringInfo(context.TODO(), spaceIdentity, d.Id())
	if err != nil {
		return err
	}

	d.SetId(peeringConn.PcxID)
	setPeeringConnectionAccepterProperties(d, peeringConn)

	return nil
}

func resourceHerokuSpacePeeringConnectionAccepterDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Config).Api

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
