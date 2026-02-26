package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// assignTeamsToTransferredRepo creates teams in target org and assigns them to the repository
// When enforceMode=true, only assigns teams that already exist in target org
func assignTeamsToTransferredRepo(client api.RESTClient, sourceOwner, repoName, targetOwner string, enforceMode bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "Retrieving team information from source repository...\n")
	}

	// Get teams from source repository
	sourceTeams, err := getRepositoryTeams(client, sourceOwner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get teams from source repository: %v", err)
	}

	if len(sourceTeams) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "No teams found in source repository to assign\n")
		}
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d teams in source repository\n", len(sourceTeams))
	}

	// Create teams in target organization and assign to repository
	for _, team := range sourceTeams {
		if enforceMode {
			// In enforce mode, only assign teams that already exist in target org
			if !teamExistsInTargetOrg(client, targetOwner, team.Name) {
				if verbose {
					fmt.Fprintf(os.Stderr, "Skipping team '%s' (does not exist in target org)\n", team.Name)
				}
				continue
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "Team '%s' exists in target org, proceeding with assignment\n", team.Name)
			}
		} else {
			// Normal mode: try to create teams if they don't exist
			if err := createOrUpdateTeamInTargetOrg(client, targetOwner, team); err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "Warning: failed to create team '%s' in target org: %v\n", team.Name, err)
				}
				continue
			}
		}

		if err := assignTeamToRepository(client, targetOwner, team.Name, repoName, team.Permission); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to assign team '%s' to repository: %v\n", team.Name, err)
			}
			continue
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "✅ Successfully assigned team '%s' with '%s' permission\n", team.Name, team.Permission)
		}
	}

	return nil
}

// getRepositoryTeams retrieves teams from a repository
func getRepositoryTeams(client api.RESTClient, owner, repo string) ([]types.Team, error) {
	var teams []struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Permission  string  `json:"permission"`
		RoleName    *string `json:"role_name"` // Custom organization role
		Permissions struct {
			Pull  bool `json:"pull"`
			Push  bool `json:"push"`
			Admin bool `json:"admin"`
		} `json:"permissions"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/teams", owner, repo), &teams)
	if err != nil {
		return nil, err
	}

	var result []types.Team
	for _, team := range teams {
		// Determine permission/role
		permission := team.Permission
		if permission == "" && team.RoleName != nil && *team.RoleName != "" {
			// Use custom role name if available
			permission = *team.RoleName
		} else if permission == "" {
			// Fallback to inferring from permissions object
			if team.Permissions.Admin {
				permission = "admin"
			} else if team.Permissions.Push {
				permission = "push"
			} else if team.Permissions.Pull {
				permission = "pull"
			} else {
				permission = "read"
			}
		}

		result = append(result, types.Team{
			Name:       team.Name,
			Permission: permission,
		})
	}

	return result, nil
}

// createOrUpdateTeamInTargetOrg creates a team in the target organization if it doesn't exist
func createOrUpdateTeamInTargetOrg(client api.RESTClient, targetOrg string, team types.Team) error {
	// First check if team already exists
	var existingTeam struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	}

	// Convert team name to slug format (lowercase, replace spaces with hyphens)
	teamSlug := strings.ToLower(strings.ReplaceAll(team.Name, " ", "-"))

	err := client.Get(fmt.Sprintf("orgs/%s/teams/%s", targetOrg, teamSlug), &existingTeam)
	if err == nil {
		// Team already exists
		if verbose {
			fmt.Fprintf(os.Stderr, "Team '%s' already exists in target organization\n", team.Name)
		}
		return nil
	}

	// Create the team
	createTeamPayload := map[string]interface{}{
		"name":        team.Name,
		"description": fmt.Sprintf("Team migrated from source repository"),
		"privacy":     "closed", // Default to closed privacy
	}

	payloadBytes, err := json.Marshal(createTeamPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal team creation payload: %v", err)
	}

	var createdTeam struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	}

	err = client.Post(fmt.Sprintf("orgs/%s/teams", targetOrg), bytes.NewBuffer(payloadBytes), &createdTeam)
	if err != nil {
		return fmt.Errorf("failed to create team: %v", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "✅ Created team '%s' in target organization\n", team.Name)
	}

	return nil
}

// assignTeamToRepository assigns a team to a repository with specified permissions
func assignTeamToRepository(client api.RESTClient, targetOrg, teamName, repoName, permission string) error {
	// Convert team name to slug format
	teamSlug := strings.ToLower(strings.ReplaceAll(teamName, " ", "-"))

	// GitHub API permission mapping
	apiPermission := permission
	switch permission {
	case "read":
		apiPermission = "pull"
	case "write":
		apiPermission = "push"
	case "admin":
		apiPermission = "admin"
	case "maintain":
		apiPermission = "maintain"
	case "triage":
		apiPermission = "triage"
	default:
		// For custom roles, keep the original permission name
		apiPermission = permission
	}

	endpoint := fmt.Sprintf("orgs/%s/teams/%s/repos/%s/%s", targetOrg, teamSlug, targetOrg, repoName)
	payload := fmt.Sprintf(`{"permission":"%s"}`, apiPermission)
	
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Team assignment API call:\n")
		fmt.Fprintf(os.Stderr, "  Endpoint: PUT %s\n", endpoint)
		fmt.Fprintf(os.Stderr, "  Payload: %s\n", payload)
		fmt.Fprintf(os.Stderr, "  Team: %s → %s\n", teamName, teamSlug)
		fmt.Fprintf(os.Stderr, "  Permission: %s → %s\n", permission, apiPermission)
	}

	// Use gh CLI directly since we know it works
	cmd := exec.Command("gh", "api", "-X", "PUT", endpoint, "--input", "-")
	cmd.Stdin = strings.NewReader(payload)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to assign team via gh CLI (endpoint: %s, payload: %s): %v - Output: %s", endpoint, payload, err, string(output))
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] Team assignment successful via gh CLI!\n")
	}

	return nil
}

// teamExistsInTargetOrg checks if a team exists in the target organization
func teamExistsInTargetOrg(client api.RESTClient, targetOrg, teamName string) bool {
	var existingTeam struct {
		ID   int    `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	}

	// Convert team name to slug format (lowercase, replace spaces with hyphens)
	teamSlug := strings.ToLower(strings.ReplaceAll(teamName, " ", "-"))

	err := client.Get(fmt.Sprintf("orgs/%s/teams/%s", targetOrg, teamSlug), &existingTeam)
	return err == nil
}

