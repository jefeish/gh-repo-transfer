package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jefeish/gh-repo-transfer/internal/types"
	"gopkg.in/yaml.v3"
)

// OutputDependencies outputs the dependencies in the specified format
func OutputDependencies(deps *types.OrganizationalDependencies, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return outputJSON(deps)
	case "yaml", "yml":
		return outputYAML(deps)
	case "table":
		return outputTable(deps)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

// OutputMultipleDependencies outputs multiple repository dependencies in the specified format
func OutputMultipleDependencies(allDeps []*types.OrganizationalDependencies, format string) error {
	switch strings.ToLower(format) {
	case "json":
		return outputMultipleJSON(allDeps)
	case "yaml", "yml":
		return outputMultipleYAML(allDeps)
	case "table":
		return outputMultipleTable(allDeps)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
}

func outputJSON(deps *types.OrganizationalDependencies) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(deps)
}

func outputYAML(deps *types.OrganizationalDependencies) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(deps)
}

func outputTable(deps *types.OrganizationalDependencies) error {
	fmt.Printf("🔍 Organizational Dependencies Analysis\n")
	fmt.Printf("════════════════════════════════════════\n\n")

	// Show validation summary if present
	if deps.Validation != nil {
		printValidationSummary(deps.Validation)
	}

	// Count dependencies
	totalDeps := 0
	codeDeps := countDependencies(deps.CodeDependencies.InternalRepositoryReferences,
		deps.CodeDependencies.GitSubmodules,
		deps.CodeDependencies.OrgPackageRegistries,
		deps.CodeDependencies.HardcodedOrgReferences,
		deps.CodeDependencies.OrgSpecificContainerRegistries)
	
	ciDeps := countDependencies(deps.ActionsCIDependencies.OrganizationSecrets,
		deps.ActionsCIDependencies.OrganizationVariables,
		deps.ActionsCIDependencies.SelfHostedRunners,
		deps.ActionsCIDependencies.EnvironmentDependencies,
		deps.ActionsCIDependencies.OrgSpecificActions,
		deps.ActionsCIDependencies.RequiredWorkflows,
		deps.ActionsCIDependencies.CrossRepoWorkflowTriggers)
	
	accessDeps := countDependencies(deps.AccessPermissions.Teams,
		deps.AccessPermissions.IndividualCollaborators,
		deps.AccessPermissions.OrganizationRoles,
		deps.AccessPermissions.OrganizationMembership,
		deps.AccessPermissions.CodeownersRequirements)
	
	securityDeps := countDependencies(deps.SecurityCompliance.SecurityCampaigns)
	
	appsDeps := countDependencies(deps.AppsIntegrations.InstalledGitHubApps,
		deps.AppsIntegrations.PersonalAccessTokens)
	
	govDeps := countPolicyDependencies(deps.OrgGovernance.OrganizationPolicies) +
		len(deps.OrgGovernance.RepositoryRulesets) +
		countDependencies(deps.OrgGovernance.IssueTemplates,
		deps.OrgGovernance.PullRequestTemplates,
		deps.OrgGovernance.RequiredStatusChecks)
	
	totalDeps = codeDeps + ciDeps + accessDeps + securityDeps + appsDeps + govDeps

	fmt.Printf("📊 Dependencies Summary:\n")
	fmt.Printf("├─ 💻 Code Dependencies: %d\n", codeDeps)
	fmt.Printf("├─ 🔄 Actions/CI Dependencies: %d\n", ciDeps)
	fmt.Printf("├─ 🔐 Access Control Dependencies: %d\n", accessDeps)
	fmt.Printf("├─ 🛡️  Security Dependencies: %d\n", securityDeps)
	fmt.Printf("├─ 🔗 Apps/Integrations Dependencies: %d\n", appsDeps)
	fmt.Printf("└─ 📋 Governance Dependencies: %d\n", govDeps)
	fmt.Printf("\n🎯 Total Organizational Dependencies: %d\n", totalDeps)
	fmt.Printf("════════════════════════════════════════\n\n")

	if totalDeps > 0 {
		fmt.Printf("⚠️  Moving this repository to another organization will require addressing these dependencies.\n\n")
	} else {
		fmt.Printf("✅ No organizational dependencies found. This repository appears ready for migration.\n\n")
	}
	
	// Always show all sections for transparency
	printDependencySection("💻 Organization-Specific Code Dependencies", codeDeps, map[string][]string{
		"Internal Repository References": deps.CodeDependencies.InternalRepositoryReferences,
		"Git Submodules": deps.CodeDependencies.GitSubmodules,
		"Organization Package Registries": deps.CodeDependencies.OrgPackageRegistries,
		"Hard-coded Organization References": deps.CodeDependencies.HardcodedOrgReferences,
		"Organization Container Registries": deps.CodeDependencies.OrgSpecificContainerRegistries,
	}, true)
	
	printDependencySection("🔄 GitHub Actions & CI/CD Dependencies", ciDeps, map[string][]string{
		"Organization Secrets": deps.ActionsCIDependencies.OrganizationSecrets,
		"Organization Variables": deps.ActionsCIDependencies.OrganizationVariables,
		"Self-hosted Runners": deps.ActionsCIDependencies.SelfHostedRunners,
		"Environment Dependencies": deps.ActionsCIDependencies.EnvironmentDependencies,
		"Organization-specific Actions": deps.ActionsCIDependencies.OrgSpecificActions,
		"Required Workflows": deps.ActionsCIDependencies.RequiredWorkflows,
		"Cross-repo Workflow Triggers": deps.ActionsCIDependencies.CrossRepoWorkflowTriggers,
	}, true)
	
	printDependencySection("🔐 Access Control & Permissions", accessDeps, map[string][]string{
		"Teams": deps.AccessPermissions.Teams,
		"Individual Collaborators": deps.AccessPermissions.IndividualCollaborators,
		"Organization Roles": deps.AccessPermissions.OrganizationRoles,
		"Organization Membership": deps.AccessPermissions.OrganizationMembership,
		"CODEOWNERS Requirements": deps.AccessPermissions.CodeownersRequirements,
	}, true)
	
	printDependencySection("🛡️  Security & Compliance Dependencies", securityDeps, map[string][]string{
		"Security Campaigns": deps.SecurityCompliance.SecurityCampaigns,
	}, true)
	
	printDependencySection("🔗 GitHub Apps & Integrations", appsDeps, map[string][]string{
		"Installed GitHub Apps": deps.AppsIntegrations.InstalledGitHubApps,
		"Personal Access Tokens": deps.AppsIntegrations.PersonalAccessTokens,
	}, true)
	
	// Custom governance section with separated policies and privileges
	printGovernanceDependencies(deps.OrgGovernance, govDeps)

	return nil
}

