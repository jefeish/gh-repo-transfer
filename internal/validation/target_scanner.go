package validation

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// ScanTargetOrganization analyzes what capabilities are available in the target organization
func ScanTargetOrganization(client api.RESTClient, targetOrg string, verbose bool) (*types.TargetOrgCapabilities, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning target organization capabilities: %s\n", targetOrg)
	}

	capabilities := &types.TargetOrgCapabilities{
		Organization: targetOrg,
	}

	// Scan available GitHub Apps
	if err := scanAvailableApps(client, targetOrg, capabilities, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan apps: %v\n", err)
		}
	}

	// Scan available teams
	if err := scanAvailableTeams(client, targetOrg, capabilities, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan teams: %v\n", err)
		}
	}

	// Scan organization policies
	if err := scanRepositoryPolicies(client, targetOrg, capabilities, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan repository policies: %v\n", err)
		}
	}

	// Scan member privileges
	if err := scanMemberPrivileges(client, targetOrg, capabilities, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan member privileges: %v\n", err)
		}
	}

	// Scan organization secrets
	if err := scanAvailableSecrets(client, targetOrg, capabilities, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan secrets: %v\n", err)
		}
	}

	// Scan organization variables
	if err := scanAvailableVariables(client, targetOrg, capabilities, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan variables: %v\n", err)
		}
	}

	// Scan self-hosted runners
	if err := scanAvailableRunners(client, targetOrg, capabilities, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan runners: %v\n", err)
		}
	}

	return capabilities, nil
}

