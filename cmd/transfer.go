package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"

	"github.com/jefeish/gh-repo-transfer/internal/analyzer"
	"github.com/jefeish/gh-repo-transfer/internal/types"
	"github.com/jefeish/gh-repo-transfer/internal/validation"
)

// transferCmd represents the transfer command
var transferCmd = &cobra.Command{
	Use:   "transfer [owner/repo...] --target-org [target-owner]",
	Short: "Transfer repository(ies) to another organization or user",
	Long: `Transfer one or more repositories to another organization or user using the GitHub REST API.

Usage:
  repo-transfer transfer [owner/repo] --target-org [target-owner]
  repo-transfer transfer [owner/repo1] [owner/repo2] --target-org [target-owner]
  repo-transfer transfer [owner/repo] --target-org [target-owner] --dry-run
  repo-transfer transfer [owner/repo] --target-org [target-owner] --verbose

This command will:
1. Validate the target organization/user exists
2. Check transfer permissions on each source repository
3. Perform dependency validation (unless --enforce is used)
4. Execute the repository transfer(s)

Multiple repositories can be transferred in batch:
  repo-transfer transfer owner/repo1 owner/repo2 owner/repo3 --target-org new-org

Examples:
  gh repo-transfer transfer owner/repo --target-org target-org
  gh repo-transfer transfer owner/repo1 owner/repo2 --target-org target-org --dry-run`,
	SilenceUsage: true,
	RunE: runTransfer,
}

var (
	targetOwnerLocal string
	teamIds          []string
	dryRunLocal      bool
)

func init() {
	rootCmd.AddCommand(transferCmd)
	// Most flags are now defined as persistent flags in root.go
	// transferCmd.Flags().StringVar(&targetOwnerLocal, "target-org", "", "Target organization or user to transfer the repository to (required)")
	// transferCmd.Flags().BoolVar(&dryRunLocal, "dry-run", false, "Show what would be transferred without actually performing the transfer")
	// transferCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	
	// Mark the --target-org flag as required
	transferCmd.MarkFlagRequired("target-org")
}

func runTransfer(cmd *cobra.Command, args []string) error {
	var repos []string
	
	if len(args) == 0 {
		// Try to get repo from current directory
		currentRepo, err := getCurrentRepo()
		if err != nil {
			return fmt.Errorf("no repository specified and could not determine current repository: %v", err)
		}
		repos = []string{currentRepo}
	} else {
		repos = args
	}

	// Validate repository format
	for _, repo := range repos {
		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			return fmt.Errorf("repository '%s' must be in format 'owner/repo'", repo)
		}
	}

	if verbose {
		if len(repos) == 1 {
			fmt.Fprintf(os.Stderr, "Preparing to transfer repository: %s to %s\n", repos[0], targetOrg)
		} else {
			fmt.Fprintf(os.Stderr, "Preparing to transfer %d repositories to %s\n", len(repos), targetOrg)
		}
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %v", err)
	}

	// Validate target owner exists (once for all repos)
	if err := validateTargetOwner(*client, targetOrg); err != nil {
		return fmt.Errorf("failed to validate target owner: %v", err)
	}

	// Validate teams exist if specified (once for all repos)
	if len(teamIds) > 0 {
		if err := validateTeams(*client, targetOrg, teamIds); err != nil {
			return fmt.Errorf("failed to validate teams: %v", err)
		}
	}

	// Pre-scan target organization capabilities once for all repositories (optimization)
	var targetCapabilities *types.TargetOrgCapabilities
	if targetOrg != "" && !enforce {
		if verbose {
			fmt.Fprintf(os.Stderr, "Scanning target organization capabilities: %s\n", targetOrg)
		}
		caps, err := validation.ScanTargetOrganization(*client, targetOrg, verbose)
		if err != nil {
			return fmt.Errorf("failed to scan target organization: %v", err)
		}
		targetCapabilities = caps
	}

	// STEP 0: Create teams in target org if --create is set
	if createTeams {
		for _, repo := range repos {
			parts := strings.Split(repo, "/")
			owner, repoName := parts[0], parts[1]
			sourceTeamPermissions, err := getRepositoryTeams(*client, owner, repoName)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Could not retrieve team permissions for %s: %v\n", repo, err)
				}
				continue
			}
			err = createTeamsInTargetOrg(*client, owner, repoName, targetOrg, sourceTeamPermissions)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Failed to create teams for %s: %v\n", repo, err)
				}
			}
		}
	}

	// Group repositories by organization for efficient batch processing
	orgRepos := groupReposByOrganization(repos)
	if verbose && len(orgRepos) > 1 {
		fmt.Fprintf(os.Stderr, "Processing %d repositories across %d organizations\n", len(repos), len(orgRepos))
	}

	// Process each repository with optimizations
	var results []transferResult
	repoIndex := 0
	for orgName, orgRepoList := range orgRepos {
		if verbose && len(orgRepoList) > 1 {
			fmt.Fprintf(os.Stderr, "\nProcessing %d repositories from organization: %s\n", len(orgRepoList), orgName)
		}
		
		for _, repo := range orgRepoList {
			parts := strings.Split(repo, "/")
			owner, repoName := parts[0], parts[1]

			if len(repos) > 1 {
				repoIndex++
				fmt.Fprintf(os.Stderr, "\n[%d/%d] Processing %s\n", repoIndex, len(repos), repo)
			}

			result := processRepoTransferOptimized(*client, owner, repoName, targetCapabilities)
			results = append(results, result)
		}
	}

	// Handle dry-run summary for multiple repos
	if dryRun {
		return displayBatchTransferSummary(results)
	}

	// Check for failures in actual transfer
	return handleBatchTransferResults(*client, results)
}