func countDependencies(slices ...[]string) int {
	total := 0
	for _, slice := range slices {
		total += len(slice)
	}
	return total
}

func countPolicyDependencies(policies []types.OrgPolicy) int {
	return len(policies)
}

func formatPoliciesForDisplay(policies []types.OrgPolicy) []string {
	var formatted []string
	for _, policy := range policies {
		policyInfo := fmt.Sprintf("%s (status: %s)", policy.Name, policy.Status)
		for i, restriction := range policy.Restrictions {
			if i == len(policy.Restrictions)-1 {
				policyInfo += "\n│      └─ " + restriction
			} else {
				policyInfo += "\n│      ├─ " + restriction
			}
		}
		formatted = append(formatted, policyInfo)
	}
	return formatted
}

// printValidationSummary displays validation results summary
func printValidationSummary(validation *types.MigrationValidation) {
	fmt.Printf("🎯 Migration Validation Summary (Target: %s)\n", validation.TargetOrganization)
	fmt.Printf("═══════════════════════════════════════════════\n")
	
	// Overall status
	statusEmoji := getStatusEmoji(validation.OverallReadiness)
	fmt.Printf("Overall Readiness: %s %s\n\n", statusEmoji, validation.OverallReadiness)
	
	// Summary counts
	fmt.Printf("📊 Validation Summary:\n")
	fmt.Printf("├─ 🟢 Ready: %d\n", validation.Summary.Ready)
	fmt.Printf("├─ 🟡 Setup Needed: %d\n", validation.Summary.SetupNeeded)
	fmt.Printf("├─ 🔴 Blockers: %d\n", validation.Summary.Blockers)
	fmt.Printf("├─ ⚪ Manual Review: %d\n", validation.Summary.Review)
	fmt.Printf("└─ ❓ Unknown: %d\n\n", validation.Summary.Unknown)
	
	fmt.Printf("🎯 Total Items Validated: %d\n", validation.Summary.Total)
	fmt.Printf("════════════════════════════════════════\n\n")
	
	// Show detailed validation results if there are issues
	if validation.Summary.Blockers > 0 || validation.Summary.SetupNeeded > 0 || validation.Summary.Review > 0 {
		printDetailedValidation(validation)
	}
}

