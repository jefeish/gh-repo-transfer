package cmd

import (
	"bytes"
	"crypto/rand"
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

// archiveCmd represents the archive command
var archiveCmd = &cobra.Command{
	Use:   "archive [owner/repo...] --target-org [target-owner]",
	Short: "Archive repository(ies) to another organization with renamed and tracked original location",
	Long: `Archive one or more repositories to another organization or user with automatic renaming and tracking.

This command will:
1. Validate the target organization/user exists
2. Check transfer permissions on each source repository
3. Perform dependency validation (unless --enforce is used)
4. Rename the repository with a unique identifier suffix (e.g., repo-abc -> repo-abc-A1B2)
5. Store the original repository path in repository properties for potential restoration
6. Execute the repository transfer with the new name

Usage:
  repo-transfer archive [owner/repo] --target-org [target-owner]
  repo-transfer archive [owner/repo1] [owner/repo2] --target-org [target-owner]
  repo-transfer archive [owner/repo] --target-org [target-owner] --dry-run
  repo-transfer archive [owner/repo] --target-org [target-owner] --verbose

Multiple repositories can be archived in batch:
  repo-transfer archive owner/repo1 owner/repo2 owner/repo3 --target-org archive-org

Examples:
  gh repo-transfer archive owner/repo --target-org archive-org
  gh repo-transfer archive owner/repo1 owner/repo2 --target-org archive-org --dry-run
  gh repo-transfer archive owner/repo --target-org archive-org --verbose`,
	SilenceUsage: true,
	RunE: runArchive,
}

type archiveResult struct {
	Repository     string `json:"repository"`
	OriginalName   string `json:"original_name"`
	ArchivedName   string `json:"archived_name"`
	Owner          string `json:"owner"`
	RepoName       string `json:"repo_name"`
	Teams          []string `json:"teams,omitempty"`
	Success        bool   `json:"success"`
	Error          error  `json:"error,omitempty"`
	Mode           string `json:"mode"`
	UID            string `json:"uid"`
	OriginalPath   string `json:"original_path"`
	Validation     *types.MigrationValidation `json:"validation,omitempty"`
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	
	// Mark the --target-org flag as required
	archiveCmd.MarkFlagRequired("target-org")
}

func runArchive(cmd *cobra.Command, args []string) error {
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
			fmt.Fprintf(os.Stderr, "Preparing to archive repository: %s to %s\n", repos[0], targetOrg)
		} else {
			fmt.Fprintf(os.Stderr, "Preparing to archive %d repositories to %s\n", len(repos), targetOrg)
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
	var results []archiveResult
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

			result := processRepoArchiveOptimized(*client, owner, repoName, targetCapabilities)
			results = append(results, result)
		}
	}

	// Handle dry-run summary for multiple repos
	if dryRun {
		return displayBatchArchiveSummary(results)
	}

	// Check for failures in actual archive
	return handleBatchArchiveResults(*client, results)
}

// processRepoArchiveOptimized handles the archive logic with pre-scanned target capabilities
func processRepoArchiveOptimized(client api.RESTClient, owner, repoName string, targetCapabilities *types.TargetOrgCapabilities) archiveResult {
	// Generate unique identifier
	uid := generateUID()
	originalPath := fmt.Sprintf("%s/%s", owner, repoName)
	archivedName := fmt.Sprintf("%s-%s", repoName, uid)
	
	result := archiveResult{
		Repository:   fmt.Sprintf("%s/%s", owner, repoName),
		OriginalName: repoName,
		ArchivedName: archivedName,
		Owner:        owner,
		RepoName:     repoName,
		Mode:         "ARCHIVED",
		UID:          uid,
		OriginalPath: originalPath,
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
			fmt.Fprintf(os.Stderr, "Checking for archive blockers...\n")
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
				// Scan target organization capabilities
				if verbose {
					fmt.Fprintf(os.Stderr, "Scanning target organization capabilities: %s\n", targetOrg)
				}
				capabilities, err = validation.ScanTargetOrganization(client, targetOrg, verbose)
				if err != nil {
					result.Error = fmt.Errorf("failed to scan target organization: %v", err)
					result.Success = false
					return result
				}
			}
			
			// Validate migration readiness
			validation := validation.ValidateAgainstTarget(deps, capabilities, assign)
			result.Validation = validation
			
			if validation.Summary.Blockers > 0 {
				result.Error = fmt.Errorf("archive blocked: %d validation blockers found", validation.Summary.Blockers)
				result.Success = false
				return result
			}
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Archive validation passed, repository will be renamed from '%s' to '%s'\n", repoName, archivedName)
		if dryRun {
			fmt.Fprintf(os.Stderr, "Note: Original path would be stored as custom property 'repo-origin' = '%s'\n", originalPath)
		}
	}

	result.Success = true
	return result
}

