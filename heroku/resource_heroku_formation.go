package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

// herokuFormation is a value type used to hold the details
// of a formation
type herokuFormation struct {
	AppName  string
	Command  string
	Quantity int
	Size     string
	Type     string
}

type formation struct {
	Id string // Id of the resource

	Formation *herokuFormation
	Client    *heroku.Service // Client to interact with the heroku API
}

func resourceHerokuFormation() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuFormationCreate,
		Read:   resourceHerokuFormationRead,
		Update: resourceHerokuFormationUpdate,
		Delete: resourceHerokuFormationDelete,

		Importer: &schema.ResourceImporter{
			State: resourceHerokuFormationImport,
		},

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
			},

			"quantity": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"size": {
				Type:      schema.TypeString,
				Required:  true,
				StateFunc: formatSize,
			},
		},
	}
}

func resourceHerokuFormationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	appName := getAppName(d)

	formation, err := resourceHerokuFormationRetrieve(d.Id(), appName, client)

	if err != nil {
		return err
	}

	d.Set("app", formation.Formation.AppName)
	d.Set("type", formation.Formation.Type)
	d.Set("quantity", formation.Formation.Quantity)
	d.Set("size", formation.Formation.Size)

	return nil
}

// this is more or less the same as update but i believe terraform needs a 'Create' method
func resourceHerokuFormationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	opts := heroku.FormationUpdateOpts{}

	appName := getAppName(d)

	// check if appName is valid
	_, err := doesHerokuAppExist(appName, client)
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("size"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] Size: %s", vs)
		opts.Size = &vs
	}

	if v, ok := d.GetOk("quantity"); ok {
		vs := v.(int)
		log.Printf("[DEBUG] Quantity: %s", vs)
		opts.Quantity = &vs
	}

	log.Printf("[DEBUG] Creating Heroku formation...")
	f, err := client.FormationUpdate(context.TODO(), appName, getFormationType(d), opts)
	if err != nil {
		return err
	}

	d.SetId(f.ID)
	log.Printf("[INFO] Formation ID: %s", d.Id())

	return resourceHerokuFormationRead(d, meta)
}

func resourceHerokuFormationUpdate(d *schema.ResourceData, meta interface{}) error {
	// Enable Partial state mode and what we successfully committed
	d.Partial(true)

	client := meta.(*heroku.Service)
	opts := heroku.FormationUpdateOpts{}

	if d.HasChange("size") {
		v := d.Get("size").(string)
		log.Printf("[DEBUG] New Size: %s", v)
		opts.Size = &v
	}

	if d.HasChange("quantity") {
		v := d.Get("quantity").(int)
		log.Printf("[DEBUG] New Quantity: %s", v)
		opts.Quantity = &v
	}

	appName := getAppName(d)

	// check if appName is valid
	_, err := doesHerokuAppExist(appName, client)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updating Heroku formation...")
	updatedFormation, err := client.FormationUpdate(context.TODO(),
		appName, getFormationType(d), opts)

	if err != nil {
		return err
	}
	d.SetId(updatedFormation.Type)

	d.Partial(false)

	return resourceHerokuFormationRead(d, meta)
}

func resourceHerokuFormationDelete(d *schema.ResourceData, meta interface{}) error {
	// there is no DELETE endpoint for the formation resource
	log.Printf("[INFO] There is no DELETE for formation resource so this is a noop")
	return nil
}

func getFormationType(d *schema.ResourceData) string {
	var formationType string
	if v, ok := d.GetOk("type"); ok {
		vs := v.(string)
		log.Printf("[DEBUG] Formation type: %s", vs)
		formationType = vs
	}

	return formationType
}

func resourceHerokuFormationRetrieve(id string, appName string, client *heroku.Service) (*formation, error) {
	formation := formation{Id: id, Client: client}

	err := formation.Update(appName)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving formation: %s", err)
	}

	return &formation, nil
}

func (f *formation) Update(appName string) error {
	var errs []error
	var err error

	log.Printf("[INFO] The formation's app name is %s", appName)
	log.Printf("[INFO] f.Id is %s", f.Id)

	formation, err := f.Client.FormationInfo(context.TODO(), appName, f.Id)
	if err != nil {
		errs = append(errs, err)
	} else {
		f.Formation = &herokuFormation{}
		f.Formation.AppName = formation.App.Name
		f.Formation.Command = formation.Command
		f.Formation.Quantity = formation.Quantity
		f.Formation.Size = formation.Size
		f.Formation.Type = formation.Type
	}

	return nil
}

func resourceHerokuFormationImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client := meta.(*heroku.Service)

	app, formationType := parseCompositeID(d.Id())

	formation, err := client.FormationInfo(context.Background(), app, formationType)
	if err != nil {
		return nil, err
	}

	d.SetId(formation.ID)
	d.Set("app", formation.App.Name)
	d.Set("type", formation.Type)
	d.Set("quantity", formation.Quantity)
	d.Set("size", formation.Size)

	return []*schema.ResourceData{d}, nil
}

func doesHerokuAppExist(appName string, client *heroku.Service) (*heroku.App, error) {
	log.Printf("app is %s", appName)
	app, err := client.AppInfo(context.TODO(), appName)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("[ERROR] Your app does not exist")
	}
	return app, nil
}

func formatSize(quant interface{}) string {
	if quant == nil || quant == (*string)(nil) {
		return ""
	}

	var rawQuant string
	switch quant.(type) {
	case string:
		rawQuant = quant.(string)
	case *string:
		rawQuant = *quant.(*string)
	default:
		return ""
	}

	// the size will generally be something like "standard-1x" or "Private-M"
	// the goal will to be split the size, capitalize zero index and then toUpper the first index
	// to get the end result of "Standard-1X". I'm doing this because the API PATCH accepts lowercase
	// but returns the formatted which causes state file issues.
	splittedString := strings.Split(rawQuant, "-")
	var formattedSlice []string

	for i := 0; i < 2; i++ { // there will only ever be two elements
		if i == 0 {
			// capitalize the first element
			formattedSlice = append(formattedSlice, strings.Title(splittedString[i]))
		} else {
			formattedSlice = append(formattedSlice, strings.ToUpper(splittedString[i]))
		}
	}

	return strings.Join(formattedSlice, "-")
}
