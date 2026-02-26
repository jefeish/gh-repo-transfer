package dependencies

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// AnalyzeAppsIntegrationsOrgLevel analyzes organization-level apps and integrations
// This data is shared across all repositories in the organization
func AnalyzeAppsIntegrationsOrgLevel(client api.RESTClient, owner string, apps *types.OrgAppsIntegrations) error {
	// Check organization-wide app installations
	var response struct {
		TotalCount    int `json:"total_count"`
		Installations []struct {
			ID      int    `json:"id"`
			AppName string `json:"app_name"`
			AppSlug string `json:"app_slug"`
			// Add app_id for GitHub Apps
			App struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				Owner     struct {
					Login string `json:"login"`
					Type  string `json:"type"`
				} `json:"owner"`
				ExternalURL string `json:"external_url"`
			} `json:"app"`
		} `json:"installations"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/installations", owner), &response)
	if err != nil {
		return fmt.Errorf("failed to get organization app installations: %v", err)
	}

	for _, installation := range response.Installations {
		appInfo := installation.AppName
		if appInfo == "" {
			appInfo = installation.AppSlug
		}
		if appInfo == "" && installation.App.Name != "" {
			appInfo = installation.App.Name
		}

		// Check if it's an organization-wide installation
		apps.InstalledGitHubApps = append(apps.InstalledGitHubApps, appInfo+" (org-wide installation)")
	}

	return nil
}

// AnalyzeOrgGovernanceOrgLevel analyzes organization-level governance policies
// This data is shared across all repositories in the organization
func AnalyzeOrgGovernanceOrgLevel(client api.RESTClient, owner string, governance *types.OrgGovernance) error {
	// Analyze organization policies
	if err := analyzeOrganizationPoliciesOrgLevel(client, owner, governance); err != nil {
		// Non-fatal error - policies might not be accessible
		return fmt.Errorf("could not access organization policies: %v", err)
	}

	// Analyze organization-level templates
	if err := analyzeOrganizationTemplates(client, owner, governance); err != nil {
		// Non-fatal error - templates might not be accessible
		return fmt.Errorf("could not analyze organization templates: %v", err)
	}

	// Separate policies for JSON output
	separatePoliciesForJSONOrgLevel(governance)

	return nil
}

// analyzeOrganizationPoliciesOrgLevel checks for organization-level policies and settings
func analyzeOrganizationPoliciesOrgLevel(client api.RESTClient, owner string, governance *types.OrgGovernance) error {
	// Check for organization security and member management policies
	if err := checkSecurityAndMemberPoliciesOrgLevel(client, owner, governance); err != nil {
		return fmt.Errorf("failed to check security and member policies: %v", err)
	}

	// Check for organization-level repository rulesets (stored for per-repo filtering)
	if err := analyzeOrgRepositoryRulesets(client, owner, governance); err != nil {
		return fmt.Errorf("failed to analyze org repository rulesets: %v", err)
	}

	// Check for organization security policies
	if err := analyzeSecurityPoliciesOrgLevel(client, owner, governance); err != nil {
		return fmt.Errorf("failed to analyze security policies: %v", err)
	}

	return nil
}

// checkSecurityAndMemberPoliciesOrgLevel checks for organization member management and security policies
func checkSecurityAndMemberPoliciesOrgLevel(client api.RESTClient, owner string, governance *types.OrgGovernance) error {
	// Check organization settings and policies
	var orgInfo struct {
		DefaultRepositoryPermission string `json:"default_repository_permission"`
		MembersCanCreateRepos       bool   `json:"members_can_create_repositories"`
		MembersCanCreatePrivateRepos bool  `json:"members_can_create_private_repositories"`
		MembersCanCreateInternalRepos bool `json:"members_can_create_internal_repositories"`
		MembersCanCreatePublicRepos  bool  `json:"members_can_create_public_repositories"`
		MembersCanCreatePages       bool   `json:"members_can_create_pages"`
		MembersCanForkPrivateRepos  bool   `json:"members_can_fork_private_repositories"`
		WebCommitSignoffRequired    bool   `json:"web_commit_signoff_required"`
		MembersCanDeleteRepos       bool   `json:"members_can_delete_repositories"`
		MembersCanDeleteIssues      bool   `json:"members_can_delete_issues"`
		MembersCanCreateTeams       bool   `json:"members_can_create_teams"`
		TwoFactorRequirementEnabled bool   `json:"two_factor_requirement_enabled"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s", owner), &orgInfo)
	if err != nil {
		return fmt.Errorf("failed to get organization info: %v", err)
	}

	// Collect policy restrictions
	var membershipRestrictions []string
	var securityRestrictions []string

	// Membership and repository restrictions
	if !orgInfo.MembersCanCreateRepos {
		membershipRestrictions = append(membershipRestrictions, "Repository creation restricted")
	}
	if !orgInfo.MembersCanForkPrivateRepos {
		membershipRestrictions = append(membershipRestrictions, "Private repository forking restricted")
	}
	if !orgInfo.MembersCanDeleteRepos {
		membershipRestrictions = append(membershipRestrictions, "Repository deletion restricted")
	}
	if !orgInfo.MembersCanDeleteIssues {
		membershipRestrictions = append(membershipRestrictions, "Issue deletion restricted")
	}
	if !orgInfo.MembersCanCreateTeams {
		membershipRestrictions = append(membershipRestrictions, "Team creation restricted")
	}

	// Security restrictions
	if orgInfo.TwoFactorRequirementEnabled {
		securityRestrictions = append(securityRestrictions, "Two-factor authentication required")
	}
	if orgInfo.WebCommitSignoffRequired {
		securityRestrictions = append(securityRestrictions, "Web commit signoff required")
	}
	if orgInfo.DefaultRepositoryPermission != "read" {
		securityRestrictions = append(securityRestrictions, fmt.Sprintf("Default repository permission: %s", orgInfo.DefaultRepositoryPermission))
	}

	// Group restrictions under policy names with status
	if len(membershipRestrictions) > 0 {
		policy := types.OrgPolicy{
			Name:         "Member Management Policy",
			Status:       "active",
			Restrictions: membershipRestrictions,
		}
		governance.OrganizationPolicies = append(governance.OrganizationPolicies, policy)
	}

	if len(securityRestrictions) > 0 {
		policy := types.OrgPolicy{
			Name:         "Security Policy",
			Status:       "active",
			Restrictions: securityRestrictions,
		}
		governance.OrganizationPolicies = append(governance.OrganizationPolicies, policy)
	}

	return nil
}