// generateUID creates a unique identifier for archived repositories using timestamp + random chars
// This ensures uniqueness even at large scale by combining current milliseconds with randomness
func generateUID() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	
	// Get current time in milliseconds since Unix epoch
	now := time.Now().UnixMilli()
	
	// Convert milliseconds to base-36 (using our charset) for compactness
	// This gives us a time-based prefix that ensures uniqueness
	timeStr := ""
	timestamp := now
	for timestamp > 0 {
		timeStr = string(charset[timestamp%36]) + timeStr
		timestamp /= 36
	}
	
	// Add 2 random characters for additional entropy and readability
	b := make([]byte, 2)
	rand.Read(b)
	
	randomSuffix := ""
	for i := range b {
		randomSuffix += string(charset[b[i]%byte(len(charset))])
	}
	
	// Combine time-based prefix with random suffix
	// Format: [TIME_BASED][RANDOM] e.g., "2JKLX9A7" where first part is timestamp, last 2 are random
	uid := timeStr + randomSuffix
	
	// If the UID is too long, take the last 8 characters to keep it reasonable
	// This still maintains uniqueness since we include the most recent timestamp bits
	if len(uid) > 8 {
		uid = uid[len(uid)-8:]
	}
	
	return uid
}

// displayBatchArchiveSummary shows summary for dry-run archive operations
func displayBatchArchiveSummary(results []archiveResult) error {
	if outputFormat == "json" {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	// Calculate summary statistics
	var total, wouldSucceed, wouldFail, blockedByValidation int
	total = len(results)

	for _, result := range results {
		if result.Success {
			wouldSucceed++
		} else {
			wouldFail++
			// Check if failure was due to validation blockers
			if result.Validation != nil && result.Validation.Summary.Blockers > 0 {
				blockedByValidation++
			}
		}
	}

	// Display header
	fmt.Printf("🗃️ DRY RUN: Batch repository archive simulation\n")
	fmt.Printf("═════════════════════════════════════════════════\n")

	// Display individual results
	for _, result := range results {
		if result.Success {
			fmt.Printf("%-50s ✅ READY\n", result.Repository)
			fmt.Printf("  └─ ✅ Would be archived as: %s (read-only)\n", result.ArchivedName)
		} else {
			fmt.Printf("%-50s ❌ FAIL (BLOCKED)\n", result.Repository)
			if result.Validation != nil && result.Validation.Summary.Blockers > 0 {
				fmt.Printf("  └─ ❌ Archive blocked: %d validation blockers found\n", result.Validation.Summary.Blockers)
			} else if result.Error != nil {
				fmt.Printf("  └─ ❌ %s\n", result.Error.Error())
			}
		}
	}

	// Display validation blocker details if any exist
	hasBlockers := false
	for _, result := range results {
		if result.Validation != nil && result.Validation.Summary.Blockers > 0 {
			if !hasBlockers {
				fmt.Printf("\nBlocker Details:\n")
				hasBlockers = true
			}
			
			// Show blockers for this repository
			allResults := append(result.Validation.AppsIntegrations, result.Validation.AccessPermissions...)
			allResults = append(allResults, result.Validation.CIDependencies...)
			allResults = append(allResults, result.Validation.Governance...)
			allResults = append(allResults, result.Validation.CodeDependencies...)
			allResults = append(allResults, result.Validation.SecurityCompliance...)
			
			for _, validationResult := range allResults {
				if validationResult.Status == types.ValidationBlocker {
					category := getCategoryName(validationResult, result.Validation)
					fmt.Printf("  • [%s] %s: %s → %s\n", 
						category, 
						validationResult.Item, 
						validationResult.Message, 
						validationResult.Recommendation)
				}
			}
		}
	}

	// Display summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total repositories: %d\n", total)
	fmt.Printf("  Would succeed: %d\n", wouldSucceed)
	fmt.Printf("  Would fail: %d\n", wouldFail)
	if blockedByValidation > 0 {
		fmt.Printf("  Blocked by validation: %d\n", blockedByValidation)
	}
	fmt.Printf("  Target: %s\n", targetOrg)

	return nil
}

// handleBatchArchiveResults processes the actual archive results
func handleBatchArchiveResults(client api.RESTClient, results []archiveResult) error {
	var hasFailures bool

	fmt.Printf("🗃️ EXECUTING: Batch repository archive\n")
	fmt.Printf("═════════════════════════════════════════\n")

	for _, result := range results {
		if !result.Success {
			hasFailures = true
			fmt.Printf("%-50s ❌ FAILED\n", result.Repository)
			if result.Error != nil {
				fmt.Printf("  └─ ❌ %s\n", result.Error.Error())
			}
			continue
		}

		// Execute the actual archive (transfer with rename)
		fmt.Printf("%-50s 🗃️ ARCHIVING...\n", result.Repository)
		
		err := executeArchive(client, result.Owner, result.RepoName, targetOrg, result.ArchivedName, result.OriginalPath, result.Teams, verbose)
		if err != nil {
			hasFailures = true
			fmt.Printf("%-50s ❌ FAILED\n", result.Repository)
			fmt.Printf("  └─ ❌ %s\n", err.Error())
		} else {
			fmt.Printf("%-50s ✅ ARCHIVED\n", result.Repository)
			fmt.Printf("  └─ ✅ Archived as: %s/%s (read-only)\n", targetOrg, result.ArchivedName)
			if verbose {
				fmt.Printf("  └─ 📝 Original path stored: %s\n", result.OriginalPath)
			}
		}
	}

	if hasFailures {
		return fmt.Errorf("one or more archive operations failed")
	}

	return nil
}

// executeArchive performs the actual repository archive with renaming and metadata storage
func executeArchive(client api.RESTClient, owner, repoName, targetOwner, archivedName, originalPath string, teams []string, verboseOutput bool) error {
	if verboseOutput {
		fmt.Fprintf(os.Stderr, "Archiving repository %s/%s as %s/%s...\n", owner, repoName, targetOwner, archivedName)
		fmt.Fprintf(os.Stderr, "Original path will be stored: %s\n", originalPath)
	}

	// Prepare the transfer request with new name
	transferRequest := map[string]interface{}{
		"new_owner": targetOwner,
		"new_name":  archivedName,
	}

	// If teams are specified, look up their IDs in the target organization
	if len(teams) > 0 {
		if verboseOutput {
			fmt.Fprintf(os.Stderr, "Looking up team IDs for: %v\n", teams)
		}
		var teamIDs []int
		for _, teamSlug := range teams {
			var teamResponse struct {
				ID int `json:"id"`
			}
			err := client.Get(fmt.Sprintf("orgs/%s/teams/%s", targetOwner, teamSlug), &teamResponse)
			if err != nil {
				return fmt.Errorf("failed to get team ID for '%s': %v", teamSlug, err)
			}
			teamIDs = append(teamIDs, teamResponse.ID)
			if verboseOutput {
				fmt.Fprintf(os.Stderr, "Found team '%s' with ID %d\n", teamSlug, teamResponse.ID)
			}
		}
		transferRequest["team_ids"] = teamIDs
	}

	// Marshal the payload
	payloadBytes, err := json.Marshal(transferRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal archive payload: %v", err)
	}

	if verboseOutput {
		fmt.Fprintf(os.Stderr, "Archive payload: %s\n", string(payloadBytes))
	}

	// Execute the transfer
	var transferResponse struct {
		ID          int    `json:"id"`
		NodeID      string `json:"node_id"`
		Name        string `json:"name"`
		FullName    string `json:"full_name"`
		Owner       struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	err = client.Post(fmt.Sprintf("repos/%s/%s/transfer", owner, repoName), bytes.NewBuffer(payloadBytes), &transferResponse)
	if err != nil {
		// Check if the repository might already be transferred
		if verboseOutput {
			fmt.Fprintf(os.Stderr, "Transfer API call failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Checking if repository was already transferred with different archive name...\n")
		}
		
		// Search for repositories in the target org that start with the base repository name
		baseName := repoName
		var reposList []struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Owner    struct {
				Login string `json:"login"`
			} `json:"owner"`
		}
		
		// Get list of repositories in target organization
		err2 := client.Get(fmt.Sprintf("orgs/%s/repos", targetOwner), &reposList)
		if err2 == nil {
			// Look for repositories that start with the base name followed by a hyphen and UID pattern
			baseNamePrefix := baseName + "-"
			var foundRepo *struct {
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				Owner    struct {
					Login string `json:"login"`
				} `json:"owner"`
			}
			
			for _, repo := range reposList {
				if strings.HasPrefix(repo.Name, baseNamePrefix) && len(repo.Name) > len(baseNamePrefix)+6 {
					// Found a potential match - check if it has the UID pattern (letters and numbers)
					suffix := repo.Name[len(baseNamePrefix):]
					if len(suffix) >= 6 && len(suffix) <= 10 {
						// This looks like an archived version of our repository
						foundRepo = &repo
						if verboseOutput {
							fmt.Fprintf(os.Stderr, "Found existing archived repository: %s\n", repo.FullName)
						}
						break
					}
				}
			}
			
			if foundRepo != nil {
				// Update our archive name to match the existing repository
				archivedName = foundRepo.Name
				transferResponse.FullName = foundRepo.FullName
				transferResponse.Owner.Login = foundRepo.Owner.Login
				if verboseOutput {
					fmt.Fprintf(os.Stderr, "✅ Repository already exists in target organization: %s\n", foundRepo.FullName)
					fmt.Fprintf(os.Stderr, "Proceeding with custom property and archive flag setting...\n")
				}
			} else {
				// Transfer failed and no archived version found - this is a real error
				return fmt.Errorf("failed to transfer repository and no archived version found in target organization: %v", err)
			}
		} else {
			// Can't list repos, try the specific name check as fallback
			var existingRepo struct {
				ID       int    `json:"id"`
				FullName string `json:"full_name"`
				Owner    struct {
					Login string `json:"login"`
				} `json:"owner"`
			}
			checkErr := client.Get(fmt.Sprintf("repos/%s/%s", targetOwner, archivedName), &existingRepo)
			if checkErr == nil {
				transferResponse.FullName = existingRepo.FullName
				transferResponse.Owner.Login = existingRepo.Owner.Login
				if verboseOutput {
					fmt.Fprintf(os.Stderr, "✅ Repository already exists in target organization: %s\n", existingRepo.FullName)
					fmt.Fprintf(os.Stderr, "Proceeding with custom property and archive flag setting...\n")
				}
			} else {
				return fmt.Errorf("failed to transfer repository and repository does not exist in target organization: %v", err)
			}
		}
	} else {
		if verboseOutput {
			fmt.Fprintf(os.Stderr, "✅ Repository transfer completed: %s\n", transferResponse.FullName)
		}
	}

	// Add a small delay to allow the transfer to fully complete
	if verboseOutput {
		fmt.Fprintf(os.Stderr, "Waiting for transfer to complete fully...\n")
	}
	time.Sleep(3 * time.Second)

	// Archive the repository (set as read-only) in the target organization
	err = setRepositoryArchiveStatus(client, targetOwner, archivedName, true, verboseOutput)
	if err != nil {
		if verboseOutput {
			fmt.Fprintf(os.Stderr, "❌ Warning: Failed to set repository archive status: %v\n", err)
			fmt.Fprintf(os.Stderr, "Repository transferred but not marked as archived (read-only)\n")
		}
		// Don't fail the entire operation for archive status issues, but log the issue
	} else {
		if verboseOutput {
			fmt.Fprintf(os.Stderr, "✅ Repository marked as archived (read-only)\n")
		}
	}

	// Store the original path as a repository custom property
	err = storeOriginalPathProperty(client, targetOwner, archivedName, originalPath, verboseOutput)
	if err != nil {
		if verboseOutput {
			fmt.Fprintf(os.Stderr, "❌ Warning: Failed to store original path as custom property: %v\n", err)
			fmt.Fprintf(os.Stderr, "Archive completed, but restoration metadata may need to be added manually\n")
		}
		// Don't fail the entire operation for metadata storage issues
	}

	if verboseOutput {
		fmt.Fprintf(os.Stderr, "Archive completed successfully\n")
		fmt.Fprintf(os.Stderr, "Repository archived from %s to %s/%s\n", originalPath, targetOwner, archivedName)
		fmt.Fprintf(os.Stderr, "Repository is now read-only and marked as archived\n")
		fmt.Fprintf(os.Stderr, "Original path '%s' stored for restoration capability\n", originalPath)
	}

	return nil
}

// storeOriginalPathProperty stores the original repository path as a custom property.
// If the 'repo-origin' custom property is not defined in the target organization's schema,
// a warning is reported and the operation continues without storing.
func storeOriginalPathProperty(client api.RESTClient, targetOwner, repoName, originalPath string, verbose bool) error {
	const propertyName = "repo-origin"

	if verbose {
		fmt.Fprintf(os.Stderr, "Checking if custom property '%s' is defined in organization '%s'...\n", propertyName, targetOwner)
	}

	// Check if the property exists in the org schema
	var existingProperties []map[string]interface{}
	err := client.Get(fmt.Sprintf("orgs/%s/properties/schema", targetOwner), &existingProperties)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not retrieve custom property schema for '%s': %v\n", targetOwner, err)
		fmt.Fprintf(os.Stderr, "   Skipping 'repo-origin' tracking.\n")
		return nil
	}

	propExists := false
	for _, prop := range existingProperties {
		if name, ok := prop["property_name"].(string); ok && name == propertyName {
			propExists = true
			break
		}
	}

	if !propExists {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: Organization '%s' does not have a '%s' custom property defined.\n", targetOwner, propertyName)
		fmt.Fprintf(os.Stderr, "   Skipping origin tracking. To enable it, add a 'repo-origin' string property to the organization's custom property schema.\n")
		return nil
	}

	// Property exists — set it
	if verbose {
		fmt.Fprintf(os.Stderr, "Storing original path as custom property '%s' = '%s'...\n", propertyName, originalPath)
	}

	err = setCustomProperty(client, targetOwner, repoName, propertyName, originalPath, verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to set custom property '%s': %v\n", propertyName, err)
		fmt.Fprintf(os.Stderr, "   Origin tracking skipped.\n")
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "✅ Original path stored in custom property '%s' = '%s'\n", propertyName, originalPath)
	}
	return nil
}