// printDetailedValidation shows detailed validation results
func printDetailedValidation(validation *types.MigrationValidation) {
	fmt.Printf("📋 Detailed Validation Results\n")
	fmt.Printf("════════════════════════════════════════\n\n")
	
	if len(validation.AppsIntegrations) > 0 {
		printValidationCategory("🔗 Apps & Integrations", validation.AppsIntegrations)
	}
	
	if len(validation.AccessPermissions) > 0 {
		printValidationCategory("🔐 Access Control", validation.AccessPermissions)
	}
	
	if len(validation.CIDependencies) > 0 {
		printValidationCategory("🔄 CI/CD Dependencies", validation.CIDependencies)
	}
	
	if len(validation.Governance) > 0 {
		printValidationCategory("📋 Governance", validation.Governance)
	}
	
	if len(validation.CodeDependencies) > 0 {
		printValidationCategory("💻 Code Dependencies", validation.CodeDependencies)
	}
	
	if len(validation.SecurityCompliance) > 0 {
		printValidationCategory("🛡️ Security & Compliance", validation.SecurityCompliance)
	}
}

// printValidationCategory prints validation results for a specific category
func printValidationCategory(title string, results []types.ValidationResult) {
	fmt.Printf("%s\n", title)
	
	for i, result := range results {
		isLast := i == len(results)-1
		prefix := "├─"
		if isLast {
			prefix = "└─"
		}
		
		statusEmoji := getStatusEmoji(result.Status)
		fmt.Printf("%s %s %s\n", prefix, statusEmoji, result.Item)
		
		// Show message and recommendation for non-ready items
		if result.Status != types.ValidationReady {
			indentPrefix := "│  "
			if isLast {
				indentPrefix = "   "
			}
			
			if result.Message != "" {
				fmt.Printf("%s   📝 %s\n", indentPrefix, result.Message)
			}
			if result.Recommendation != "" {
				fmt.Printf("%s   💡 %s\n", indentPrefix, result.Recommendation)
			}
		}
	}
	
	fmt.Printf("\n")
}

// getStatusEmoji returns emoji for validation status
func getStatusEmoji(status types.ValidationStatus) string {
	switch status {
	case types.ValidationReady:
		return "🟢"
	case types.ValidationSetupNeeded:
		return "🟡"
	case types.ValidationBlocker:
		return "🔴"
	case types.ValidationReview:
		return "⚪"
	default:
		return "❓"
	}
}

// printGovernanceDependencies displays governance dependencies with separated sections
func printGovernanceDependencies(governance types.OrgGovernance, totalCount int) {
	fmt.Printf("📋 Organizational Governance\n")
	if totalCount == 0 {
		fmt.Printf("└─ No dependencies found\n\n")
		return
	}

	// Build sections to display using pre-separated data
	sections := []struct {
		name  string
		items []string
	}{}

	// Repository Policies section (use pre-separated data)
	if len(governance.RepositoryPolicies) > 0 {
		var formattedRepoPolicies []string
		for _, policy := range governance.RepositoryPolicies {
			policyInfo := fmt.Sprintf("%s (status: %s)", policy.Name, policy.Status)
			for i, restriction := range policy.Restrictions {
				if i == len(policy.Restrictions)-1 {
					policyInfo += "\n     └─ " + restriction
				} else {
					policyInfo += "\n     ├─ " + restriction
				}
			}
			formattedRepoPolicies = append(formattedRepoPolicies, policyInfo)
		}
		sections = append(sections, struct {
			name  string
			items []string
		}{"Repository Policies", formattedRepoPolicies})
	}

	// Member Privileges section (use pre-separated data)
	if len(governance.MemberPrivileges) > 0 {
		sections = append(sections, struct {
			name  string
			items []string
		}{"Member Privileges", governance.MemberPrivileges})
	}

	// Add other governance items
	if len(governance.RepositoryRulesets) > 0 {
		var formattedRepoRulesets []string
		for _, ruleset := range governance.RepositoryRulesets {
			rulesetInfo := fmt.Sprintf("%s (status: %s)", ruleset.Name, ruleset.Status)
			for i, restriction := range ruleset.Restrictions {
				if i == len(ruleset.Restrictions)-1 {
					rulesetInfo += "\n     └─ " + restriction
				} else {
					rulesetInfo += "\n     ├─ " + restriction
				}
			}
			formattedRepoRulesets = append(formattedRepoRulesets, rulesetInfo)
		}
		sections = append(sections, struct {
			name  string
			items []string
		}{"Repository Rulesets", formattedRepoRulesets})
	}

	if len(governance.IssueTemplates) > 0 {
		sections = append(sections, struct {
			name  string
			items []string
		}{"Issue Templates", governance.IssueTemplates})
	}

	if len(governance.PullRequestTemplates) > 0 {
		sections = append(sections, struct {
			name  string
			items []string
		}{"Pull Request Templates", governance.PullRequestTemplates})
	}

	if len(governance.RequiredStatusChecks) > 0 {
		sections = append(sections, struct {
			name  string
			items []string
		}{"Required Status Checks", governance.RequiredStatusChecks})
	}

	// Print all sections
	for sectionIdx, section := range sections {
		isLastSection := sectionIdx == len(sections)-1
		sectionPrefix := "├─"
		if isLastSection {
			sectionPrefix = "└─"
		}

		fmt.Printf("%s %s (%d):\n", sectionPrefix, section.name, len(section.items))

		for itemIdx, item := range section.items {
			isLastItem := itemIdx == len(section.items)-1
			itemPrefix := "│  ├─"
			if isLastSection && isLastItem {
				itemPrefix = "   └─"
			} else if isLastItem {
				itemPrefix = "│  └─"
			}

			// Handle multi-line items (like policies with restrictions)
			lines := strings.Split(item, "\n")
			fmt.Printf("%s %s\n", itemPrefix, lines[0])
			
			// Print additional lines for policy restrictions with proper tree formatting
			for _, line := range lines[1:] {
				if isLastSection {
					fmt.Printf("   %s\n", line)
				} else {
					fmt.Printf("│  %s\n", line)
				}
			}
		}
	}

	fmt.Printf("\n")
}