// analyzeOrgRepositoryRulesets analyzes organization-level repository rulesets
// These are stored in org context and filtered per-repository during analysis
func analyzeOrgRepositoryRulesets(client api.RESTClient, owner string, governance *types.OrgGovernance) error {
	var rulesets []struct {
		ID         int    `json:"id"`
		Name       string `json:"name"`
		Enforcement string `json:"enforcement"`
		Source     string `json:"source"`
		Target     string `json:"target"`
		Conditions struct {
			RefName struct {
				Include []string `json:"include"`
				Exclude []string `json:"exclude"`
			} `json:"ref_name"`
			RepositoryName struct {
				Include []string `json:"include"`
				Exclude []string `json:"exclude"`
				Protected bool   `json:"protected"`
			} `json:"repository_name"`
			RepositoryProperty struct {
				Include []string `json:"include"`
				Exclude []string `json:"exclude"`
			} `json:"repository_property"`
		} `json:"conditions"`
		Rules []struct {
			Type       string                 `json:"type"`
			Parameters map[string]interface{} `json:"parameters"`
		} `json:"rules"`
	}
	
	err := client.Get(fmt.Sprintf("orgs/%s/rulesets", owner), &rulesets)
	if err != nil {
		return nil // Non-fatal - rulesets might not be accessible
	}
	
	// Store all org-level repository rulesets for later filtering
	for _, ruleset := range rulesets {
		if ruleset.Target == "repository" {
			var restrictions []string
			
			// Add enforcement status
			restrictions = append(restrictions, fmt.Sprintf("Enforcement: %s", ruleset.Enforcement))
			
			// Add targeting information
			if len(ruleset.Conditions.RepositoryName.Include) > 0 {
				restrictions = append(restrictions, fmt.Sprintf("Targets repos: %s", strings.Join(ruleset.Conditions.RepositoryName.Include, ", ")))
			} else if len(ruleset.Conditions.RepositoryName.Exclude) == 0 && !ruleset.Conditions.RepositoryName.Protected {
				// No includes and no excludes and not protected-only = targets all repositories
				restrictions = append(restrictions, "Targets repos: All repositories")
			}
			
			if len(ruleset.Conditions.RepositoryName.Exclude) > 0 {
				restrictions = append(restrictions, fmt.Sprintf("Excludes repos: %s", strings.Join(ruleset.Conditions.RepositoryName.Exclude, ", ")))
			}
			if ruleset.Conditions.RepositoryName.Protected {
				restrictions = append(restrictions, "Applies to protected repositories")
			}
			
			// Add rule summary
			if len(ruleset.Rules) > 0 {
				ruleTypes := make([]string, 0, len(ruleset.Rules))
				for _, rule := range ruleset.Rules {
					ruleTypes = append(ruleTypes, rule.Type)
				}
				restrictions = append(restrictions, fmt.Sprintf("Rules: %s", strings.Join(ruleTypes, ", ")))
			}
			
			orgPolicy := types.OrgPolicy{
				Name:         ruleset.Name,
				Status:       ruleset.Enforcement,
				Restrictions: restrictions,
			}
			governance.OrganizationPolicies = append(governance.OrganizationPolicies, orgPolicy)
		}
	}

	return nil
}