// validateTargetOwner checks if the target organization or user exists
func validateTargetOwner(client api.RESTClient, target string) error {
	// Try as organization first
	var orgResponse struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s", target), &orgResponse)
	if err == nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "✅ Target organization '%s' exists\n", target)
		}
		return nil
	}

	// Try as user
	var userResponse struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	}

	err = client.Get(fmt.Sprintf("users/%s", target), &userResponse)
	if err == nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "✅ Target user '%s' exists\n", target)
		}
		return nil
	}

	return fmt.Errorf("target '%s' not found (not an organization or user)", target)
}

// validateTeams checks if the specified teams exist in the target organization
func validateTeams(client api.RESTClient, targetOrg string, teams []string) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Validating teams in target organization...\n")
	}

	for _, teamSlug := range teams {
		var teamResponse struct {
			ID   int    `json:"id"`
			Slug string `json:"slug"`
			Name string `json:"name"`
		}

		err := client.Get(fmt.Sprintf("orgs/%s/teams/%s", targetOrg, teamSlug), &teamResponse)
		if err != nil {
			return fmt.Errorf("team '%s' not found in organization '%s': %v", teamSlug, targetOrg, err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "✅ Team '%s' exists in %s\n", teamSlug, targetOrg)
		}
	}

	return nil
}

// validateSourceRepository checks if the source repository exists and can be transferred
func validateSourceRepository(client api.RESTClient, owner, repo string) error {
	var repoResponse struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
		Owner    struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"owner"`
		Permissions struct {
			Admin bool `json:"admin"`
		} `json:"permissions"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s", owner, repo), &repoResponse)
	if err != nil {
		return fmt.Errorf("repository '%s/%s' not found or not accessible: %v", owner, repo, err)
	}

	if !repoResponse.Permissions.Admin {
		return fmt.Errorf("admin permissions required to transfer repository '%s/%s'", owner, repo)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "✅ Source repository '%s' is valid and transferable\n", repoResponse.FullName)
		fmt.Fprintf(os.Stderr, "  - Private: %t\n", repoResponse.Private)
		fmt.Fprintf(os.Stderr, "  - Owner type: %s\n", repoResponse.Owner.Type)
	}

	return nil
}

