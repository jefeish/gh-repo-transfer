package dependencies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// AnalyzeOrgGovernance analyzes organizational governance dependencies
func AnalyzeOrgGovernance(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Repository rulesets are repository-level, NOT organizational governance - skip them completely
	
	// Analyze organization policies (check per repo to see which ones apply)
	if err := analyzeOrganizationPolicies(client, owner, repo, deps); err != nil {
		// Non-fatal error - policies might not be accessible
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access organization policies: %v\n", err)
		}
	}

	// Analyze organization-level repository rulesets (filter to ones that apply to this repo)
	if err := analyzeOrgLevelRepositoryRulesets(client, owner, repo, deps); err != nil {
		// Non-fatal error - rulesets might not be accessible
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access org-level repository rulesets: %v\n", err)
		}
	}

	// Analyze templates
	if err := analyzeGovernanceTemplates(client, owner, repo, deps); err != nil {
		// Non-fatal error - templates might not be accessible
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not analyze templates: %v\n", err)
		}
	}

	// Separate policies into repository policies and member privileges for JSON output
	separatePoliciesForJSON(deps)

	return nil
}

// analyzeRepositoryRulesets analyzes organization-level repository rulesets
func analyzeRepositoryRulesets(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// First, get the repository ID
	var repoInfo struct {
		ID int `json:"id"`
	}
	
	err := client.Get(fmt.Sprintf("repos/%s/%s", owner, repo), &repoInfo)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %v", err)
	}

	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Repository '%s' has ID: %d\n", repo, repoInfo.ID)
	}

	// Try repository-specific rulesets first (these include org-level rules that apply to this repo)
	if err := analyzeRepoRulesets(client, owner, repo, deps); err != nil {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access repository rulesets: %v\n", err)
		}
	}

	// Also try organization rulesets as fallback
	if err := analyzeOrgRulesets(client, owner, repo, deps); err != nil {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access organization rulesets: %v\n", err)
		}
	}

	return nil
}

// analyzeRepoRulesets analyzes rulesets that apply to this specific repository
func analyzeRepoRulesets(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Get list of rulesets first
	var rulesets []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Target      string `json:"target"`
		Enforcement string `json:"enforcement"`
		Source      string `json:"source"`
		SourceType  string `json:"source_type"`
	}

	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Checking for repository rulesets via repos/%s/%s/rulesets\n", owner, repo)
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/rulesets", owner, repo), &rulesets)
	if err != nil {
		return err // Repository rulesets not accessible
	}

	// Repository-specific rulesets are now analyzed through analyzeRepositoryPolicies
	// for proper categorization into Repository Policies vs Repository Rulesets
	return nil
}

// analyzeOrgRulesets analyzes organization-level rulesets (fallback)
func analyzeOrgRulesets(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	var rulesets []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Target      string `json:"target"`
		Enforcement string `json:"enforcement"`
		Source      string `json:"source"`
	}

	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Checking for organization-level rulesets via orgs/%s/rulesets\n", owner)
	}

	err := client.Get(fmt.Sprintf("orgs/%s/rulesets", owner), &rulesets)
	if err != nil {
		return err // Organization rulesets not accessible
	}

	// Repository rulesets are now analyzed through analyzeRepositoryPolicies for proper categorization
	return nil
}

// analyzeGovernanceTemplates analyzes issue and PR templates
func analyzeGovernanceTemplates(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Check for organization-level templates first
	if err := analyzeOrgTemplates(client, owner, deps); err != nil {
		// Non-fatal error - continue with repo-level checks
	}

	// Check for issue templates
	if err := analyzeIssueTemplates(client, owner, repo, deps); err != nil {
		// Non-fatal error
	}

	// Check for PR templates
	if err := analyzePRTemplates(client, owner, repo, deps); err != nil {
		// Non-fatal error
	}

	return nil
}

