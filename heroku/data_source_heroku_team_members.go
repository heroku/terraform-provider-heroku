package heroku

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	heroku "github.com/heroku/heroku-go/v5"
)

var (
	TeamMemberRoles = []string{"admin", "member", "viewer", "collaborator", "owner"}
)

func dataSourceHerokuTeamMembers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceHerokuTeamMembersRead,
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
					Type:         schema.TypeString,
					Sensitive:    true,
					ValidateFunc: validation.StringInSlice(TeamMemberRoles, false),
				},
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
							Type:     schema.TypeString,
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

func dataSourceHerokuTeamMembersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Config).Api
	roles := make([]string, 0)

	teamName := d.Get("team").(string)
	rolesRaw := d.Get("roles").([]interface{})

	for _, r := range rolesRaw {
		roles = append(roles, r.(string))
	}

	teamMembers, listErr := client.TeamMemberList(context.TODO(), teamName,
		&heroku.ListRange{
			Field:      "id",
			Max:        1000,
			Descending: false,
		},
	)
	if listErr != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Unable to retrieve members for team %s", teamName),
			Detail:   listErr.Error(),
		})
		return diags
	}

	if len(teamMembers) == 0 {
		return diag.Errorf("no members found for team %s", teamName)
	}

	d.SetId(teamName)

	members := make([]map[string]interface{}, 0)

	for _, m := range teamMembers {
		if SliceContainsString(TeamMemberRoles, *m.Role) {
			member := make(map[string]interface{})
			member["team_member_id"] = m.ID
			member["user_id"] = m.User.ID
			member["email"] = m.User.Email
			member["role"] = *m.Role
			member["federated"] = m.Federated
			member["two_factor_authentication"] = m.TwoFactorAuthentication
			members = append(members, member)
		}
	}

	d.Set("team", teamName)
	d.Set("roles", roles)
	d.Set("members", members)

	return diags
}