// executeTransfer performs the actual repository transfer
func executeTransfer(client api.RESTClient, owner, repo, targetOwner string, teams []string, preservePermissions bool) error {
	// Collect source team permissions before transfer if we need to preserve them
	var sourceTeamPermissions []types.Team
	if len(teams) > 0 && preservePermissions {
		if verbose {
			fmt.Fprintf(os.Stderr, "Collecting team permissions from source repository before transfer...\n")
		}
		var err error
		sourceTeamPermissions, err = getRepositoryTeams(client, owner, repo)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Could not retrieve team permissions: %v\n", err)
			}
		} else {
			for _, team := range sourceTeamPermissions {
				if verbose {
					fmt.Fprintf(os.Stderr, "Source team '%s' has '%s' permission\n", team.Name, team.Permission)
				}
			}
		}
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "🔄 Initiating repository transfer...\n")
		fmt.Fprintf(os.Stderr, "Source: %s/%s\n", owner, repo)
		fmt.Fprintf(os.Stderr, "Target: %s\n", targetOwner)
	}

	// Prepare transfer payload
	transferPayload := map[string]interface{}{
		"new_owner": targetOwner,
	}

	// If teams are specified, look up their IDs in the target organization
	if len(teams) > 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "Looking up team IDs for: %v\n", teams)
		}
		
		var teamIds []int
		for _, teamName := range teams {
			teamId, err := getTeamIdByName(client, targetOwner, teamName)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Could not find team '%s' in target org: %v\n", teamName, err)
				}
				continue
			}
			teamIds = append(teamIds, teamId)
			if verbose {
				fmt.Fprintf(os.Stderr, "Found team '%s' with ID: %d\n", teamName, teamId)
			}
		}
		
		// If teams are specified, include team_ids in the transfer payload (step 1)
		if len(teamIds) > 0 {
			transferPayload["team_ids"] = teamIds
			if verbose {
				fmt.Fprintf(os.Stderr, "Including %d team_ids in transfer payload: %v\n", len(teamIds), teamIds)
			}
		}
	}

	// Marshal the payload
	payloadBytes, err := json.Marshal(transferPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal transfer payload: %v", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Transfer payload: %s\n", string(payloadBytes))
	}

	// Perform the actual repository transfer
	var transferResponse struct {
		ID          int    `json:"id"`
		NodeID      string `json:"node_id"`
		Name        string `json:"name"`
		FullName    string `json:"full_name"`
		Owner       struct {
			Login string `json:"login"`
		} `json:"owner"`
	}

	err = client.Post(fmt.Sprintf("repos/%s/%s/transfer", owner, repo), bytes.NewBuffer(payloadBytes), &transferResponse)
	if err != nil {
		return fmt.Errorf("repository transfer failed: %v", err)
	}

	fmt.Printf("✅ Repository transferred successfully!\n")
	fmt.Printf("   New location: %s\n", transferResponse.FullName)

	// Store the original path as a repository custom property (repo-origin)
	originalPath := fmt.Sprintf("%s/%s", owner, repo)
	if verbose {
		fmt.Fprintf(os.Stderr, "Storing origin tracking: '%s'\n", originalPath)
	}
	if err := storeOriginalPathProperty(client, targetOwner, repo, originalPath, verbose); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: Origin tracking encountered an error: %v\n", err)
		}
	}

	// Assign teams with their original permissions (pure two-step approach)
	if len(teams) > 0 && preservePermissions && len(sourceTeamPermissions) > 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "Assigning teams with preserved permissions...\n")
		}
		
		// Wait longer for transfer to complete fully and GitHub to update permissions  
		time.Sleep(10 * time.Second)
		
		// Assign each team with its original permission
		for _, originalTeam := range sourceTeamPermissions {
			if verbose {
				fmt.Fprintf(os.Stderr, "Assigning team '%s' with '%s' permission\n", originalTeam.Name, originalTeam.Permission)
			}
			
			err = assignTeamToRepository(client, targetOwner, originalTeam.Name, repo, originalTeam.Permission)
			if err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: Failed to assign team '%s': %v\n", originalTeam.Name, err)
				}
			} else {
				if verbose {
					fmt.Fprintf(os.Stderr, "✅ Successfully assigned team '%s' with '%s' permission\n", originalTeam.Name, originalTeam.Permission)
				}
			}
		}
		
		if verbose {
			fmt.Fprintf(os.Stderr, "✅ Team assignment completed\n")
		}
	}

	return nil
}

// assignPreCollectedTeamsToRepo assigns teams to a repository using pre-collected team names
// This is used when team information was collected before transfer but the source repo no longer exists
func assignPreCollectedTeamsToRepo(client api.RESTClient, targetOwner, repoName string, teamNames []string) error {
	if len(teamNames) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "No teams to assign\n")
		}
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Assigning %d pre-collected teams to repository...\n", len(teamNames))
	}

	for _, teamName := range teamNames {
		// Convert team name to slug format
		teamSlug := strings.ToLower(strings.ReplaceAll(teamName, " ", "-"))

		// Check if team exists in target organization
		var teamResponse struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		err := client.Get(fmt.Sprintf("orgs/%s/teams/%s", targetOwner, teamSlug), &teamResponse)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Team '%s' not found in target org, skipping: %v\n", teamName, err)
			}
			continue
		}

		// Assign team to repository with default permissions (since we don't have original permissions)
		// Note: This is a limitation when using transfer command - we only have team names, not permissions
		assignPayload := map[string]interface{}{
			"permission": "push", // Default to push/write permissions
		}

		payloadBytes, err := json.Marshal(assignPayload)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to marshal payload for team '%s': %v\n", teamName, err)
			}
			continue
		}

		err = client.Put(fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", targetOwner, teamSlug, targetOwner, repoName), bytes.NewBuffer(payloadBytes), nil)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Failed to assign team '%s' to repository: %v\n", teamName, err)
			}
			continue
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "✅ Successfully assigned team '%s' with 'push' permission\n", teamName)
		}
	}

	return nil
}

