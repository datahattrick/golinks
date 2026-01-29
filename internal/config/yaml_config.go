package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLConfig represents the structure of the config.yaml file.
// Complex hierarchical config that's easier to manage in YAML than env vars.
type YAMLConfig struct {
	Organizations  []OrganizationConfig `yaml:"organizations"`
	Groups         []GroupConfig        `yaml:"groups"`
	AutoAssignment AutoAssignmentConfig `yaml:"auto_assignment"`
	Defaults       DefaultsConfig       `yaml:"defaults"`
}

// OrganizationConfig defines an organization in the YAML config.
type OrganizationConfig struct {
	Slug    string   `yaml:"slug"`
	Name    string   `yaml:"name"`
	Domains []string `yaml:"domains,omitempty"` // For email-based auto-assignment
}

// GroupConfig defines a group in the YAML config.
type GroupConfig struct {
	Slug         string `yaml:"slug"`
	Name         string `yaml:"name"`
	Tier         int    `yaml:"tier"`                    // 1-99, priority for link resolution
	Organization string `yaml:"organization,omitempty"` // Organization slug
	Parent       string `yaml:"parent,omitempty"`       // Parent group slug
}

// AutoAssignmentConfig defines how users are auto-assigned to groups.
type AutoAssignmentConfig struct {
	Claim    string              `yaml:"claim"`    // OIDC claim name (e.g., "groups", "roles")
	Mappings map[string][]string `yaml:"mappings"` // Claim value -> group slugs
}

// DefaultsConfig defines default settings.
type DefaultsConfig struct {
	GroupRole  string `yaml:"group_role"`  // Default role when adding users to groups
	UserActive bool   `yaml:"user_active"` // Whether new users are active by default
	AutoCreate bool   `yaml:"auto_create"` // Create missing orgs/groups on reference
}

// LoadYAMLConfig loads the YAML configuration file.
// Path is determined by CONFIG_FILE env var, defaulting to "config.yaml".
// Returns nil without error if the config file doesn't exist.
func LoadYAMLConfig() (*YAMLConfig, error) {
	path := getEnv("CONFIG_FILE", "config.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file is optional
			return nil, nil
		}
		return nil, err
	}

	var cfg YAMLConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Defaults.GroupRole == "" {
		cfg.Defaults.GroupRole = "member"
	}

	return &cfg, nil
}

// GetOrganizationBySlug finds an organization by its slug.
func (c *YAMLConfig) GetOrganizationBySlug(slug string) *OrganizationConfig {
	if c == nil {
		return nil
	}
	for i := range c.Organizations {
		if c.Organizations[i].Slug == slug {
			return &c.Organizations[i]
		}
	}
	return nil
}

// GetOrganizationByDomain finds an organization by email domain.
func (c *YAMLConfig) GetOrganizationByDomain(domain string) *OrganizationConfig {
	if c == nil {
		return nil
	}
	for i := range c.Organizations {
		for _, d := range c.Organizations[i].Domains {
			if d == domain {
				return &c.Organizations[i]
			}
		}
	}
	return nil
}

// GetGroupBySlug finds a group by its slug.
func (c *YAMLConfig) GetGroupBySlug(slug string) *GroupConfig {
	if c == nil {
		return nil
	}
	for i := range c.Groups {
		if c.Groups[i].Slug == slug {
			return &c.Groups[i]
		}
	}
	return nil
}

// GetGroupsForOrganization returns all groups belonging to an organization.
func (c *YAMLConfig) GetGroupsForOrganization(orgSlug string) []GroupConfig {
	if c == nil {
		return nil
	}
	var groups []GroupConfig
	for _, g := range c.Groups {
		if g.Organization == orgSlug {
			groups = append(groups, g)
		}
	}
	return groups
}

// GetGroupsForClaimValue returns group slugs for an OIDC claim value.
func (c *YAMLConfig) GetGroupsForClaimValue(value string) []string {
	if c == nil || c.AutoAssignment.Mappings == nil {
		return nil
	}
	return c.AutoAssignment.Mappings[value]
}