// analyzeOrgTemplates checks for organization-level issue and PR templates
func analyzeOrgTemplates(client api.RESTClient, owner string, deps *types.OrganizationalDependencies) error {
	// Check .github repository for organization-level templates
	orgRepos := []string{".github"} // Only check .github repo to avoid 404s
	
	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Checking for organization-level templates in .github repo\n")
	}
	
	for _, orgRepo := range orgRepos {
		// First check if the organization repo exists
		var repoInfo interface{}
		err := client.Get(fmt.Sprintf("repos/%s/%s", owner, orgRepo), &repoInfo)
		if err != nil {
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Organization repo %s/%s not accessible: %v\n", owner, orgRepo, err)
			}
			continue // Repo doesn't exist or not accessible
		}
		
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Found organization repo %s/%s, checking for templates\n", owner, orgRepo)
		}
		
		// Check for organization-level issue templates
		orgIssueLocations := []string{
			".github/ISSUE_TEMPLATE",
			".github/issue_template.md",
			"ISSUE_TEMPLATE",
		}
		
		for _, location := range orgIssueLocations {
			var content interface{}
			err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, orgRepo, location), &content)
			if err != nil {
				continue // Template doesn't exist at this location
			}
			
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Found organization issue template: %s in %s/%s\n", location, owner, orgRepo)
			}
			
			// Check if this is a directory (ISSUE_TEMPLATE folder) or a single file
			if location == ".github/ISSUE_TEMPLATE" || location == "ISSUE_TEMPLATE" {
				// This is likely a directory, get the files inside
				var templateFiles []struct {
					Name string `json:"name"`
					Type string `json:"type"`
				}
				
				err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, orgRepo, location), &templateFiles)
				if err == nil {
					// List each template file
					for _, file := range templateFiles {
						if file.Type == "file" {
							templateInfo := fmt.Sprintf("%s/%s", location, file.Name)
							deps.OrgGovernance.IssueTemplates = append(deps.OrgGovernance.IssueTemplates, templateInfo)
						}
					}
				} else {
					// Fallback to just the directory name if we can't read contents
					templateInfo := fmt.Sprintf("%s", location)
					deps.OrgGovernance.IssueTemplates = append(deps.OrgGovernance.IssueTemplates, templateInfo)
				}
			} else {
				// Single file template
				templateInfo := fmt.Sprintf("%s", location)
				deps.OrgGovernance.IssueTemplates = append(deps.OrgGovernance.IssueTemplates, templateInfo)
			}
			break // Found template in this repo
		}
		
		// Check for organization-level PR templates
		orgPRLocations := []string{
			".github/PULL_REQUEST_TEMPLATE",
			".github/pull_request_template.md",
			".github/PULL_REQUEST_TEMPLATE.md", 
			"PULL_REQUEST_TEMPLATE",
			"pull_request_template.md",
			"PULL_REQUEST_TEMPLATE.md",
		}
		
		for _, location := range orgPRLocations {
			var content interface{}
			err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, orgRepo, location), &content)
			if err != nil {
				continue // Template doesn't exist at this location
			}
			
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Found organization PR template: %s in %s/%s\n", location, owner, orgRepo)
			}
			
			// Check if this is a directory (PULL_REQUEST_TEMPLATE folder) or a single file
			if location == ".github/PULL_REQUEST_TEMPLATE" || location == "PULL_REQUEST_TEMPLATE" {
				// This is likely a directory, get the files inside
				var templateFiles []struct {
					Name string `json:"name"`
					Type string `json:"type"`
				}
				
				err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, orgRepo, location), &templateFiles)
				if err == nil {
					// List each template file
					for _, file := range templateFiles {
						if file.Type == "file" {
							templateInfo := fmt.Sprintf("%s/%s", location, file.Name)
							deps.OrgGovernance.PullRequestTemplates = append(deps.OrgGovernance.PullRequestTemplates, templateInfo)
						}
					}
				} else {
					// Fallback to just the directory name if we can't read contents
					templateInfo := fmt.Sprintf("%s", location)
					deps.OrgGovernance.PullRequestTemplates = append(deps.OrgGovernance.PullRequestTemplates, templateInfo)
				}
			} else {
				// Single file template
				templateInfo := fmt.Sprintf("%s", location)
				deps.OrgGovernance.PullRequestTemplates = append(deps.OrgGovernance.PullRequestTemplates, templateInfo)
			}
			break // Found template in this repo
		}
	}
	
	return nil
}