// transferResult holds the result of processing a single repository transfer
type transferResult struct {
	Repository        string
	Owner             string
	RepoName          string
	Success           bool
	BlockerCount      int
	ValidationDetails *types.MigrationValidation
	Error             error
	Mode              string
	Teams             []string // Team names from source repository (populated when --assign is used)
}

// processRepoTransfer handles the transfer logic for a single repository
func processRepoTransfer(client api.RESTClient, owner, repoName string) transferResult {
	return processRepoTransferOptimized(client, owner, repoName, nil)
}

// processRepoTransferOptimized handles the transfer logic with pre-scanned target capabilities
func processRepoTransferOptimized(client api.RESTClient, owner, repoName string, targetCapabilities *types.TargetOrgCapabilities) transferResult {
	result := transferResult{
		Repository: fmt.Sprintf("%s/%s", owner, repoName),
		Owner:      owner,
		RepoName:   repoName,
		Mode:       "VALIDATED",
	}

	// Collect team information if --assign is used, before any validation that might fail
	if assign {
		if verbose {
			fmt.Fprintf(os.Stderr, "Collecting team information from source repository for assignment...\n")
		}
		sourceTeams, err := getRepositoryTeams(client, owner, repoName)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: Could not retrieve teams from source repository: %v\n", err)
			}
		} else {
			for _, team := range sourceTeams {
				result.Teams = append(result.Teams, team.Name)
				if verbose {
					fmt.Fprintf(os.Stderr, "Found team '%s' in source repository\n", team.Name)
				}
			}
		}
	}

	// Check current repository status
	if err := validateSourceRepository(client, owner, repoName); err != nil {
		result.Error = fmt.Errorf("failed to validate source repository: %v", err)
		result.Success = false
		return result
	}

	// Perform dependency validation unless enforced
	if !enforce {
		if verbose {
			fmt.Fprintf(os.Stderr, "Checking for transfer blockers...\n")
		}
		
		// Analyze dependencies to check for blockers
		deps, err := analyzer.AnalyzeOrganizationalDependencies(client, owner, repoName, verbose)
		if err != nil {
			result.Error = fmt.Errorf("failed to analyze dependencies: %v", err)
			result.Success = false
			return result
		}
		
		// If target org is specified, validate against it (use pre-scanned capabilities if available)
		if targetOrg != "" {
			var capabilities *types.TargetOrgCapabilities
			var err error
			
			if targetCapabilities != nil {
				// Use pre-scanned capabilities (batch optimization)
				capabilities = targetCapabilities
			} else {
				// Fallback to individual scanning (single repo mode)
				capabilities, err = validation.ScanTargetOrganization(client, targetOrg, verbose)
				if err != nil {
					result.Error = fmt.Errorf("failed to scan target organization: %v", err)
					result.Success = false
					return result
				}
			}
			
			validationResult := validation.ValidateAgainstTarget(deps, capabilities, assign)
			result.BlockerCount = validationResult.Summary.Blockers
			result.ValidationDetails = validationResult
			
			if result.BlockerCount > 0 {
				result.Mode = "BLOCKED"
				result.Success = false
				result.Error = fmt.Errorf("❌ Transfer blocked: %d validation blockers found\n%s", result.BlockerCount, formatValidationBlockers(validationResult))
				return result
			}
			
			if verbose {
				fmt.Fprintf(os.Stderr, "✅ No transfer blockers found\n")
			}
		}
	} else {
		result.Mode = "ENFORCED"
		if verbose {
			fmt.Fprintf(os.Stderr, "⚠️  ENFORCED: Skipping dependency validation checks\n")
		}
	}

	result.Success = true
	return result
}

// formatValidationBlockers formats validation blocker details for error messages
func formatValidationBlockers(validation *types.MigrationValidation) string {
	if validation == nil {
		return ""
	}

	var blockers []string

	// Collect all validation blockers from each category
	categories := map[string][]types.ValidationResult{
		"Code Dependencies":   validation.CodeDependencies,
		"CI/CD Dependencies":  validation.CIDependencies,
		"Access Permissions":  validation.AccessPermissions,
		"Security Compliance": validation.SecurityCompliance,
		"Apps & Integrations": validation.AppsIntegrations,
		"Governance":          validation.Governance,
	}

	for category, results := range categories {
		for _, result := range results {
			if result.Status == types.ValidationBlocker {
				blockerMsg := fmt.Sprintf("  • [%s] %s: %s", category, result.Item, result.Message)
				if result.Recommendation != "" {
					blockerMsg += fmt.Sprintf(" → %s", result.Recommendation)
				}
				blockers = append(blockers, blockerMsg)
			}
		}
	}

	if len(blockers) == 0 {
		return "  No specific blocker details available"
	}

	return "\nBlocker Details:\n" + strings.Join(blockers, "\n")
}

