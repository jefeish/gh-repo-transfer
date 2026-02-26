package batch

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/dependencies"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// OrganizationContext holds cached organization-level data that can be shared across repositories
type OrganizationContext struct {
	Organization     string
	Apps             types.OrgAppsIntegrations
	Governance       types.OrgGovernance
	SecurityCampaigns []string
	OrganizationRoles []string
	OrgInfo          struct {
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
	mutex sync.RWMutex
}

// BatchAnalysisResult represents the result for a single repository
type BatchAnalysisResult struct {
	Repository string
	Result     *types.OrganizationalDependencies
	Error      error
}

// BatchAnalyzer handles batch analysis of multiple repositories
type BatchAnalyzer struct {
	client  api.RESTClient
	verbose bool
	orgCtx  *OrganizationContext
}

// NewBatchAnalyzer creates a new batch analyzer
func NewBatchAnalyzer(client api.RESTClient, verbose bool) *BatchAnalyzer {
	return &BatchAnalyzer{
		client:  client,
		verbose: verbose,
	}
}

// AnalyzeRepositories performs batch analysis on multiple repositories in the same organization
func (ba *BatchAnalyzer) AnalyzeRepositories(repos []string) ([]BatchAnalysisResult, error) {
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories provided")
	}

	// Extract organization from the first repository
	// Assuming all repos are in the same organization
	owner, _, err := parseRepository(repos[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository %s: %v", repos[0], err)
	}

	// Step 1: Load organization-level context (cached across all repos)
	if ba.verbose {
		fmt.Fprintf(os.Stderr, "Loading organization context for: %s\n", owner)
	}
	
	orgCtx, err := ba.loadOrganizationContext(owner)
	if err != nil {
		return nil, fmt.Errorf("failed to load organization context: %v", err)
	}
	ba.orgCtx = orgCtx

	// Step 2: Analyze each repository with shared org context
	results := make([]BatchAnalysisResult, len(repos))
	
	// Use goroutines for parallel repository analysis
	var wg sync.WaitGroup
	for i, repo := range repos {
		wg.Add(1)
		go func(index int, repository string) {
			defer wg.Done()
			
			if ba.verbose {
				fmt.Fprintf(os.Stderr, "Analyzing repository: %s\n", repository)
			}
			
			result, err := ba.analyzeRepositoryWithContext(repository)
			results[index] = BatchAnalysisResult{
				Repository: repository,
				Result:     result,
				Error:      err,
			}
		}(i, repo)
	}
	
	wg.Wait()
	
	if ba.verbose {
		fmt.Fprintf(os.Stderr, "Batch analysis completed for %d repositories\n", len(repos))
	}
	
	return results, nil
}

// loadOrganizationContext loads and caches organization-level data
func (ba *BatchAnalyzer) loadOrganizationContext(owner string) (*OrganizationContext, error) {
	ctx := &OrganizationContext{
		Organization: owner,
	}

	var wg sync.WaitGroup
	var errs []error
	var errMutex sync.Mutex

	addError := func(err error) {
		if err != nil {
			errMutex.Lock()
			errs = append(errs, err)
			errMutex.Unlock()
		}
	}

	// Load Apps & Integrations (organization-level)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ba.verbose {
			fmt.Fprintf(os.Stderr, "Loading organization apps...\n")
		}
		err := ba.loadOrganizationApps(owner, ctx)
		addError(err)
	}()

	// Load Organization Governance (organization-level - Member Privileges, Templates)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ba.verbose {
			fmt.Fprintf(os.Stderr, "Loading organization governance (member privileges, templates)...\n")
		}
		err := ba.loadOrganizationGovernance(owner, ctx)
		addError(err)
	}()

	// Load Organization Info (organization-level)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ba.verbose {
			fmt.Fprintf(os.Stderr, "Loading organization info...\n")
		}
		err := ba.loadOrganizationInfo(owner, ctx)
		addError(err)
	}()

	// Load Security Campaigns (organization-level)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ba.verbose {
			fmt.Fprintf(os.Stderr, "Loading security campaigns...\n")
		}
		err := ba.loadSecurityCampaigns(owner, ctx)
		addError(err)
	}()

	wg.Wait()

	// Return first error if any occurred
	if len(errs) > 0 {
		if ba.verbose {
			for _, err := range errs {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
		// Don't fail completely for organization context loading errors
		// as these are non-fatal warnings
	}

	return ctx, nil
}

// analyzeRepositoryWithContext analyzes a single repository using the shared organization context
func (ba *BatchAnalyzer) analyzeRepositoryWithContext(repoSpec string) (*types.OrganizationalDependencies, error) {
	owner, repo, err := parseRepository(repoSpec)
	if err != nil {
		return nil, err
	}

	deps := &types.OrganizationalDependencies{
		Repository: repoSpec,
	}

	// Copy organization-level data from context
	ba.orgCtx.mutex.RLock()
	deps.AppsIntegrations.InstalledGitHubApps = ba.orgCtx.Apps.InstalledGitHubApps
	// Copy org-level governance (Member Privileges, Templates)
	deps.OrgGovernance = ba.orgCtx.Governance
	ba.orgCtx.mutex.RUnlock()

	var wg sync.WaitGroup
	var errs []error
	var errMutex sync.Mutex

	addError := func(err error) {
		if err != nil {
			errMutex.Lock()
			errs = append(errs, err)
			errMutex.Unlock()
		}
	}

	// Repository-specific analyses (these must be done per repo)
	
	// 1. Code Dependencies (repository-specific)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dependencies.AnalyzeCodeDependencies(ba.client, owner, repo, deps)
		if err != nil && ba.verbose {
			addError(fmt.Errorf("code dependencies: %v", err))
		}
	}()

	// 2. CI/CD Dependencies (repository-specific)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dependencies.AnalyzeActionsCIDependencies(ba.client, owner, repo, deps)
		if err != nil && ba.verbose {
			addError(fmt.Errorf("CI/CD dependencies: %v", err))
		}
	}()

	// 3. Access Control (repository-specific parts)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dependencies.AnalyzeAccessPermissions(ba.client, owner, repo, deps)
		if err != nil && ba.verbose {
			addError(fmt.Errorf("access permissions: %v", err))
		}
	}()

	// 4. Security & Compliance (repository-specific)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dependencies.AnalyzeSecurityCompliance(ba.client, owner, repo, deps)
		if err != nil && ba.verbose {
			addError(fmt.Errorf("security compliance: %v", err))
		}
	}()

	// 5. Repository-specific Governance (Repository Policies and Repository Rulesets only)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := ba.analyzeRepositorySpecificGovernance(owner, repo, deps)
		if err != nil && ba.verbose {
			addError(fmt.Errorf("repository governance: %v", err))
		}
	}()

	wg.Wait()

	// Log warnings but don't fail
	if len(errs) > 0 && ba.verbose {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "Warning for %s: %v\n", repoSpec, err)
		}
	}

	return deps, nil
}