func analyzeIssueTemplates(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	templateLocations := []string{
		".github/ISSUE_TEMPLATE",
		".github/issue_template.md",
		"ISSUE_TEMPLATE.md",
	}

	for _, location := range templateLocations {
		var content interface{}
		err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, location), &content)
		if err != nil {
			continue // Template doesn't exist at this location
		}

		templateInfo := fmt.Sprintf("Issue template: %s", location)
		deps.OrgGovernance.IssueTemplates = append(deps.OrgGovernance.IssueTemplates, templateInfo)
		break // Found at least one template
	}

	return nil
}

func analyzePRTemplates(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	templateLocations := []string{
		".github/pull_request_template.md",
		".github/PULL_REQUEST_TEMPLATE.md",
		"pull_request_template.md",
		"PULL_REQUEST_TEMPLATE.md",
	}

	for _, location := range templateLocations {
		var content struct {
			Content string `json:"content"`
		}
		
		err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, location), &content)
		if err != nil {
			continue // Template doesn't exist at this location
		}

		templateInfo := fmt.Sprintf("PR template: %s", location)
		deps.OrgGovernance.PullRequestTemplates = append(deps.OrgGovernance.PullRequestTemplates, templateInfo)
		break // Found at least one template
	}

	return nil
}

// analyzeBranchProtections analyzes branch protection rules
func analyzeBranchProtections(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Get repository branches
	var branches []struct {
		Name      string `json:"name"`
		Protected bool   `json:"protected"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/branches", owner, repo), &branches)
	if err != nil {
		return err
	}

	for _, branch := range branches {
		if branch.Protected {
			// Get detailed protection information
			var protection struct {
				RequiredStatusChecks struct {
					Strict   bool     `json:"strict"`
					Contexts []string `json:"contexts"`
					Checks   []struct {
						Context string `json:"context"`
						AppID   *int   `json:"app_id"`
					} `json:"checks"`
				} `json:"required_status_checks"`
				RequiredPullRequestReviews struct {
					RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
					DismissStaleReviews         bool `json:"dismiss_stale_reviews"`
					RequireCodeOwnerReviews     bool `json:"require_code_owner_reviews"`
				} `json:"required_pull_request_reviews"`
				RequiredConversationResolution struct {
					Enabled bool `json:"enabled"`
				} `json:"required_conversation_resolution"`
			}

			err := client.Get(fmt.Sprintf("repos/%s/%s/branches/%s/protection", owner, repo, branch.Name), &protection)
			if err != nil {
				// Log the specific error for debugging
				if verbose := checkVerbose(); verbose {
					fmt.Fprintf(os.Stderr, "Could not get protection details for branch '%s': %v\n", branch.Name, err)
				}
				// Still include the protected branch but note that details are unavailable
				protectionDesc := fmt.Sprintf("%s (branch protected - API access limited)", branch.Name)
				deps.OrgGovernance.RequiredStatusChecks = append(deps.OrgGovernance.RequiredStatusChecks, protectionDesc)
				continue
			}

			// Extract and categorize protection requirements
			hasStatusChecks := len(protection.RequiredStatusChecks.Contexts) > 0 || len(protection.RequiredStatusChecks.Checks) > 0

			// Process required status checks specifically
			if hasStatusChecks {
				// Add context-based checks
				for _, context := range protection.RequiredStatusChecks.Contexts {
					checkDesc := fmt.Sprintf("%s (branch: %s, type: context check)", context, branch.Name)
					deps.OrgGovernance.RequiredStatusChecks = append(deps.OrgGovernance.RequiredStatusChecks, checkDesc)
				}
				
				// Add app-based checks
				for _, check := range protection.RequiredStatusChecks.Checks {
					checkDesc := fmt.Sprintf("%s (branch: %s, type: app check", check.Context, branch.Name)
					if check.AppID != nil {
						checkDesc += fmt.Sprintf(", app ID: %d", *check.AppID)
					}
					checkDesc += ")"
					deps.OrgGovernance.RequiredStatusChecks = append(deps.OrgGovernance.RequiredStatusChecks, checkDesc)
				}
			}

			// Handle other branch protection rules (separate from status checks)
			var otherProtections []string
			
			// Required reviews
			if protection.RequiredPullRequestReviews.RequiredApprovingReviewCount > 0 {
				reviewDetails := fmt.Sprintf("%d approving reviews", protection.RequiredPullRequestReviews.RequiredApprovingReviewCount)
				if protection.RequiredPullRequestReviews.RequireCodeOwnerReviews {
					reviewDetails += " + code owner review"
				}
				otherProtections = append(otherProtections, reviewDetails)
			}

			// Conversation resolution
			if protection.RequiredConversationResolution.Enabled {
				otherProtections = append(otherProtections, "conversation resolution")
			}

			// If there are other protections but no status checks, still record the branch
			if len(otherProtections) > 0 && !hasStatusChecks {
				protectionDesc := fmt.Sprintf("%s (protected: %s)", branch.Name, strings.Join(otherProtections, ", "))
				deps.OrgGovernance.RequiredStatusChecks = append(deps.OrgGovernance.RequiredStatusChecks, protectionDesc)
			}
		}
	}

	return nil
}

// Helper function to check if verbose mode is enabled (simplified for this context)
func checkVerbose() bool {
	// This is a simplified implementation - in a real application,
	// you might want to pass verbose as a parameter or use a global variable
	for _, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" {
			return true
		}
	}
	return false
}

// analyzeOrganizationPolicies checks for organization-level policies and settings
func analyzeOrganizationPolicies(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Checking for organization policies\n")
	}

	// Check for organization security and member management policies
	if err := checkSecurityAndMemberPolicies(client, owner, deps); err != nil {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access organization policies: %v\n", err)
		}
	}

	// Check for organization repository policies configured via GitHub UI (filtered for this repository)
	if err := analyzeRepositoryPolicies(client, owner, repo, deps); err != nil {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access repository policies: %v\n", err)
		}
		// Continue anyway - don't fail on repository policy errors
	}

	// Check for organization security policies
	if err := analyzeSecurityPolicies(client, owner, deps); err != nil {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access security policies: %v\n", err)
		}
	}

	return nil
}

// checkSecurityAndMemberPolicies checks for organization member management and security policies
func checkSecurityAndMemberPolicies(client api.RESTClient, owner string, deps *types.OrganizationalDependencies) error {
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
		deps.OrgGovernance.OrganizationPolicies = append(deps.OrgGovernance.OrganizationPolicies, policy)
	}

	if len(securityRestrictions) > 0 {
		policy := types.OrgPolicy{
			Name:         "Security Policy",
			Status:       "active",
			Restrictions: securityRestrictions,
		}
		deps.OrgGovernance.OrganizationPolicies = append(deps.OrgGovernance.OrganizationPolicies, policy)
	}

	return nil
}

// analyzeSecurityPolicies checks for organization security policies
func analyzeSecurityPolicies(client api.RESTClient, owner string, deps *types.OrganizationalDependencies) error {
	// Check for organization SECURITY.md policy
	var content interface{}
	err := client.Get(fmt.Sprintf("repos/%s/.github/contents/SECURITY.md", owner), &content)
	if err == nil {
		policy := types.OrgPolicy{
			Name:         "Organization Security Policy",
			Status:       "active", 
			Restrictions: []string{"SECURITY.md file present"},
		}
		deps.OrgGovernance.OrganizationPolicies = append(deps.OrgGovernance.OrganizationPolicies, policy)
	}

	// Check for dependabot security updates policy
	err = client.Get(fmt.Sprintf("repos/%s/.github/contents/.github/dependabot.yml", owner), &content)
	if err == nil {
		policy := types.OrgPolicy{
			Name:         "Dependabot Configuration Policy",
			Status:       "active",
			Restrictions: []string{"Automated dependency updates configured"},
		}
		deps.OrgGovernance.OrganizationPolicies = append(deps.OrgGovernance.OrganizationPolicies, policy)
	}

	return nil
}

// analyzeRepositoryPolicies checks for organization repository policies that apply to the specific repository
func analyzeRepositoryPolicies(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Checking for repository policies via branch protection and rulesets\n")
	}

	// Get repository branch protection rules as they represent repository policies
	if err := analyzeBranchProtectionPolicies(client, owner, repo, deps); err != nil {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access branch protection: %v\n", err)
		}
	}

	// Get repository-level rulesets (these are actual repository policies)
	if err := analyzeRepositoryRulesetPolicies(client, owner, repo, deps); err != nil {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Could not access repository rulesets: %v\n", err)
		}
	}

	return nil
}

// analyzeBranchProtectionPolicies extracts repository policies from branch protection rules
func analyzeBranchProtectionPolicies(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Get list of branches
	var branches []struct {
		Name string `json:"name"`
		Protection struct {
			Enabled bool `json:"enabled"`
		} `json:"protection"`
	}
	
	err := client.Get(fmt.Sprintf("repos/%s/%s/branches", owner, repo), &branches)
	if err != nil {
		return err
	}

	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Found %d branches, checking for protected branches\n", len(branches))
	}

	protectedBranches := 0
	var restrictions []string

	for _, branch := range branches {
		if branch.Protection.Enabled {
			protectedBranches++
			
			// Get detailed branch protection
			var protection struct {
				RequiredStatusChecks struct {
					Strict   bool `json:"strict"`
					Contexts []string `json:"contexts"`
					Checks   []struct {
						Context string `json:"context"`
						AppID   int    `json:"app_id"`
					} `json:"checks"`
				} `json:"required_status_checks"`
				RequiredPullRequestReviews struct {
					RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
					DismissStaleReviews         bool `json:"dismiss_stale_reviews"`
					RequireCodeOwnerReviews     bool `json:"require_code_owner_reviews"`
				} `json:"required_pull_request_reviews"`
				EnforceAdmins struct {
					Enabled bool `json:"enabled"`
				} `json:"enforce_admins"`
				RequiredLinearHistory struct {
					Enabled bool `json:"enabled"`
				} `json:"required_linear_history"`
				RequiredSignatures struct {
					Enabled bool `json:"enabled"`
				} `json:"required_signatures"`
				AllowForcePushes struct {
					Enabled bool `json:"enabled"`
				} `json:"allow_force_pushes"`
			}
			
			protErr := client.Get(fmt.Sprintf("repos/%s/%s/branches/%s/protection", owner, repo, branch.Name), &protection)
			if protErr == nil {
				if protection.RequiredPullRequestReviews.RequiredApprovingReviewCount > 0 {
					restrictions = append(restrictions, fmt.Sprintf("Branch '%s': Requires %d approving reviews", 
						branch.Name, protection.RequiredPullRequestReviews.RequiredApprovingReviewCount))
				}
				if protection.RequiredPullRequestReviews.RequireCodeOwnerReviews {
					restrictions = append(restrictions, fmt.Sprintf("Branch '%s': Code owner reviews required", branch.Name))
				}
				if protection.EnforceAdmins.Enabled {
					restrictions = append(restrictions, fmt.Sprintf("Branch '%s': Admin enforcement enabled", branch.Name))
				}
				if protection.RequiredLinearHistory.Enabled {
					restrictions = append(restrictions, fmt.Sprintf("Branch '%s': Linear history required", branch.Name))
				}
				if protection.RequiredSignatures.Enabled {
					restrictions = append(restrictions, fmt.Sprintf("Branch '%s': Signed commits required", branch.Name))
				}
				if !protection.AllowForcePushes.Enabled {
					restrictions = append(restrictions, fmt.Sprintf("Branch '%s': Force pushes disabled", branch.Name))
				}
				if len(protection.RequiredStatusChecks.Contexts) > 0 || len(protection.RequiredStatusChecks.Checks) > 0 {
					restrictions = append(restrictions, fmt.Sprintf("Branch '%s': Required status checks configured", branch.Name))
				}
			}
		}
	}

	if protectedBranches > 0 {
		policy := types.OrgPolicy{
			Name:         "Branch Protection Policy",
			Status:       "active",
			Restrictions: restrictions,
		}
		deps.OrgGovernance.OrganizationPolicies = append(deps.OrgGovernance.OrganizationPolicies, policy)
		
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Found branch protection policy with %d protected branches\n", protectedBranches)
		}
	}

	return nil
}

// analyzeRepositoryRulesetPolicies gets repository-level rulesets
func analyzeRepositoryRulesetPolicies(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Get repository rulesets
	var rulesets []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Target      string `json:"target"`
		Enforcement string `json:"enforcement"`
		Source      string `json:"source"`
		SourceType  string `json:"source_type"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/rulesets", owner, repo), &rulesets)
	if err != nil {
		return err
	}

	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Found %d repository rulesets\n", len(rulesets))
	}

	for _, ruleset := range rulesets {
		// Include rulesets that are repository-related (repository, branch, or policy-related push rules)
		if ruleset.Target == "repository" || ruleset.Target == "branch" || (ruleset.Target == "push" && strings.Contains(strings.ToLower(ruleset.Name), "policy")) {
			var restrictions []string
			
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Processing ruleset: %s (target: %s, source: %s)\n", ruleset.Name, ruleset.Target, ruleset.SourceType)
			}
			
			// Get detailed ruleset
			var detailedRuleset struct {
				Rules []struct {
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
			
			detailErr := client.Get(fmt.Sprintf("repos/%s/%s/rulesets/%d", owner, repo, ruleset.ID), &detailedRuleset)
			if detailErr == nil {
				for _, rule := range detailedRuleset.Rules {
					switch rule.Type {
					case "pull_request":
						restrictions = append(restrictions, "Pull request rules enforced")
					case "required_status_checks":
						restrictions = append(restrictions, "Required status checks enforced")
					case "deletion", "creation", "update":
						restrictions = append(restrictions, fmt.Sprintf("%s restrictions enforced", strings.Title(rule.Type)))
					case "required_linear_history":
						restrictions = append(restrictions, "Linear history required")
					case "force_push":
						restrictions = append(restrictions, "Force push restrictions")
					case "required_signatures":
						restrictions = append(restrictions, "Commit signatures required")
					default:
						restrictions = append(restrictions, fmt.Sprintf("Rule type: %s", rule.Type))
					}
				}
				
				if len(detailedRuleset.Conditions.RefName.Include) > 0 {
					restrictions = append(restrictions, fmt.Sprintf("Applies to: %s", strings.Join(detailedRuleset.Conditions.RefName.Include, ", ")))
				}
			} else {
				if verbose := checkVerbose(); verbose {
					fmt.Fprintf(os.Stderr, "Could not get detailed ruleset %d: %v\n", ruleset.ID, detailErr)
				}
				// Add basic info if detailed fetch fails
				restrictions = append(restrictions, fmt.Sprintf("Source: %s", ruleset.Source))
			}
			
			if len(restrictions) == 0 {
				restrictions = append(restrictions, fmt.Sprintf("Enforcement: %s", ruleset.Enforcement))
			}

			// Name the policy appropriately and determine category based on ruleset target
			var policyName string
			if ruleset.Target == "repository" {
				// Repository-level policies (like repository_visibility, repository_create)
				if strings.Contains(strings.ToLower(ruleset.Name), "policy") {
					policyName = ruleset.Name // Use original name if it contains "policy"
				} else {
					policyName = fmt.Sprintf("Repository Policy: %s", ruleset.Name)
				}
			} else {
				// Branch-level rulesets (like branch protection, workflows)
				policyName = fmt.Sprintf("Repository Ruleset: %s", ruleset.Name)
			}

			policy := types.OrgPolicy{
				Name:         policyName,
				Status:       ruleset.Enforcement,
				Restrictions: restrictions,
			}
			deps.OrgGovernance.OrganizationPolicies = append(deps.OrgGovernance.OrganizationPolicies, policy)
			
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Added repository policy: %s\n", policy.Name)
			}
		} else {
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Skipping ruleset: %s (target: %s, not repository-related)\n", ruleset.Name, ruleset.Target)
			}
		}
	}

	return nil
}