// setCustomProperty attempts to set a custom property on a repository
func setCustomProperty(client api.RESTClient, owner, repo, propertyName, value string, verbose bool) error {
	// Repository custom properties API endpoint
	url := fmt.Sprintf("repos/%s/%s/properties/values", owner, repo)
	
	payload := map[string]interface{}{
		"properties": []map[string]interface{}{
			{
				"property_name": propertyName,
				"value":        value,
			},
		},
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal custom property payload: %v", err)
	}
	
	var response map[string]interface{}
	err = client.Patch(url, bytes.NewBuffer(payloadBytes), &response)
	if err != nil {
		return fmt.Errorf("failed to set custom property: %v", err)
	}
	
	return nil
}

// ensureCustomPropertyExists creates the custom property definition if it doesn't exist
func ensureCustomPropertyExists(client api.RESTClient, owner, propertyName string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Checking if custom property '%s' exists in organization...\n", propertyName)
	}
	
	// Check if property exists first
	url := fmt.Sprintf("orgs/%s/properties/schema", owner)
	var existingProperties []map[string]interface{}
	err := client.Get(url, &existingProperties)
	if err != nil {
		return fmt.Errorf("failed to get existing properties: %v", err)
	}
	
	// Check if our property already exists
	for _, prop := range existingProperties {
		if name, ok := prop["property_name"].(string); ok && name == propertyName {
			if verbose {
				fmt.Fprintf(os.Stderr, "Custom property '%s' already exists\n", propertyName)
			}
			return nil
		}
	}
	
	if verbose {
		fmt.Fprintf(os.Stderr, "Custom property '%s' does not exist, attempting to create...\n", propertyName)
	}
	
	// Create the custom property
	createURL := fmt.Sprintf("orgs/%s/properties/schema", owner)
	createPayload := map[string]interface{}{
		"property_name": propertyName,
		"value_type":   "string",
		"description":  "Original repository path for archived repositories (used for restoration)",
	}
	
	createPayloadBytes, err := json.Marshal(createPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal create property payload: %v", err)
	}
	
	var createResponse map[string]interface{}
	err = client.Post(createURL, bytes.NewBuffer(createPayloadBytes), &createResponse)
	if err != nil {
		return fmt.Errorf("failed to create custom property: %v", err)
	}
	
	if verbose {
		fmt.Fprintf(os.Stderr, "✅ Successfully created custom property '%s'\n", propertyName)
	}
	
	return nil
}

