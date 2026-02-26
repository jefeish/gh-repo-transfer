package dependencies

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// AnalyzeAppsIntegrations analyzes GitHub Apps and integrations dependencies
func AnalyzeAppsIntegrations(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Analyze installed GitHub Apps at the organization level
	if err := analyzeInstalledGitHubApps(client, owner, repo, deps); err != nil {
		// Non-fatal error - GitHub Apps might not be accessible
		// For debugging, let's see what the error is
		fmt.Printf("Debug: GitHub Apps analysis error: %v\n", err)
	}

	// Note: Personal Access Tokens can't be easily detected through the API
	// as they would require access to user settings, which isn't available
	// This would need to be documented as a manual check

	return nil
}

// analyzeInstalledGitHubApps analyzes GitHub Apps installed in the organization
func analyzeInstalledGitHubApps(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Try repository installations first (more reliable)
	if err := analyzeRepoInstallations(client, owner, repo, deps); err != nil {
		// Fallback to organization installations
		if err := analyzeOrgInstallations(client, owner, repo, deps); err != nil {
			return err
		}
	}
	return nil
}

// analyzeRepoInstallations checks GitHub Apps installed for this specific repository
func analyzeRepoInstallations(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// The repository installations API returns an object with an installations array
	var response struct {
		TotalCount    int `json:"total_count"`
		Installations []struct {
			ID  int `json:"id"`
			App struct {
				ID          int    `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
				ExternalURL string `json:"external_url"`
			} `json:"app"`
			Account struct {
				Login string `json:"login"`
				Type  string `json:"type"`
			} `json:"account"`
			RepositorySelection string   `json:"repository_selection"`
			Permissions         struct{} `json:"permissions"`
			Events              []string `json:"events"`
			CreatedAt           string   `json:"created_at"`
			UpdatedAt           string   `json:"updated_at"`
		} `json:"installations"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/installations", owner, repo), &response)
	if err != nil {
		return err
	}

	for _, installation := range response.Installations {
		appInfo := fmt.Sprintf("%s (app ID: %d)", installation.App.Name, installation.App.ID)
		if installation.App.ExternalURL != "" {
			appInfo += fmt.Sprintf(" - %s", installation.App.ExternalURL)
		}
		deps.AppsIntegrations.InstalledGitHubApps = append(deps.AppsIntegrations.InstalledGitHubApps, appInfo)
	}

	return nil
}

// analyzeOrgInstallations checks GitHub Apps installed at organization level (fallback)
func analyzeOrgInstallations(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Use the correct structure based on actual API response
	var response struct {
		TotalCount    int `json:"total_count"`
		Installations []struct {
			ID                  int    `json:"id"`
			AppID               int    `json:"app_id"`
			AppSlug             string `json:"app_slug"`
			RepositorySelection string `json:"repository_selection"`
			Permissions         struct{} `json:"permissions"`
		} `json:"installations"`
	}

	err := client.Get(fmt.Sprintf("orgs/%s/installations", owner), &response)
	if err != nil {
		return err
	}

	for _, installation := range response.Installations {
		appName := installation.AppSlug
		if appName == "" {
			appName = fmt.Sprintf("App ID %d", installation.AppID)
		}

		if installation.RepositorySelection == "all" {
			appInfo := fmt.Sprintf("%s (org-wide installation)", appName)
			deps.AppsIntegrations.InstalledGitHubApps = append(deps.AppsIntegrations.InstalledGitHubApps, appInfo)
		} else {
			// For selective installations, we can't reliably check which specific repos have access
			// via the public API, so we include them with a note for manual verification
			appInfo := fmt.Sprintf("%s (selective installation - verify access)", appName)
			deps.AppsIntegrations.InstalledGitHubApps = append(deps.AppsIntegrations.InstalledGitHubApps, appInfo)
		}
	}

	return nil
}