package types

// ValidationStatus represents the migration readiness status
type ValidationStatus string

const (
	ValidationReady       ValidationStatus = "ready"        // Available in target org
	ValidationSetupNeeded ValidationStatus = "setup_needed" // Can be created/configured
	ValidationBlocker     ValidationStatus = "blocker"      // Not available/incompatible
	ValidationWarning     ValidationStatus = "warning"      // Issue but not blocking
	ValidationReview      ValidationStatus = "review"       // Requires manual review
	ValidationUnknown     ValidationStatus = "unknown"      // Could not determine
)

// ValidationResult represents the validation of a single dependency
type ValidationResult struct {
	Item           string           `json:"item"`
	Status         ValidationStatus `json:"status"`
	Message        string           `json:"message,omitempty"`
	Recommendation string           `json:"recommendation,omitempty"`
}

// MigrationValidation contains validation results for all dependency categories
type MigrationValidation struct {
	TargetOrganization string                       `json:"target_organization"`
	OverallReadiness   ValidationStatus             `json:"overall_readiness"`
	Summary            ValidationSummary            `json:"summary"`
	CodeDependencies   []ValidationResult           `json:"code_dependencies,omitempty"`
	CIDependencies     []ValidationResult           `json:"ci_dependencies,omitempty"`
	AccessPermissions  []ValidationResult           `json:"access_permissions,omitempty"`
	SecurityCompliance []ValidationResult           `json:"security_compliance,omitempty"`
	AppsIntegrations   []ValidationResult           `json:"apps_integrations,omitempty"`
	Governance         []ValidationResult           `json:"governance,omitempty"`
}

// ValidationSummary provides counts by validation status
type ValidationSummary struct {
	Ready       int `json:"ready"`
	SetupNeeded int `json:"setup_needed"`
	Blockers    int `json:"blockers"`
	Warnings    int `json:"warnings"`
	Review      int `json:"review"`
	Unknown     int `json:"unknown"`
	Total       int `json:"total"`
}

// TargetOrgCapabilities represents what's available in the target organization
type TargetOrgCapabilities struct {
	Organization        string              `json:"organization"`
	Apps                []string            `json:"apps"`
	Teams               []string            `json:"teams"`
	RepositoryPolicies  []OrgPolicy         `json:"repository_policies"`   // Actual repo-level policies
	MemberPrivileges    OrgMemberPrivileges `json:"member_privileges"`     // Org-wide member settings
	Rulesets            []string            `json:"rulesets"`
	Secrets             []string            `json:"secrets"`
	Variables           []string            `json:"variables"`
	Runners             []string            `json:"runners"`
}

// OrganizationalDependencies represents all categories of dependencies
// that tie a repository to its organizational context
type OrganizationalDependencies struct {
	Repository               string                   `json:"repository" yaml:"repository"`
	CodeDependencies         CodeDependencies         `json:"organization_specific_code_dependencies" yaml:"organization_specific_code_dependencies"`
	ActionsCIDependencies    ActionsCIDependencies    `json:"github_actions_cicd_dependencies" yaml:"github_actions_cicd_dependencies"`
	AccessPermissions        AccessPermissions        `json:"access_control_permissions" yaml:"access_control_permissions"`
	SecurityCompliance       SecurityCompliance       `json:"security_compliance_dependencies" yaml:"security_compliance_dependencies"`
	AppsIntegrations         AppsIntegrations         `json:"github_apps_integrations_dependencies" yaml:"github_apps_integrations_dependencies"`
	OrgGovernance           OrgGovernance            `json:"organizational_governance_dependencies" yaml:"organizational_governance_dependencies"`
	Validation              *MigrationValidation     `json:"migration_validation,omitempty" yaml:"migration_validation,omitempty"`
}

// CodeDependencies represents organization-specific code dependencies
type CodeDependencies struct {
	InternalRepositoryReferences      []string `json:"internal_repository_references"`
	GitSubmodules                     []string `json:"git_submodules"`
	OrgPackageRegistries              []string `json:"organization_package_registries"`
	HardcodedOrgReferences           []string `json:"hardcoded_organization_references"`
	OrgSpecificContainerRegistries    []string `json:"organization_specific_container_registries"`
}

// ActionsCIDependencies represents GitHub Actions and CI/CD dependencies
type ActionsCIDependencies struct {
	OrganizationSecrets              []string `json:"organization_secrets"`
	OrganizationVariables            []string `json:"organization_variables"`
	SelfHostedRunners                []string `json:"self_hosted_runners"`
	EnvironmentDependencies          []string `json:"environment_dependencies"`
	OrgSpecificActions               []string `json:"organization_specific_actions"`
	RequiredWorkflows                []string `json:"required_workflows"`
	CrossRepoWorkflowTriggers        []string `json:"cross_repo_workflow_triggers"`
}

// AccessPermissions represents access control and permissions
type AccessPermissions struct {
	Teams                           []string `json:"teams"`
	IndividualCollaborators         []string `json:"individual_collaborators"`
	OrganizationRoles               []string `json:"organization_roles"`
	OrganizationMembership          []string `json:"organization_membership"`
	CodeownersRequirements          []string `json:"codeowners_requirements"`
}