// createTeamsInTargetOrg creates teams in target org that don't already exist (Step 0)
func createTeamsInTargetOrg(client api.RESTClient, sourceOwner, repoName, targetOrg string, sourceTeamPermissions []types.Team) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "🔨 Step 0: Creating teams in target org '%s' (if they don't exist)...\n", targetOrg)
	}

	if len(sourceTeamPermissions) == 0 {
		if verbose {
			fmt.Fprintf(os.Stderr, "No teams found in source repository to create\n")
		}
		return nil
	}

	createdCount := 0
	skippedCount := 0

	for _, team := range sourceTeamPermissions {
		// Check if team already exists in target org
		if teamExistsInTargetOrg(client, targetOrg, team.Name) {
			if verbose {
				fmt.Fprintf(os.Stderr, "✅ Team '%s' already exists in target org\n", team.Name)
			}
			skippedCount++
			continue
		}

		// Create team in target org
		if verbose {
			fmt.Fprintf(os.Stderr, "Creating team '%s' in target org...\n", team.Name)
		}

		err := createTeamInOrg(client, targetOrg, team.Name)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to create team '%s': %v\n", team.Name, err)
			}
			continue
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "✅ Successfully created team '%s' in target org\n", team.Name)
		}
		createdCount++
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "🔨 Step 0 completed: Created %d teams, %d already existed\n", createdCount, skippedCount)
	}

	return nil
}

// createTeamInOrg creates a new team in the specified organization
func createTeamInOrg(client api.RESTClient, targetOrg, teamName string) error {
	// Create team payload
	createPayload := map[string]interface{}{
		"name":    teamName,
		"privacy": "closed", // Default to closed for security
	}

	payloadBytes, err := json.Marshal(createPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal team creation payload: %v", err)
	}

	// Use gh CLI to create team (since we know it works)
	cmd := exec.Command("gh", "api", "-X", "POST", fmt.Sprintf("orgs/%s/teams", targetOrg), "--input", "-")
	cmd.Stdin = strings.NewReader(string(payloadBytes))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create team via gh CLI: %v - Output: %s", err, string(output))
	}

	return nil
}