// Helper functions for loading organization-level data
func (ba *BatchAnalyzer) loadOrganizationApps(owner string, ctx *OrganizationContext) error {
	return dependencies.AnalyzeAppsIntegrationsOrgLevel(ba.client, owner, &ctx.Apps)
}

func (ba *BatchAnalyzer) loadOrganizationGovernance(owner string, ctx *OrganizationContext) error {
	return dependencies.AnalyzeOrgGovernanceOrgLevel(ba.client, owner, &ctx.Governance)
}

func (ba *BatchAnalyzer) loadOrganizationInfo(owner string, ctx *OrganizationContext) error {
	return ba.client.Get(fmt.Sprintf("orgs/%s", owner), &ctx.OrgInfo)
}

// analyzeRepositorySpecificGovernance analyzes only the repository-specific governance parts
func (ba *BatchAnalyzer) analyzeRepositorySpecificGovernance(owner, repo string, deps *types.OrganizationalDependencies) error {
	// Filter organization-level rulesets to find ones that target this specific repository
	if ba.orgCtx != nil {
		if err := ba.filterOrgRulesetsForRepo(owner, repo, &ba.orgCtx.Governance, deps); err != nil {
			if ba.verbose && !strings.Contains(err.Error(), "404") {
				fmt.Fprintf(os.Stderr, "Could not filter org rulesets for %s: %v\n", repo, err)
			}
		}
	}

	return nil
}

func (ba *BatchAnalyzer) loadSecurityCampaigns(owner string, ctx *OrganizationContext) error {
	var campaigns []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	err := ba.client.Get(fmt.Sprintf("orgs/%s/security/campaigns", owner), &campaigns)
	if err != nil {
		return err
	}
	
	for _, campaign := range campaigns {
		ctx.SecurityCampaigns = append(ctx.SecurityCampaigns, campaign.Name)
	}
	return nil
}

// filterOrgRulesetsForRepo filters organization-level rulesets to find ones that target the specific repository
func (ba *BatchAnalyzer) filterOrgRulesetsForRepo(owner, repo string, orgGovernance *types.OrgGovernance, deps *types.OrganizationalDependencies) error {
	// Check each org-level ruleset to see if it applies to this repository
	for _, policy := range orgGovernance.OrganizationPolicies {
		// Parse ruleset details from the policy (stored in restrictions)
		if ba.rulesetAppliesToRepo(policy, repo) {
			// This org-level ruleset applies to this repository
			deps.OrgGovernance.RepositoryPolicies = append(deps.OrgGovernance.RepositoryPolicies, types.OrgPolicy{
				Name:         policy.Name,
				Status:       policy.Status,
				Restrictions: policy.Restrictions,
			})
		}
	}
	return nil
}

// rulesetAppliesToRepo determines if an org-level ruleset applies to the specific repository
func (ba *BatchAnalyzer) rulesetAppliesToRepo(policy types.OrgPolicy, repo string) bool {
	// Check targeting information in the restrictions
	for _, restriction := range policy.Restrictions {
		// Check if this ruleset includes specific repositories
		if strings.Contains(restriction, "Targets repos:") {
			// Extract repository list and check if our repo is included
			targets := strings.TrimPrefix(restriction, "Targets repos: ")
			
			// Handle "All repositories" case
			if targets == "All repositories" {
				return true
			}
			
			// Handle specific repository lists
			targetList := strings.Split(targets, ", ")
			for _, target := range targetList {
				// Support wildcards and exact matches
				if target == repo || strings.Contains(target, "*") {
					return true
				}
			}
			return false // Explicitly targets repos but not this one
		}
		
		// Check if this ruleset excludes specific repositories 
		if strings.Contains(restriction, "Excludes repos:") {
			excludes := strings.TrimPrefix(restriction, "Excludes repos: ")
			excludeList := strings.Split(excludes, ", ")
			for _, exclude := range excludeList {
				if exclude == repo || strings.Contains(exclude, "*") {
					return false // Explicitly excluded
				}
			}
		}
	}
	
	// If no specific targeting info found, assume it applies to all repositories
	// unless it has explicit include targets (which would mean it doesn't apply)
	for _, restriction := range policy.Restrictions {
		if strings.Contains(restriction, "Targets repos:") {
			return false // Has specific targets but we're not in them
		}
	}
	return true // No specific targeting, applies to all repos
}

// parseRepository parses a repository specification into owner and repo
func parseRepository(repoSpec string) (string, string, error) {
	parts := strings.Split(repoSpec, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("repository must be in format 'owner/repo'")
	}
	return parts[0], parts[1], nil
}