// Helper function to check if policy is about member privileges (moved from validation package)
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

func printDependencySection(title string, count int, dependencies map[string][]string, showEmpty bool) {
	if count == 0 && !showEmpty {
		return
	}
	
	fmt.Printf("%s\n", title)
	if count == 0 {
		fmt.Printf("└─ No dependencies found\n\n")
		return
	}
	
	// Convert map to ordered slice to control the display order and identify last item
	type categoryInfo struct {
		name  string
		items []string
	}
	
	var categories []categoryInfo
	for category, items := range dependencies {
		if len(items) > 0 {
			categories = append(categories, categoryInfo{name: category, items: items})
		}
	}
	
	for i, cat := range categories {
		isLastCategory := (i == len(categories)-1)
		
		if isLastCategory {
			fmt.Printf("└─ %s (%d):\n", cat.name, len(cat.items))
			// Use spaces instead of │ for the last category
			for j, item := range cat.items {
				if j == len(cat.items)-1 {
					fmt.Printf("   └─ %s\n", item)
				} else {
					fmt.Printf("   ├─ %s\n", item)
				}
			}
		} else {
			fmt.Printf("├─ %s (%d):\n", cat.name, len(cat.items))
			for j, item := range cat.items {
				if j == len(cat.items)-1 {
					fmt.Printf("│  └─ %s\n", item)
				} else {
					fmt.Printf("│  ├─ %s\n", item)
				}
			}
		}
	}
	fmt.Printf("\n")
}

// Multiple repository output functions