// scanAvailableApps checks what GitHub Apps are available in the target organization
func scanAvailableApps(client api.RESTClient, targetOrg string, capabilities *types.TargetOrgCapabilities, verbose bool) error {
	var installations []struct {
		AppName string `json:"app_name"`
		AppSlug string `json:"app_slug"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/installations", targetOrg), &installations)
	if err != nil {
		return fmt.Errorf("failed to get app installations: %v", err)
	}

	for _, installation := range installations {
		appInfo := installation.AppName
		if appInfo == "" {
			appInfo = installation.AppSlug
		}
		capabilities.Apps = append(capabilities.Apps, appInfo)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d apps in target org\n", len(capabilities.Apps))
	}

	return nil
}

// scanAvailableTeams checks what teams are available in the target organization
func scanAvailableTeams(client api.RESTClient, targetOrg string, capabilities *types.TargetOrgCapabilities, verbose bool) error {
	var teams []struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/teams", targetOrg), &teams)
	if err != nil {
		return fmt.Errorf("failed to get teams: %v", err)
	}

	for _, team := range teams {
		capabilities.Teams = append(capabilities.Teams, team.Name)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d teams in target org\n", len(capabilities.Teams))
	}

	return nil
}

// scanRepositoryPolicies checks for actual repository-level policies in target organization
func scanRepositoryPolicies(client api.RESTClient, targetOrg string, capabilities *types.TargetOrgCapabilities, verbose bool) error {
	var policies []types.OrgPolicy
	
	// Check for organization repository policies (these appear in the GitHub UI under Organization Settings > Repository policies)
	var repoPolicies []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		PolicyType  string `json:"policy_type"`
		Scope       string `json:"scope"`
		Rules       []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"rules"`
	}
	
	// Try the organization policies endpoint (this might be the correct one for Repository Policies in UI)
	err := client.Get(fmt.Sprintf("orgs/%s/policies", targetOrg), &repoPolicies)
	if err != nil && verbose {
		fmt.Fprintf(os.Stderr, "Could not access org policies endpoint: %v\n", err)
	}
	
	if err == nil {
		for _, policy := range repoPolicies {
			var restrictions []string
			if policy.Description != "" {
				restrictions = append(restrictions, policy.Description)
			}
			for _, rule := range policy.Rules {
				restrictions = append(restrictions, fmt.Sprintf("Rule: %s", rule.Name))
			}
			if len(restrictions) == 0 {
				restrictions = append(restrictions, fmt.Sprintf("Type: %s, Scope: %s", policy.PolicyType, policy.Scope))
			}
			
			orgPolicy := types.OrgPolicy{
				Name:         policy.Name,
				Status:       policy.Status,
				Restrictions: restrictions,
			}
			policies = append(policies, orgPolicy)
		}
	}
	
	// Also check for organization repository policies via alternative endpoint
	var altPolicies []struct {
		Name   string `json:"name"`
		Url    string `json:"url"`
		State  string `json:"state"`
		Body   string `json:"body"`
	}
	
	err = client.Get(fmt.Sprintf("orgs/%s/repository-policies", targetOrg), &altPolicies)
	if err != nil && verbose {
		fmt.Fprintf(os.Stderr, "Could not access repository-policies endpoint: %v\n", err)
	}
	
	if err == nil {
		for _, policy := range altPolicies {
			var restrictions []string
			if policy.Body != "" {
				restrictions = append(restrictions, policy.Body)
			}
			
			orgPolicy := types.OrgPolicy{
				Name:         policy.Name,
				Status:       policy.State,
				Restrictions: restrictions,
			}
			policies = append(policies, orgPolicy)
		}
	}

	// Check for organization-level rulesets that are actual repository policies (not just rulesets)
	var rulesets []struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Enforcement string `json:"enforcement"`
		Source     string `json:"source"`
		Target     string `json:"target"`
		Rules      []struct {
			Type       string `json:"type"`
			Parameters map[string]interface{} `json:"parameters"`
		} `json:"rules"`
	}
	
	err = client.Get(fmt.Sprintf("orgs/%s/rulesets", targetOrg), &rulesets)
	if err == nil {
		for _, ruleset := range rulesets {
			// Only include rulesets that are explicitly marked as policies (not just branch protection)
			if ruleset.Target == "repository" && strings.Contains(strings.ToLower(ruleset.Name), "policy") {
				var restrictions []string
				
				// Get detailed ruleset information
				var detailedRuleset struct {
					ID         int    `json:"id"`
					Name       string `json:"name"`
					Enforcement string `json:"enforcement"`
					Rules      []struct {
						Type       string `json:"type"`
						Parameters map[string]interface{} `json:"parameters"`
					} `json:"rules"`
					Conditions struct {
						RefName struct {
							Include []string `json:"include"`
							Exclude []string `json:"exclude"`
						} `json:"ref_name"`
					} `json:"conditions"`
				}
				
				detailErr := client.Get(fmt.Sprintf("orgs/%s/rulesets/%d", targetOrg, ruleset.ID), &detailedRuleset)
				if detailErr == nil {
					// Add rule details
					for _, rule := range detailedRuleset.Rules {
						switch rule.Type {
						case "pull_request":
							if params, ok := rule.Parameters["required_approving_review_count"].(float64); ok {
								restrictions = append(restrictions, fmt.Sprintf("Requires %g approving reviews", params))
							}
							if params, ok := rule.Parameters["dismiss_stale_reviews_on_push"].(bool); ok && params {
								restrictions = append(restrictions, "Dismiss stale reviews on push")
							}
							if params, ok := rule.Parameters["require_code_owner_review"].(bool); ok && params {
								restrictions = append(restrictions, "Require code owner review")
							}
						case "required_status_checks":
							if params, ok := rule.Parameters["required_status_checks"].([]interface{}); ok {
								for _, check := range params {
									if checkMap, ok := check.(map[string]interface{}); ok {
										if context, exists := checkMap["context"].(string); exists {
											restrictions = append(restrictions, fmt.Sprintf("Required status check: %s", context))
										}
									}
								}
							}
						case "creation", "deletion":
							restrictions = append(restrictions, fmt.Sprintf("%s restricted", strings.Title(rule.Type)))
						case "update":
							if params, ok := rule.Parameters["update_allows_fetch_and_merge"].(bool); ok && !params {
								restrictions = append(restrictions, "Force push disabled")
							}
						case "required_linear_history":
							restrictions = append(restrictions, "Linear history required")
						case "force_push":
							restrictions = append(restrictions, "Force push disabled")
						case "required_signatures":
							restrictions = append(restrictions, "Signed commits required")
						case "branch_name_pattern":
							restrictions = append(restrictions, "Branch naming pattern enforced")
						case "commit_message_pattern":
							restrictions = append(restrictions, "Commit message pattern enforced")
						case "commit_author_email_pattern":
							restrictions = append(restrictions, "Commit author email pattern enforced")
						case "committer_email_pattern":
							restrictions = append(restrictions, "Committer email pattern enforced")
						default:
							restrictions = append(restrictions, fmt.Sprintf("Rule: %s", rule.Type))
						}
					}
					
					// Add branch conditions if present
					if len(detailedRuleset.Conditions.RefName.Include) > 0 {
						restrictions = append(restrictions, fmt.Sprintf("Applies to branches: %s", strings.Join(detailedRuleset.Conditions.RefName.Include, ", ")))
					}
				} else {
					// Fallback to basic rule types if detailed fetch fails
					for _, rule := range ruleset.Rules {
						restrictions = append(restrictions, fmt.Sprintf("Rule: %s", rule.Type))
					}
				}
				
				// Only add enforcement status if no actual rules were found
				if len(restrictions) == 0 {
					restrictions = append(restrictions, fmt.Sprintf("Enforcement: %s", ruleset.Enforcement))
				}
				
				policy := types.OrgPolicy{
					Name:         ruleset.Name,
					Status:       ruleset.Enforcement,
					Restrictions: restrictions,
				}
				policies = append(policies, policy)
			}
		}
	}

	// Check for organization security policy
	var content interface{}
	err = client.Get(fmt.Sprintf("repos/%s/.github/contents/SECURITY.md", targetOrg), &content)
	if err == nil {
		policy := types.OrgPolicy{
			Name:         "Organization Security Policy",
			Status:       "active",
			Restrictions: []string{"SECURITY.md file present"},
		}
		policies = append(policies, policy)
	}

	// Check for dependabot configuration policy
	err = client.Get(fmt.Sprintf("repos/%s/.github/contents/.github/dependabot.yml", targetOrg), &content)
	if err == nil {
		policy := types.OrgPolicy{
			Name:         "Dependabot Configuration Policy", 
			Status:       "active",
			Restrictions: []string{"Automated dependency updates configured"},
		}
		policies = append(policies, policy)
	}

	capabilities.RepositoryPolicies = policies
	
	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d repository policies in target org\n", len(capabilities.RepositoryPolicies))
	}

	return nil
}

