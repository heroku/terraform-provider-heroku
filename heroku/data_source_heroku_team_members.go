package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

const (
	RoleAll = "all"
)

func dataSourceHerokuTeamMembers() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceHerokuTeamMembersRead,
		Schema: map[string]*schema.Schema{
			"team": {
				Type:     schema.TypeString,
				Required: true,
			},

			"roles": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type:      schema.TypeString,
					Sensitive: true,
					ValidateFunc: validation.StringInSlice(
						[]string{"admin", "member", "viewer", "collaborator", RoleAll}, false),
				},
				ValidateFunc: validateTeamMemberRoles,
			},

			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"team_member_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"user_id": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"email": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"role": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"federated": {
							Type:     schema.TypeBool,
							Computed: true,
						},

						"two_factor_authentication": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func validateTeamMemberRoles(v interface{}, k string) (ws []string, errors []error) {
	// Check if 'all' is set as a role along with other ones. If this is the case,
	// return an error to set just 'all' or one or more of the other roles.
	isAllRoleSet := false
	rolesRaw := v.([]interface{})

	for _, r := range rolesRaw {
		if r.(string) == RoleAll {
			isAllRoleSet = true
			continue
		}

		if isAllRoleSet {
			errors = append(errors, fmt.Errorf("please set the roles attribute to either 'all' or one or more other roles"))
		}
	}

	return ws, errors
}

func dataSourceHerokuTeamMembersRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Config).Api
	roles := make([]string, 0)

	teamName := d.Get("name").(string)
	rolesRaw := d.Get("roles").([]interface{})

	for _, r := range rolesRaw {
		roles = append(roles, r.(string))
	}

	teamMembers, err := client.TeamMemberList(context.TODO(), teamName,
		&heroku.ListRange{Max: 1000, Descending: false},
	)
	if err != nil {
		return err
	}

	d.SetId(teamName)

	for _, m := range teamMembers {
		m.User
	}

	var setErr error
	setErr = d.Set("name", team.Name)
	setErr = d.Set("default", team.Default)
	setErr = d.Set("membership_limit", team.MembershipLimit)
	setErr = d.Set("provisioned_licenses", team.ProvisionedLicenses)
	setErr = d.Set("type", team.Type)

	return setErr
}