func outputMultipleJSON(allDeps []*types.OrganizationalDependencies) error {
	output := map[string]interface{}{
		"repositories": allDeps,
		"summary": generateBatchSummary(allDeps),
	}
	
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputMultipleYAML(allDeps []*types.OrganizationalDependencies) error {
	output := map[string]interface{}{
		"repositories": allDeps,
		"summary": generateBatchSummary(allDeps),
	}
	
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(output)
}

func outputMultipleTable(allDeps []*types.OrganizationalDependencies) error {
	fmt.Printf("🔍 Batch Organizational Dependencies Analysis\n")
	fmt.Printf("═══════════════════════════════════════════════\n\n")

	// Print summary
	summary := generateBatchSummary(allDeps)
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("  Repositories analyzed: %d\n", summary.TotalRepositories)
	fmt.Printf("  Organizations: %d\n", summary.TotalOrganizations)
	fmt.Printf("  Total dependencies: %d\n", summary.TotalDependencies)
	
	if len(summary.ValidationSummary) > 0 {
		fmt.Printf("  Validation status:\n")
		for status, count := range summary.ValidationSummary {
			fmt.Printf("    %s: %d repositories\n", status, count)
		}
	}
	fmt.Printf("\n")

	// Output each repository's analysis
	for i, deps := range allDeps {
		fmt.Printf("📦 Repository %d: %s\n", i+1, deps.Repository)
		fmt.Printf("─────────────────────────────────────────\n")
		
		// Use existing table output function for individual repos
		err := outputTable(deps)
		if err != nil {
			return fmt.Errorf("failed to output table for repository %s: %v", deps.Repository, err)
		}
		
		if i < len(allDeps)-1 {
			fmt.Printf("\n═══════════════════════════════════════════════\n\n")
		}
	}

	return nil
}

// BatchSummary provides a summary of batch analysis results
type BatchSummary struct {
	TotalRepositories  int            `json:"total_repositories" yaml:"total_repositories"`
	TotalOrganizations int            `json:"total_organizations" yaml:"total_organizations"`
	TotalDependencies  int            `json:"total_dependencies" yaml:"total_dependencies"`
	ValidationSummary  map[string]int `json:"validation_summary,omitempty" yaml:"validation_summary,omitempty"`
}

func generateBatchSummary(allDeps []*types.OrganizationalDependencies) BatchSummary {
	summary := BatchSummary{
		TotalRepositories: len(allDeps),
		ValidationSummary: make(map[string]int),
	}
	
	orgs := make(map[string]bool)
	totalDeps := 0
	
	for _, deps := range allDeps {
		// Extract organization from repository name
		parts := strings.Split(deps.Repository, "/")
		if len(parts) >= 2 {
			orgs[parts[0]] = true
		}
		
		// Count total dependencies for this repository
		var repoOrgPolicyNames []string
		for _, policy := range deps.OrgGovernance.RepositoryPolicies {
			repoOrgPolicyNames = append(repoOrgPolicyNames, policy.Name)
		}
		
		repoDeps := countDependencies(
			deps.CodeDependencies.InternalRepositoryReferences,
			deps.ActionsCIDependencies.OrganizationSecrets,
			deps.ActionsCIDependencies.OrganizationVariables,
			deps.AccessPermissions.Teams,
			deps.AccessPermissions.IndividualCollaborators,
			deps.SecurityCompliance.SecurityCampaigns,
			deps.AppsIntegrations.InstalledGitHubApps,
			repoOrgPolicyNames,
		)
		totalDeps += repoDeps
		
		// Count validation status if available
		if deps.Validation != nil {
			// Count items across all validation categories
			allValidations := append(deps.Validation.CodeDependencies, 
				append(deps.Validation.CIDependencies,
					append(deps.Validation.AccessPermissions,
						append(deps.Validation.SecurityCompliance,
							append(deps.Validation.AppsIntegrations, deps.Validation.Governance...)...)...)...)...)
			
			for _, validation := range allValidations {
				summary.ValidationSummary[string(validation.Status)]++
			}
		}
	}
	
	summary.TotalOrganizations = len(orgs)
	summary.TotalDependencies = totalDeps
	
	return summary
}

// OutputSeparateFiles outputs each repository analysis to individual JSON files
func OutputSeparateFiles(allDeps []*types.OrganizationalDependencies, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Creating separate JSON files for %d repositories\n", len(allDeps))
	}
	
	for _, deps := range allDeps {
		// Generate safe filename from repository name
		filename := generateSafeFilename(deps.Repository) + ".json"
		
		if verbose {
			fmt.Fprintf(os.Stderr, "Writing %s\n", filename)
		}
		
		// Create file
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", filename, err)
		}
		defer file.Close()
		
		// Write JSON to file
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(deps); err != nil {
			return fmt.Errorf("failed to write JSON to file %s: %v", filename, err)
		}
	}
	
	if verbose {
		fmt.Fprintf(os.Stderr, "Successfully created %d JSON files\n", len(allDeps))
	} else {
		fmt.Printf("Created %d individual JSON files\n", len(allDeps))
	}
	
	return nil
}

// generateSafeFilename converts repository name to filesystem-safe filename
func generateSafeFilename(repoName string) string {
	// Replace problematic characters with safe alternatives
	safe := strings.ReplaceAll(repoName, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, ":", "_")
	safe = strings.ReplaceAll(safe, "*", "_")
	safe = strings.ReplaceAll(safe, "?", "_")
	safe = strings.ReplaceAll(safe, "\"", "_")
	safe = strings.ReplaceAll(safe, "<", "_")
	safe = strings.ReplaceAll(safe, ">", "_")
	safe = strings.ReplaceAll(safe, "|", "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	
	// Add prefix for clarity and avoid conflicts
	return "repo-analysis_" + safe
}