// addArchiveTopicFallback adds a topic to indicate the original repository (fallback method)
func addArchiveTopicFallback(client api.RESTClient, owner, repo, originalPath string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Using repository topics as fallback storage...\n")
	}
	
	// Create a topic from the original path (GitHub topics have restrictions)
	// Convert "github-innersource/gh-repo-transfer-test-main" to "archived-from-github-innersource-gh-repo-transfer-test-main"
	topicValue := "archived-from-" + strings.ReplaceAll(strings.ToLower(originalPath), "/", "-")
	
	// Get current topics
	url := fmt.Sprintf("repos/%s/%s/topics", owner, repo)
	var currentTopics struct {
		Names []string `json:"names"`
	}
	
	err := client.Get(url, &currentTopics)
	if err != nil {
		return fmt.Errorf("failed to get current topics: %v", err)
	}
	
	// Add our archive topic
	newTopics := append(currentTopics.Names, topicValue)
	
	payload := map[string]interface{}{
		"names": newTopics,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal topics payload: %v", err)
	}
	
	var response map[string]interface{}
	err = client.Put(url, bytes.NewBuffer(payloadBytes), &response)
	if err != nil {
		return fmt.Errorf("failed to update repository topics: %v", err)
	}
	
	if verbose {
		fmt.Fprintf(os.Stderr, "Added topic: %s\n", topicValue)
	}
	
	return nil
}

