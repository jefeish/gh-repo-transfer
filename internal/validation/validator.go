package validation

import (
	"fmt"
	"strings"

	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// ValidateAgainstTarget compares source dependencies against target org capabilities
func ValidateAgainstTarget(deps *types.OrganizationalDependencies, capabilities *types.TargetOrgCapabilities, assignTeams bool) *types.MigrationValidation {
	validation := &types.MigrationValidation{
		TargetOrganization: capabilities.Organization,
		Summary:           types.ValidationSummary{},
	}

	// Validate each dependency category
	validation.AppsIntegrations = validateAppsIntegrations(deps.AppsIntegrations, capabilities)
	validation.AccessPermissions = validateAccessPermissions(deps.AccessPermissions, capabilities, assignTeams)
	validation.CIDependencies = validateCIDependencies(deps.ActionsCIDependencies, capabilities)
	validation.Governance = validateGovernance(deps.OrgGovernance, capabilities)
	validation.CodeDependencies = validateCodeDependencies(deps.CodeDependencies, capabilities)
	validation.SecurityCompliance = validateSecurityCompliance(deps.SecurityCompliance, capabilities)

	// Calculate summary and overall readiness
	validation.Summary = calculateSummary(validation)
	validation.OverallReadiness = determineOverallReadiness(validation.Summary)

	return validation
}

// validateAppsIntegrations checks if required apps are available in target org
func validateAppsIntegrations(apps types.AppsIntegrations, capabilities *types.TargetOrgCapabilities) []types.ValidationResult {
	var results []types.ValidationResult

	for _, app := range apps.InstalledGitHubApps {
		// Extract app name from the formatted string
		appName := extractAppName(app)
		
		status := types.ValidationSetupNeeded
		message := ""
		recommendation := ""

		// Check if app is available in target org
		if isAppAvailable(appName, capabilities.Apps) {
			status = types.ValidationReady
			message = "App is available in target organization"
		} else if isCommonApp(appName) {
			status = types.ValidationSetupNeeded
			message = "Standard app, can be installed"
			recommendation = fmt.Sprintf("Install %s in target organization", appName)
		} else {
			status = types.ValidationBlocker
			message = "Custom app, requires manual setup"
			recommendation = "Review app requirements and setup in target org"
		}

		results = append(results, types.ValidationResult{
			Item:           app,
			Status:         status,
			Message:        message,
			Recommendation: recommendation,
		})
	}

	return results
}

// validateAccessPermissions checks teams and collaborator access in target org
func validateAccessPermissions(access types.AccessPermissions, capabilities *types.TargetOrgCapabilities, assignTeams bool) []types.ValidationResult {
	var results []types.ValidationResult

	// Validate teams - missing teams are now always blockers
	for _, team := range access.Teams {
		teamName := extractTeamName(team)
		
		status := types.ValidationBlocker
		message := "Team does not exist in target organization"
		recommendation := fmt.Sprintf("Create team '%s' in target organization", teamName)

		if isTeamAvailable(teamName, capabilities.Teams) {
			status = types.ValidationReady
			message = "Team exists in target organization"
			recommendation = ""
		}

		results = append(results, types.ValidationResult{
			Item:           team,
			Status:         status,
			Message:        message,
			Recommendation: recommendation,
		})
	}

	// Individual collaborators are now warnings instead of requiring manual review
	for _, collaborator := range access.IndividualCollaborators {
		results = append(results, types.ValidationResult{
			Item:           collaborator,
			Status:         types.ValidationWarning,
			Message:        "Individual access requires manual setup in target organization",
			Recommendation: "Invite user to target organization and configure permissions",
		})
	}

	// Validate CODEOWNERS requirements
	for _, requirement := range access.CodeownersRequirements {
		if strings.HasPrefix(requirement, "Team: @") {
			// Extract team name from format "Team: @org/team-name"
			teamRef := strings.TrimPrefix(requirement, "Team: @")
			parts := strings.Split(teamRef, "/")
			if len(parts) == 2 {
				teamName := parts[1]
				
				status := types.ValidationBlocker
				message := "CODEOWNERS team does not exist in target organization"
				recommendation := fmt.Sprintf("Create team '%s' in target organization or update CODEOWNERS", teamName)

				if isTeamAvailable(teamName, capabilities.Teams) {
					status = types.ValidationReady
					message = "CODEOWNERS team exists in target organization"
					recommendation = ""
				}

				results = append(results, types.ValidationResult{
					Item:           requirement,
					Status:         status,
					Message:        message,
					Recommendation: recommendation,
				})
			}
		} else if strings.HasPrefix(requirement, "User: @") {
			// CODEOWNERS users are warnings instead of requiring manual review
			results = append(results, types.ValidationResult{
				Item:           requirement,
				Status:         types.ValidationWarning,
				Message:        "CODEOWNERS user requires manual setup in target organization",
				Recommendation: "Invite user to target organization or update CODEOWNERS",
			})
		}
	}

	return results
}

// validateCIDependencies checks CI/CD dependencies like secrets, variables, runners
func validateCIDependencies(ci types.ActionsCIDependencies, capabilities *types.TargetOrgCapabilities) []types.ValidationResult {
	var results []types.ValidationResult

	// Validate organization secrets
	for _, secret := range ci.OrganizationSecrets {
		secretName := extractSecretName(secret)
		
		status := types.ValidationSetupNeeded
		message := "Secret needs to be created in target organization"
		recommendation := fmt.Sprintf("Create secret '%s' in target organization", secretName)

		if isSecretAvailable(secretName, capabilities.Secrets) {
			status = types.ValidationReady
			message = "Secret exists in target organization"
			recommendation = ""
		}

		results = append(results, types.ValidationResult{
			Item:           secret,
			Status:         status,
			Message:        message,
			Recommendation: recommendation,
		})
	}

	// Validate organization variables
	for _, variable := range ci.OrganizationVariables {
		variableName := extractVariableName(variable)
		
		status := types.ValidationSetupNeeded
		message := "Variable needs to be created in target organization"
		recommendation := fmt.Sprintf("Create variable '%s' in target organization", variableName)

		if isVariableAvailable(variableName, capabilities.Variables) {
			status = types.ValidationReady
			message = "Variable exists in target organization"
			recommendation = ""
		}

		results = append(results, types.ValidationResult{
			Item:           variable,
			Status:         status,
			Message:        message,
			Recommendation: recommendation,
		})
	}

	// Validate self-hosted runners
	for _, runner := range ci.SelfHostedRunners {
		runnerName := extractRunnerName(runner)
		
		status := types.ValidationSetupNeeded
		message := "Self-hosted runner needs to be set up"
		recommendation := fmt.Sprintf("Configure runner '%s' in target organization", runnerName)

		if isRunnerAvailable(runnerName, capabilities.Runners) {
			status = types.ValidationReady
			message = "Runner is available in target organization"
			recommendation = ""
		}

		results = append(results, types.ValidationResult{
			Item:           runner,
			Status:         status,
			Message:        message,
			Recommendation: recommendation,
		})
	}

	// Required workflows need manual review
	for _, workflow := range ci.RequiredWorkflows {
		results = append(results, types.ValidationResult{
			Item:           workflow,
			Status:         types.ValidationReview,
			Message:        "Required workflow policy needs manual configuration",
			Recommendation: "Set up equivalent required workflow policy in target organization",
		})
	}

	return results
}

// validateGovernance checks governance policies and templates
func validateGovernance(governance types.OrgGovernance, capabilities *types.TargetOrgCapabilities) []types.ValidationResult {
	var results []types.ValidationResult

	// Validate organization policies - distinguish between repo policies and member privileges
	for _, policy := range governance.OrganizationPolicies {
		if isMemberPrivilegePolicy(policy) {
			// This is actually member privilege configuration, not a repository policy
			result := validateMemberPrivilegePolicy(policy, capabilities.MemberPrivileges)
			results = append(results, result)
		} else {
			// This is an actual repository-level policy
			result := validateRepositoryPolicy(policy, capabilities.RepositoryPolicies)
			results = append(results, result)
		}
	}

	// Templates need manual review
	for _, template := range governance.IssueTemplates {
		results = append(results, types.ValidationResult{
			Item:           template,
			Status:         types.ValidationReview,
			Message:        "Issue template requires manual setup",
			Recommendation: "Copy template to target organization's .github repository",
		})
	}

	for _, template := range governance.PullRequestTemplates {
		results = append(results, types.ValidationResult{
			Item:           template,
			Status:         types.ValidationReview,
			Message:        "PR template requires manual setup", 
			Recommendation: "Copy template to target organization's .github repository",
		})
	}

	return results
}

// validateCodeDependencies checks code-related dependencies
func validateCodeDependencies(code types.CodeDependencies, capabilities *types.TargetOrgCapabilities) []types.ValidationResult {
	var results []types.ValidationResult

	// Git submodules need verification regardless of being external
	for _, submodule := range code.GitSubmodules {
		if strings.Contains(submodule, "external dependency") {
			results = append(results, types.ValidationResult{
				Item:           submodule,
				Status:         types.ValidationReview,
				Message:        "External repository access needs verification",
				Recommendation: "Verify target organization has access to this external repository",
			})
		} else {
			results = append(results, types.ValidationResult{
				Item:           submodule,
				Status:         types.ValidationReview,
				Message:        "Internal submodule, may need access setup",
				Recommendation: "Ensure target org has access to submodule repository",
			})
		}
	}

	return results
}

// validateSecurityCompliance checks security dependencies
func validateSecurityCompliance(security types.SecurityCompliance, capabilities *types.TargetOrgCapabilities) []types.ValidationResult {
	var results []types.ValidationResult

	// Security campaigns need manual review
	for _, campaign := range security.SecurityCampaigns {
		results = append(results, types.ValidationResult{
			Item:           campaign,
			Status:         types.ValidationReview,
			Message:        "Security campaign requires manual setup",
			Recommendation: "Configure equivalent security measures in target organization",
		})
	}

	return results
}

// Helper functions for extracting names and checking availability

func extractAppName(appString string) string {
	// Extract app name from formatted strings like "app-name (org-wide installation)"
	if idx := strings.Index(appString, " ("); idx != -1 {
		return appString[:idx]
	}
	return appString
}

func extractTeamName(teamString string) string {
	// Extract team name from formatted strings like "team-name (permission)"
	if idx := strings.Index(teamString, " ("); idx != -1 {
		return teamString[:idx]
	}
	return teamString
}

func extractSecretName(secretString string) string {
	return secretString // Assuming secrets are already just names
}

func extractVariableName(variableString string) string {
	return variableString // Assuming variables are already just names
}

func extractRunnerName(runnerString string) string {
	return runnerString // Assuming runners are already just names
}

func isAppAvailable(appName string, availableApps []string) bool {
	for _, available := range availableApps {
		if strings.EqualFold(appName, available) {
			return true
		}
	}
	return false
}

func isCommonApp(appName string) bool {
	commonApps := []string{"dependabot", "github-actions", "codecov", "sonarcloud"}
	for _, common := range commonApps {
		if strings.Contains(strings.ToLower(appName), common) {
			return true
		}
	}
	return false
}

func isTeamAvailable(teamName string, availableTeams []string) bool {
	for _, available := range availableTeams {
		if strings.EqualFold(teamName, available) {
			return true
		}
	}
	return false
}

func isSecretAvailable(secretName string, availableSecrets []string) bool {
	for _, available := range availableSecrets {
		if strings.EqualFold(secretName, available) {
			return true
		}
	}
	return false
}

func isVariableAvailable(variableName string, availableVariables []string) bool {
	for _, available := range availableVariables {
		if strings.EqualFold(variableName, available) {
			return true
		}
	}
	return false
}

func isRunnerAvailable(runnerName string, availableRunners []string) bool {
	for _, available := range availableRunners {
		if strings.EqualFold(runnerName, available) {
			return true
		}
	}
	return false
}

func isPolicyAvailable(policy types.OrgPolicy, availablePolicies []types.OrgPolicy) bool {
	for _, available := range availablePolicies {
		if strings.EqualFold(policy.Name, available.Name) {
			return true
		}
	}
	return false
}

// hasRelatedOrgSettings checks if target org has related settings (not exact policy match)
func hasRelatedOrgSettings(sourcePolicy types.OrgPolicy, targetSettings []types.OrgPolicy) bool {
	// Check if any restrictions from source policy are present in target org settings
	for _, sourcRestriction := range sourcePolicy.Restrictions {
		for _, targetSetting := range targetSettings {
			for _, targetRestriction := range targetSetting.Restrictions {
				if strings.Contains(strings.ToLower(targetRestriction), strings.ToLower(sourcRestriction)) ||
					strings.Contains(strings.ToLower(sourcRestriction), strings.ToLower(targetRestriction)) {
					return true
				}
			}
		}
	}
	return false
}

// isMemberPrivilegePolicy determines if this policy is actually about member privileges
func isMemberPrivilegePolicy(policy types.OrgPolicy) bool {
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

// validateMemberPrivilegePolicy validates member privilege requirements against target org settings
func validateMemberPrivilegePolicy(policy types.OrgPolicy, targetPrivileges types.OrgMemberPrivileges) types.ValidationResult {
	// Check if target org member privileges meet the source policy requirements
	missingRestrictions := []string{}
	
	for _, restriction := range policy.Restrictions {
		restrictionLower := strings.ToLower(restriction)
		
		switch {
		case strings.Contains(restrictionLower, "repository creation restricted"):
			if targetPrivileges.CanCreateRepos {
				missingRestrictions = append(missingRestrictions, "Repository creation needs to be restricted")
			}
		case strings.Contains(restrictionLower, "private repository forking restricted"):
			if targetPrivileges.CanForkPrivateRepos {
				missingRestrictions = append(missingRestrictions, "Private repository forking needs to be restricted")
			}
		case strings.Contains(restrictionLower, "two-factor authentication required"):
			if !targetPrivileges.TwoFactorRequired {
				missingRestrictions = append(missingRestrictions, "Two-factor authentication needs to be required")
			}
		case strings.Contains(restrictionLower, "web commit signoff required"):
			if !targetPrivileges.WebCommitSignoffRequired {
				missingRestrictions = append(missingRestrictions, "Web commit signoff needs to be required")
			}
		}
	}
	
	// Determine validation status
	if len(missingRestrictions) == 0 {
		return types.ValidationResult{
			Item:           fmt.Sprintf("%s (status: %s)", policy.Name, policy.Status),
			Status:         types.ValidationReady,
			Message:        "Member privilege settings meet policy requirements",
			Recommendation: "",
		}
	} else if len(missingRestrictions) < len(policy.Restrictions) {
		return types.ValidationResult{
			Item:           fmt.Sprintf("%s (status: %s)", policy.Name, policy.Status),
			Status:         types.ValidationSetupNeeded,
			Message:        fmt.Sprintf("Some member privileges need adjustment (%d missing)", len(missingRestrictions)),
			Recommendation: fmt.Sprintf("Configure missing restrictions: %s", strings.Join(missingRestrictions, ", ")),
		}
	} else {
		return types.ValidationResult{
			Item:           fmt.Sprintf("%s (status: %s)", policy.Name, policy.Status),
			Status:         types.ValidationSetupNeeded,
			Message:        "Member privileges need configuration to meet policy requirements",
			Recommendation: fmt.Sprintf("Configure required restrictions: %s", strings.Join(missingRestrictions, ", ")),
		}
	}
}

// validateRepositoryPolicy validates actual repository policies
func validateRepositoryPolicy(policy types.OrgPolicy, targetPolicies []types.OrgPolicy) types.ValidationResult {
	// Check for actual policy matches (not just settings)
	for _, targetPolicy := range targetPolicies {
		if strings.EqualFold(policy.Name, targetPolicy.Name) {
			return types.ValidationResult{
				Item:           fmt.Sprintf("%s (status: %s)", policy.Name, policy.Status),
				Status:         types.ValidationReview,
				Message:        "Similar repository policy found, requires verification",
				Recommendation: "Verify policy configuration matches requirements",
			}
		}
	}
	
	return types.ValidationResult{
		Item:           fmt.Sprintf("%s (status: %s)", policy.Name, policy.Status),
		Status:         types.ValidationSetupNeeded,
		Message:        "Repository policy needs to be configured",
		Recommendation: fmt.Sprintf("Set up '%s' repository policy in target organization", policy.Name),
	}
}

// calculateSummary counts validation results by status
func calculateSummary(validation *types.MigrationValidation) types.ValidationSummary {
	summary := types.ValidationSummary{}
	
	allResults := append(validation.AppsIntegrations, validation.AccessPermissions...)
	allResults = append(allResults, validation.CIDependencies...)
	allResults = append(allResults, validation.Governance...)
	allResults = append(allResults, validation.CodeDependencies...)
	allResults = append(allResults, validation.SecurityCompliance...)
	
	for _, result := range allResults {
		summary.Total++
		switch result.Status {
		case types.ValidationReady:
			summary.Ready++
		case types.ValidationSetupNeeded:
			summary.SetupNeeded++
		case types.ValidationBlocker:
			summary.Blockers++
		case types.ValidationWarning:
			summary.Warnings++
		case types.ValidationReview:
			summary.Review++
		default:
			summary.Unknown++
		}
	}
	
	return summary
}

// determineOverallReadiness calculates overall migration readiness
func determineOverallReadiness(summary types.ValidationSummary) types.ValidationStatus {
	if summary.Blockers > 0 {
		return types.ValidationBlocker
	}
	if summary.SetupNeeded > 0 || summary.Review > 0 {
		return types.ValidationSetupNeeded
	}
	if summary.Warnings > 0 {
		return types.ValidationWarning
	}
	if summary.Ready == summary.Total {
		return types.ValidationReady
	}
	return types.ValidationUnknown
}