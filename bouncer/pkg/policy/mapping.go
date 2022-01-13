package policy

import (
	"encoding/json"
	"io/ioutil"
	"log"

	bouncer "github.com/platform9/pf9-qbert/bouncer/pkg/api"
)

// Store role mappings
type RoleMapper struct {
	mapping map[string]string
}

// Get role to group binding from the mappings file
func New() *RoleMapper {
	mapping := make(map[string]string)
	data, err := ioutil.ReadFile("/etc/config/rbac_mappings.json")
	if err == nil {
		json.Unmarshal(data, &mapping)
	}
	log.Println("Data as read from the mappings.json file :", mapping)
	return &RoleMapper{mapping}
}

// Returns list of groups that user belongs to based on user's role in keystone
func (m *RoleMapper) GetGroupsFromTokenRole(token *bouncer.KeystoneToken) []string {
	log.Println("Token roles", token.Roles)
	groups := []string{}
	if token.Roles != nil {
		for _, role := range token.Roles {
			groups = append(groups, m.mapping[role.Name])
		}
	}
	return groups
}