// analyzeSecurityPoliciesOrgLevel analyzes organization security policies
func analyzeSecurityPoliciesOrgLevel(client api.RESTClient, owner string, governance *types.OrgGovernance) error {
	// Check for organization SECURITY.md policy
	var content interface{}
	err := client.Get(fmt.Sprintf("repos/%s/.github/contents/SECURITY.md", owner), &content)
	if err == nil {
		policy := types.OrgPolicy{
			Name:         "Organization Security Policy",
			Status:       "active", 
			Restrictions: []string{"SECURITY.md file present"},
		}
		governance.OrganizationPolicies = append(governance.OrganizationPolicies, policy)
	}

	// Check for dependabot security updates policy
	err = client.Get(fmt.Sprintf("repos/%s/.github/contents/.github/dependabot.yml", owner), &content)
	if err == nil {
		policy := types.OrgPolicy{
			Name:         "Dependabot Configuration Policy",
			Status:       "active",
			Restrictions: []string{"Automated dependency updates configured"},
		}
		governance.OrganizationPolicies = append(governance.OrganizationPolicies, policy)
	}

	return nil
}

// analyzeOrganizationTemplates analyzes organization-level templates
func analyzeOrganizationTemplates(client api.RESTClient, owner string, governance *types.OrgGovernance) error {
	orgRepo := ".github"
	
	// Check if organization has a .github repository for templates
	var repoInfo struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	
	err := client.Get(fmt.Sprintf("repos/%s/%s", owner, orgRepo), &repoInfo)
	if err != nil {
		return nil // No organization .github repo, skip template analysis
	}

	// Check for issue templates in organization .github repo
	issueTemplateLocations := []string{
		".github/ISSUE_TEMPLATE",
		"ISSUE_TEMPLATE",
	}
	
	for _, location := range issueTemplateLocations {
		var content interface{}
		err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, orgRepo, location), &content)
		if err == nil {
			templateInfo := fmt.Sprintf("%s in %s/%s", location, owner, orgRepo)
			governance.IssueTemplates = append(governance.IssueTemplates, templateInfo)
			break
		}
	}

	// Check for PR templates in organization .github repo  
	prTemplateLocations := []string{
		".github/PULL_REQUEST_TEMPLATE",
		"PULL_REQUEST_TEMPLATE",
	}
	
	for _, location := range prTemplateLocations {
		var content interface{}
		err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, orgRepo, location), &content)
		if err == nil {
			templateInfo := fmt.Sprintf("%s in %s/%s", location, owner, orgRepo)
			governance.PullRequestTemplates = append(governance.PullRequestTemplates, templateInfo)
			break
		}
	}

	return nil
}

// separatePoliciesForJSONOrgLevel separates OrganizationPolicies into RepositoryPolicies and MemberPrivileges for JSON output
func separatePoliciesForJSONOrgLevel(governance *types.OrgGovernance) {
	var repoPolicies []types.OrgPolicy
	var memberPrivileges []string
	
	for _, policy := range governance.OrganizationPolicies {
		// Use the same logic as the table formatter to categorize policies
		if isMemberPrivilegePolicyOrgLevel(policy) {
			// For member privileges, extract individual restrictions
			for _, restriction := range policy.Restrictions {
				memberPrivileges = append(memberPrivileges, restriction)
			}
		} else {
			// This is a repository policy
			repoPolicies = append(repoPolicies, policy)
		}
	}
	
	// Update the governance structure with separated data
	governance.RepositoryPolicies = repoPolicies
	governance.MemberPrivileges = memberPrivileges
}

// isMemberPrivilegePolicyOrgLevel determines if a policy should be categorized as member privileges
func isMemberPrivilegePolicyOrgLevel(policy types.OrgPolicy) bool {
	// Policies with "policy" in the name are explicitly configured repository policies
	if strings.Contains(strings.ToLower(policy.Name), "policy") && policy.Name != "Member Management Policy" {
		return false
	}
	
	memberPrivilegeKeywords := []string{
		"member management",
		"repository creation",
		"private repository forking",
		"two-factor authentication",
		"web commit signoff",
	}
	
	policyNameLower := strings.ToLower(policy.Name)
	for _, keyword := range memberPrivilegeKeywords {
		if strings.Contains(policyNameLower, keyword) {
			return true
		}
	}
	
	// Check restrictions content
	for _, restriction := range policy.Restrictions {
		restrictionLower := strings.ToLower(restriction)
		for _, keyword := range memberPrivilegeKeywords {
			if strings.Contains(restrictionLower, keyword) {
				return true
			}
		}
	}
	
	return false
}