// separatePoliciesForJSON separates OrganizationPolicies into RepositoryPolicies and MemberPrivileges for JSON output
func separatePoliciesForJSON(deps *types.OrganizationalDependencies) {
	var repoPolicies []types.OrgPolicy
	var repoRulesets []types.OrgPolicy
	var memberPrivileges []string
	
	for _, policy := range deps.OrgGovernance.OrganizationPolicies {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Categorizing policy: %s\n", policy.Name)
		}
		
		// Use the same logic as the table formatter to categorize policies
		if isMemberPrivilegePolicy(policy) {
			// For member privileges, extract individual restrictions
			for _, restriction := range policy.Restrictions {
				memberPrivileges = append(memberPrivileges, restriction)
			}
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "  -> Categorized as Member Privilege\n")
			}
		} else if isRepositoryRuleset(policy) {
			// This is a repository ruleset (branch-level rules)
			repoRulesets = append(repoRulesets, policy)
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "  -> Categorized as Repository Ruleset\n")
			}
		} else {
			// This is a repository policy (repo-level rules)
			repoPolicies = append(repoPolicies, policy)
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "  -> Categorized as Repository Policy\n")
			}
		}
	}
	
	// Update the governance structure with separated data
	deps.OrgGovernance.RepositoryPolicies = repoPolicies
	deps.OrgGovernance.RepositoryRulesets = repoRulesets
	deps.OrgGovernance.MemberPrivileges = memberPrivileges
}