// displayBatchTransferSummary shows dry-run results for multiple repositories
func displayBatchTransferSummary(results []transferResult) error {
	fmt.Printf("🔍 DRY RUN: Batch repository transfer simulation\n")
	fmt.Printf("═════════════════════════════════════════════════\n")
	
	successCount := 0
	blockedCount := 0
	enforcedCount := 0
	
	for _, result := range results {
		status := "❌ FAIL"
		if result.Success {
			status = "✅ SUCCESS"
			successCount++
		} else if result.Mode == "BLOCKED" {
			blockedCount++
		}
		
		if result.Mode == "ENFORCED" {
			enforcedCount++
		}
		
		fmt.Printf("%-50s %s (%s)\n", result.Repository, status, result.Mode)
		if !result.Success && result.Error != nil {
			fmt.Printf("  └─ %v\n", result.Error)
		}
	}
	
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total repositories: %d\n", len(results))
	fmt.Printf("  Would succeed: %d\n", successCount)
	fmt.Printf("  Would fail: %d\n", len(results)-successCount)
	if blockedCount > 0 {
		fmt.Printf("  Blocked by validation: %d\n", blockedCount)
	}
	if enforcedCount > 0 {
		fmt.Printf("  Enforced transfers: %d\n", enforcedCount)
	}
	fmt.Printf("  Target: %s\n", targetOrg)
	
	return nil
}

// handleBatchTransferResults processes actual transfer results
func handleBatchTransferResults(client api.RESTClient, results []transferResult) error {
	successCount := 0
	var failures []string
	
	for _, result := range results {
		if result.Success {
			successCount++
			// Perform actual transfer
			if verbose {
				fmt.Fprintf(os.Stderr, "Executing transfer for %s...\n", result.Repository)
			}
			
			// Determine which teams to include in transfer
			var teamsForTransfer []string
			if assign {
				// Use pre-collected teams from validation phase
				if verbose {
					fmt.Fprintf(os.Stderr, "Using %d teams collected during validation...\n", len(result.Teams))
				}
				for _, teamName := range result.Teams {
					if verbose {
						fmt.Fprintf(os.Stderr, "Processing team: '%s'\n", teamName)
					}
					// Only include teams that exist in target org (when --enforce) or all teams
					if enforce {
						if teamExistsInTargetOrg(client, targetOrg, teamName) {
							teamsForTransfer = append(teamsForTransfer, teamName)
							if verbose {
								fmt.Fprintf(os.Stderr, "Including team '%s' (exists in target org)\n", teamName)
							}
						} else {
							if verbose {
								fmt.Fprintf(os.Stderr, "Skipping team '%s' (does not exist in target org)\n", teamName)
							}
						}
					} else {
						teamsForTransfer = append(teamsForTransfer, teamName)
						if verbose {
							fmt.Fprintf(os.Stderr, "Including team '%s' for transfer\n", teamName)
						}
					}
				}
			} else {
				// Use teams from CLI flags (teamIds)
				teamsForTransfer = teamIds
			}
			
			if err := executeTransfer(client, result.Owner, result.RepoName, targetOrg, teamsForTransfer, assign); err != nil {
				failures = append(failures, fmt.Sprintf("%s: transfer execution failed: %v", result.Repository, err))
				successCount-- // Decrement since this actually failed
			}
		} else {
			failures = append(failures, fmt.Sprintf("%s: %v", result.Repository, result.Error))
		}
	}
	
	if len(failures) > 0 {
		fmt.Printf("❌ Batch transfer completed with %d/%d failures:\n", len(failures), len(results))
		for _, failure := range failures {
			fmt.Printf("  - %s\n", failure)
		}
		return fmt.Errorf("batch transfer had %d failures", len(failures))
	}
	
	fmt.Printf("✅ Successfully transferred %d repositories to %s\n", successCount, targetOrg)
	return nil
}

// getTeamIdByName looks up a team ID by name in the target organization
func getTeamIdByName(client api.RESTClient, targetOrg, teamName string) (int, error) {
	// Convert team name to slug format (lowercase, replace spaces with hyphens)
	teamSlug := strings.ToLower(strings.ReplaceAll(teamName, " ", "-"))

	var team struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/teams/%s", targetOrg, teamSlug), &team)
	if err != nil {
		return 0, err
	}

	return team.ID, nil
}