// scanMemberPrivileges checks organization-wide member privilege settings
func scanMemberPrivileges(client api.RESTClient, targetOrg string, capabilities *types.TargetOrgCapabilities, verbose bool) error {
	// Check organization settings for member privileges
	var orgInfo struct {
		MembersCanCreateRepos       bool   `json:"members_can_create_repositories"`
		MembersCanForkPrivateRepos  bool   `json:"members_can_fork_private_repositories"`
		TwoFactorRequirementEnabled bool   `json:"two_factor_requirement_enabled"`
		WebCommitSignoffRequired    bool   `json:"web_commit_signoff_required"`
		DefaultRepositoryPermission string `json:"default_repository_permission"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s", targetOrg), &orgInfo)
	if err != nil {
		return fmt.Errorf("failed to get organization info: %v", err)
	}

	// Store member privilege settings
	var restrictions []string
	if !orgInfo.MembersCanCreateRepos {
		restrictions = append(restrictions, "Repository creation restricted")
	}
	if !orgInfo.MembersCanForkPrivateRepos {
		restrictions = append(restrictions, "Private repository forking restricted")
	}
	if orgInfo.TwoFactorRequirementEnabled {
		restrictions = append(restrictions, "Two-factor authentication required")
	}
	if orgInfo.WebCommitSignoffRequired {
		restrictions = append(restrictions, "Web commit signoff required")
	}

	capabilities.MemberPrivileges = types.OrgMemberPrivileges{
		CanCreateRepos:          orgInfo.MembersCanCreateRepos,
		CanForkPrivateRepos:     orgInfo.MembersCanForkPrivateRepos,
		TwoFactorRequired:       orgInfo.TwoFactorRequirementEnabled,
		WebCommitSignoffRequired: orgInfo.WebCommitSignoffRequired,
		DefaultPermission:       orgInfo.DefaultRepositoryPermission,
		RestrictionsActive:      restrictions,
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d member privilege restrictions in target org\n", len(capabilities.MemberPrivileges.RestrictionsActive))
	}

	return nil
}

// scanAvailableSecrets checks organization secrets in the target organization
func scanAvailableSecrets(client api.RESTClient, targetOrg string, capabilities *types.TargetOrgCapabilities, verbose bool) error {
	var secrets struct {
		Secrets []struct {
			Name string `json:"name"`
		} `json:"secrets"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/actions/secrets", targetOrg), &secrets)
	if err != nil {
		return fmt.Errorf("failed to get secrets: %v", err)
	}

	for _, secret := range secrets.Secrets {
		capabilities.Secrets = append(capabilities.Secrets, secret.Name)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d secrets in target org\n", len(capabilities.Secrets))
	}

	return nil
}

// scanAvailableVariables checks organization variables in the target organization
func scanAvailableVariables(client api.RESTClient, targetOrg string, capabilities *types.TargetOrgCapabilities, verbose bool) error {
	var variables struct {
		Variables []struct {
			Name string `json:"name"`
		} `json:"variables"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/actions/variables", targetOrg), &variables)
	if err != nil {
		return fmt.Errorf("failed to get variables: %v", err)
	}

	for _, variable := range variables.Variables {
		capabilities.Variables = append(capabilities.Variables, variable.Name)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d variables in target org\n", len(capabilities.Variables))
	}

	return nil
}

// scanAvailableRunners checks self-hosted runners in the target organization
func scanAvailableRunners(client api.RESTClient, targetOrg string, capabilities *types.TargetOrgCapabilities, verbose bool) error {
	var runners struct {
		Runners []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"runners"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/actions/runners", targetOrg), &runners)
	if err != nil {
		return fmt.Errorf("failed to get runners: %v", err)
	}

	for _, runner := range runners.Runners {
		if strings.ToLower(runner.Status) == "online" {
			capabilities.Runners = append(capabilities.Runners, runner.Name)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d active runners in target org\n", len(capabilities.Runners))
	}

	return nil
}