// isMemberPrivilegePolicy determines if a policy should be categorized as member privileges
// This mirrors the logic in internal/output/formatter.go
func isMemberPrivilegePolicy(policy types.OrgPolicy) bool {
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

// isRepositoryRuleset determines if a policy should be categorized as a repository ruleset
// Repository rulesets are branch-level rules while repository policies are repo-level rules
func isRepositoryRuleset(policy types.OrgPolicy) bool {
	// Check if it's explicitly a repository ruleset (starts with "Repository Ruleset:")
	if strings.HasPrefix(policy.Name, "Repository Ruleset:") {
		return true
	}
	
	// Check if restrictions indicate branch-level rules
	for _, restriction := range policy.Restrictions {
		restrictionLower := strings.ToLower(restriction)
		if strings.Contains(restrictionLower, "applies to:") ||
		   strings.Contains(restrictionLower, "deletion restrictions") ||
		   strings.Contains(restrictionLower, "pull request rules") ||
		   strings.Contains(restrictionLower, "non_fast_forward") ||
		   strings.Contains(restrictionLower, "workflows") ||
		   strings.Contains(restrictionLower, "force push") ||
		   strings.Contains(restrictionLower, "linear history") ||
		   strings.Contains(restrictionLower, "required signatures") ||
		   strings.Contains(restrictionLower, "required status checks") {
			return true
		}
	}
	
	// Repository policies typically have these rule types
	for _, restriction := range policy.Restrictions {
		restrictionLower := strings.ToLower(restriction)
		if strings.Contains(restrictionLower, "repository_visibility") ||
		   strings.Contains(restrictionLower, "repository_create") ||
		   strings.Contains(restrictionLower, "repository_delete") {
			return false // This is definitely a repository policy, not a ruleset
		}
	}
	
	return false
}

// filterPoliciesForRepository filters organization policies to only include those that apply to the specific repository
func filterPoliciesForRepository(policies []struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	PolicyType  string `json:"policy_type"`
	Scope       string `json:"scope"`
	TargetRepos []string `json:"target_repositories,omitempty"`
	TargetPattern string `json:"repository_pattern,omitempty"`
	Rules       []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"rules"`
}, repoName string) []struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	PolicyType  string `json:"policy_type"`
	Scope       string `json:"scope"`
	TargetRepos []string `json:"target_repositories,omitempty"`
	TargetPattern string `json:"repository_pattern,omitempty"`
	Rules       []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"rules"`
} {
	var filtered []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
		PolicyType  string `json:"policy_type"`
		Scope       string `json:"scope"`
		TargetRepos []string `json:"target_repositories,omitempty"`
		TargetPattern string `json:"repository_pattern,omitempty"`
		Rules       []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"rules"`
	}
	
	for _, policy := range policies {
		// Check if policy applies to this repository
		if policyAppliestoRepository(policy, repoName) {
			filtered = append(filtered, policy)
		}
	}
	
	return filtered
}

// policyAppliestoRepository checks if a specific policy applies to the given repository
func policyAppliestoRepository(policy struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	PolicyType  string `json:"policy_type"`
	Scope       string `json:"scope"`
	TargetRepos []string `json:"target_repositories,omitempty"`
	TargetPattern string `json:"repository_pattern,omitempty"`
	Rules       []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"rules"`
}, repoName string) bool {
	// If policy has no specific targeting, it applies to all repos in org
	if len(policy.TargetRepos) == 0 && policy.TargetPattern == "" && policy.Scope == "organization" {
		return true
	}
	
	// Check if repository is explicitly listed in target repos
	for _, targetRepo := range policy.TargetRepos {
		if targetRepo == repoName {
			return true
		}
	}
	
	// Check if repository matches the target pattern (basic wildcard matching)
	if policy.TargetPattern != "" {
		matched, _ := filepath.Match(policy.TargetPattern, repoName)
		return matched
	}
	
	// If policy has specific targeting but this repo doesn't match, exclude it
	if len(policy.TargetRepos) > 0 || policy.TargetPattern != "" {
		return false
	}
	
	// Default: policy applies (organization-wide policy with no specific targeting)
	return true
}