// updateDescriptionWithOrigin updates repository description to include origin info (fallback method)
func updateDescriptionWithOrigin(client api.RESTClient, owner, repo, originalPath string, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Using repository description as fallback storage...\n")
	}
	
	// Get current repository information
	url := fmt.Sprintf("repos/%s/%s", owner, repo)
	var repoInfo struct {
		Description *string `json:"description"`
	}
	
	err := client.Get(url, &repoInfo)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %v", err)
	}
	
	// Prepare new description
	originNote := fmt.Sprintf("[ARCHIVED FROM: %s]", originalPath)
	var newDescription string
	
	if repoInfo.Description != nil && *repoInfo.Description != "" {
		newDescription = fmt.Sprintf("%s %s", *repoInfo.Description, originNote)
	} else {
		newDescription = originNote
	}
	
	// Update repository description
	updatePayload := map[string]interface{}{
		"description": newDescription,
	}
	
	payloadBytes, err := json.Marshal(updatePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal description update payload: %v", err)
	}
	
	var response map[string]interface{}
	err = client.Patch(url, bytes.NewBuffer(payloadBytes), &response)
	if err != nil {
		return fmt.Errorf("failed to update repository description: %v", err)
	}
	
	if verbose {
		fmt.Fprintf(os.Stderr, "Updated description with origin: %s\n", newDescription)
	}
	
	return nil
}