// SecurityCompliance represents security and compliance dependencies
type SecurityCompliance struct {
	SecurityCampaigns               []string `json:"security_campaigns"`
}

// AppsIntegrations represents GitHub Apps and integrations
type AppsIntegrations struct {
	InstalledGitHubApps             []string `json:"installed_github_apps"`
	PersonalAccessTokens            []string `json:"personal_access_tokens"`
}

// OrgAppsIntegrations represents organization-level apps and integrations
// Used for caching organization-level data in batch processing
type OrgAppsIntegrations struct {
	InstalledGitHubApps             []string `json:"installed_github_apps"`
}

// OrgPolicy represents a structured organizational policy
type OrgPolicy struct {
	Name         string   `json:"name"`
	Status       string   `json:"status"`
	Restrictions []string `json:"restrictions"`
}

// OrgMemberPrivileges represents organization-wide member settings
type OrgMemberPrivileges struct {
	CanCreateRepos          bool     `json:"can_create_repos"`
	CanForkPrivateRepos     bool     `json:"can_fork_private_repos"`
	TwoFactorRequired       bool     `json:"two_factor_required"`
	WebCommitSignoffRequired bool    `json:"web_commit_signoff_required"`
	DefaultPermission       string   `json:"default_permission"`
	RestrictionsActive      []string `json:"restrictions_active"`
}

// OrgGovernance represents organizational governance dependencies
type OrgGovernance struct {
	OrganizationPolicies            []OrgPolicy `json:"-"`                    // Internal use only, not in JSON
	RepositoryPolicies              []OrgPolicy `json:"repository_policies"`
	MemberPrivileges                []string    `json:"member_privileges"`
	RepositoryRulesets              []OrgPolicy `json:"repository_rulesets"`
	IssueTemplates                  []string    `json:"issue_templates"`
	PullRequestTemplates            []string    `json:"pull_request_templates"`
	RequiredStatusChecks            []string    `json:"required_status_checks"`
}

// Legacy types for governance inspection (to be refactored)
type GovernanceConfig struct {
	Repository       RepoInfo         `json:"repository" yaml:"repository"`
	RepoSettings     RepoSettings     `json:"repository_settings,omitempty" yaml:"repository_settings,omitempty"`
	SecuritySettings SecuritySettings `json:"security_settings,omitempty" yaml:"security_settings,omitempty"`
	Rulesets         []Ruleset        `json:"rulesets,omitempty" yaml:"rulesets,omitempty"`
	Collaborators    []Collaborator   `json:"collaborators,omitempty" yaml:"collaborators,omitempty"`
	Teams            []Team           `json:"teams,omitempty" yaml:"teams,omitempty"`
	Labels           []Label          `json:"labels,omitempty" yaml:"labels,omitempty"`
	Milestones       []Milestone      `json:"milestones,omitempty" yaml:"milestones,omitempty"`
}

type RepoInfo struct {
	Owner string `json:"owner" yaml:"owner"`
	Name  string `json:"name" yaml:"name"`
}

type RepoSettings struct {
	Private              bool   `json:"private" yaml:"private"`
	Archived             bool   `json:"archived" yaml:"archived"`
	DefaultBranch        string `json:"default_branch" yaml:"default_branch"`
	HasIssues            bool   `json:"has_issues" yaml:"has_issues"`
	HasProjects          bool   `json:"has_projects" yaml:"has_projects"`
	HasWiki              bool   `json:"has_wiki" yaml:"has_wiki"`
	AllowMergeCommit     bool   `json:"allow_merge_commit" yaml:"allow_merge_commit"`
	AllowSquashMerge     bool   `json:"allow_squash_merge" yaml:"allow_squash_merge"`
	AllowRebaseMerge     bool   `json:"allow_rebase_merge" yaml:"allow_rebase_merge"`
	DeleteBranchOnMerge  bool   `json:"delete_branch_on_merge" yaml:"delete_branch_on_merge"`
}

type SecuritySettings struct {
	VulnerabilityAlerts            bool `json:"vulnerability_alerts" yaml:"vulnerability_alerts"`
	AutomatedSecurityFixes         bool `json:"automated_security_fixes" yaml:"automated_security_fixes"`
	SecretScanning                 bool `json:"secret_scanning" yaml:"secret_scanning"`
	SecretScanningPushProtection   bool `json:"secret_scanning_push_protection" yaml:"secret_scanning_push_protection"`
	DependencyGraphEnabled         bool `json:"dependency_graph_enabled" yaml:"dependency_graph_enabled"`
}

type Ruleset struct {
	ID          int    `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Target      string `json:"target" yaml:"target"`
	Enforcement string `json:"enforcement" yaml:"enforcement"`
	Source      string `json:"source" yaml:"source"`
}

type Collaborator struct {
	Login       string `json:"login" yaml:"login"`
	Permission  string `json:"permission" yaml:"permission"`
}

type Team struct {
	Name        string `json:"name" yaml:"name"`
	Permission  string `json:"permission" yaml:"permission"`
}

type Label struct {
	Name        string `json:"name" yaml:"name"`
	Color       string `json:"color" yaml:"color"`
	Description string `json:"description" yaml:"description"`
}

type Milestone struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	State       string `json:"state" yaml:"state"`
	DueOn       string `json:"due_on" yaml:"due_on"`
}