// AnalyzeRepositoryRulesets analyzes only repository rulesets (for batch optimization)
func AnalyzeRepositoryRulesets(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	return analyzeRepositoryRulesets(client, owner, repo, deps)
}

// FilterOrgRulesetsForRepository filters organization-level rulesets to show only ones that apply to the specific repository
func FilterOrgRulesetsForRepository(orgGovernance *types.OrgGovernance, repo string, deps *types.OrganizationalDependencies) error {
	// This function would be called from batch analyzer to filter org-level rulesets
	// However, the actual filtering logic is implemented in batch.go for efficiency
	return nil
}

// analyzeOrgLevelRepositoryRulesets analyzes org-level repository rulesets for single-repo analysis
func analyzeOrgLevelRepositoryRulesets(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Checking for org-level repository rulesets\n")
	}
	
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
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Failed to get org rulesets: %v\n", err)
		}
		return fmt.Errorf("failed to get org rulesets: %v", err)
	}
	
	if verbose := checkVerbose(); verbose {
		fmt.Fprintf(os.Stderr, "Found %d total rulesets\n", len(rulesets))
	}
	
	// Filter to org-level repository rulesets that apply to this specific repository
	for _, ruleset := range rulesets {
		if verbose := checkVerbose(); verbose {
			fmt.Fprintf(os.Stderr, "Checking ruleset: %s, target: %s\n", ruleset.Name, ruleset.Target)
		}
		
		if ruleset.Target == "repository" && rulesetAppliesToRepo(ruleset, repo) {
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Ruleset %s applies to repo %s\n", ruleset.Name, repo)
			}
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
			deps.OrgGovernance.OrganizationPolicies = append(deps.OrgGovernance.OrganizationPolicies, orgPolicy)
			
			if verbose := checkVerbose(); verbose {
				fmt.Fprintf(os.Stderr, "Added repository policy: %s (total: %d)\n", orgPolicy.Name, len(deps.OrgGovernance.OrganizationPolicies))
			}
		}
	}

	return nil
}

// rulesetAppliesToRepo determines if an org-level ruleset applies to the specific repository
func rulesetAppliesToRepo(ruleset struct {
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
}, repo string) bool {
	// Check if there are specific includes
	if len(ruleset.Conditions.RepositoryName.Include) > 0 {
		for _, target := range ruleset.Conditions.RepositoryName.Include {
			if target == repo || strings.Contains(target, "*") {
				// Check excludes
				for _, exclude := range ruleset.Conditions.RepositoryName.Exclude {
					if exclude == repo || strings.Contains(exclude, "*") {
						return false
					}
				}
				return true
			}
		}
		return false // Has specific includes but we're not in them
	}
	
	// No specific includes, check if we're excluded
	for _, exclude := range ruleset.Conditions.RepositoryName.Exclude {
		if exclude == repo || strings.Contains(exclude, "*") {
			return false
		}
	}
	
	// No specific targeting, applies to all repos unless it's protected-only and repo isn't protected
	// For now, assume all repos can have the ruleset applied (we don't check protected status here)
	return true
}