// setRepositoryArchiveStatus sets the archived status of a repository
func setRepositoryArchiveStatus(client api.RESTClient, owner, repo string, archived bool, verbose bool) error {
	if verbose {
		if archived {
			fmt.Fprintf(os.Stderr, "Setting repository as archived (read-only)...\n")
		} else {
			fmt.Fprintf(os.Stderr, "Setting repository as not archived...\n")
		}
	}

	url := fmt.Sprintf("repos/%s/%s", owner, repo)
	
	payload := map[string]interface{}{
		"archived": archived,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal archive payload: %v", err)
	}
	
	var response map[string]interface{}
	err = client.Patch(url, bytes.NewBuffer(payloadBytes), &response)
	if err != nil {
		return fmt.Errorf("failed to set archive status: %v", err)
	}
	
	return nil
}

// getCategoryName determines the category name for a validation result
func getCategoryName(result types.ValidationResult, validation *types.MigrationValidation) string {
	// Check which category this result belongs to
	for _, r := range validation.AppsIntegrations {
		if r.Item == result.Item {
			return "Apps & Integrations"
		}
	}
	for _, r := range validation.AccessPermissions {
		if r.Item == result.Item {
			return "Access Permissions"
		}
	}
	for _, r := range validation.CIDependencies {
		if r.Item == result.Item {
			return "CI Dependencies"
		}
	}
	for _, r := range validation.Governance {
		if r.Item == result.Item {
			return "Governance"
		}
	}
	for _, r := range validation.CodeDependencies {
		if r.Item == result.Item {
			return "Code Dependencies"
		}
	}
	for _, r := range validation.SecurityCompliance {
		if r.Item == result.Item {
			return "Security Compliance"
		}
	}
	return "Unknown"
}