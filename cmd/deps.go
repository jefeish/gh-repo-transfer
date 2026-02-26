package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/spf13/cobra"
	
	"github.com/jefeish/gh-repo-transfer/internal/analyzer"
	"github.com/jefeish/gh-repo-transfer/internal/batch"
	"github.com/jefeish/gh-repo-transfer/internal/output"
	"github.com/jefeish/gh-repo-transfer/internal/types"
	"github.com/jefeish/gh-repo-transfer/internal/validation"
)

// depsCmd represents the deps command
var depsCmd = &cobra.Command{
	Use:   "deps [owner/repo...]",
	Short: "Analyze organizational dependencies",
	Long: `Analyze all organizational dependencies that would need to be addressed 
when moving repository(ies) to another organization. This includes:

1. Organization-Specific Code Dependencies
2. GitHub Actions & CI/CD Dependencies  
3. Access Control & Permissions
4. Security & Compliance Dependencies
5. GitHub Apps & Integrations
6. Organizational Governance

Multiple repositories can be specified for batch analysis:
  gh repo-transfer deps owner/repo1 owner/repo2 owner/repo3
  
When analyzing multiple repositories from the same organization, 
organization-level data is cached to reduce API calls.

When --target-org is specified, automatic migration validation is performed
against the target organization's capabilities.`,
	RunE: runDepsAnalysis,
}

var targetOrgLocal string
var separateFilesLocal bool

func init() {
	rootCmd.AddCommand(depsCmd)
	// Flags are now defined as persistent flags in root.go
}

func runDepsAnalysis(cmd *cobra.Command, args []string) error {
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

	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create API client: %v", err)
	}

	// Group repositories by organization for efficient batch processing
	orgRepos := groupReposByOrganization(repos)

	if verbose {
		if len(repos) == 1 {
			fmt.Fprintf(os.Stderr, "Analyzing organizational dependencies for repository: %s\n", repos[0])
		} else {
			fmt.Fprintf(os.Stderr, "Analyzing organizational dependencies for %d repositories across %d organizations\n", 
				len(repos), len(orgRepos))
		}
	}

	// Process repositories with batch optimization when multiple repos from same org
	var allDeps []*types.OrganizationalDependencies
	
	for orgName, orgRepoList := range orgRepos {
		if len(orgRepoList) == 1 {
			// Single repository - use standard analysis
			parts := strings.Split(orgRepoList[0], "/")
			owner, repoName := parts[0], parts[1]
			
			deps, err := analyzer.AnalyzeOrganizationalDependencies(*client, owner, repoName, verbose)
			if err != nil {
				return fmt.Errorf("failed to analyze organizational dependencies for %s: %v", orgRepoList[0], err)
			}
			allDeps = append(allDeps, deps)
		} else {
			// Multiple repositories from same organization - use batch analysis
			if verbose {
				fmt.Fprintf(os.Stderr, "Using batch analysis for %d repositories in organization %s\n", 
					len(orgRepoList), orgName)
			}
			
			batchAnalyzer := batch.NewBatchAnalyzer(*client, verbose)
			orgResults, err := batchAnalyzer.AnalyzeRepositories(orgRepoList)
			if err != nil {
				return fmt.Errorf("failed to batch analyze repositories for organization %s: %v", orgName, err)
			}
			
			// Convert BatchAnalysisResult to OrganizationalDependencies
			for _, result := range orgResults {
				if result.Error != nil {
					return fmt.Errorf("failed to analyze repository %s: %v", result.Repository, result.Error)
				}
				allDeps = append(allDeps, result.Result)
			}
		}
	}

	// If target organization is specified, perform validation for all repositories
	if targetOrg != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "Performing validation against target organization: %s\n", targetOrg)
		}
		
		capabilities, err := validation.ScanTargetOrganization(*client, targetOrg, verbose)
		if err != nil {
			return fmt.Errorf("failed to scan target organization: %v", err)
		}
		
		for _, deps := range allDeps {
			deps.Validation = validation.ValidateAgainstTarget(deps, capabilities, false)
		}
	}

	// Output results
	if separateFiles {
		// Output each repository to separate JSON files
		return output.OutputSeparateFiles(allDeps, verbose)
	} else if len(allDeps) == 1 {
		// Single repository output
		return output.OutputDependencies(allDeps[0], outputFormat)
	} else {
		// Multiple repositories output
		return output.OutputMultipleDependencies(allDeps, outputFormat)
	}
}

// groupReposByOrganization groups repositories by their organization for batch processing
func groupReposByOrganization(repos []string) map[string][]string {
	orgRepos := make(map[string][]string)
	
	for _, repo := range repos {
		parts := strings.Split(repo, "/")
		owner := parts[0]
		
		if orgRepos[owner] == nil {
			orgRepos[owner] = make([]string, 0)
		}
		orgRepos[owner] = append(orgRepos[owner], repo)
	}
	
	return orgRepos
}

func getCurrentRepo() (string, error) {
	client, err := api.DefaultRESTClient()
	if err != nil {
		return "", err
	}

	// This is a simplified approach - in a real implementation,
	// you might want to parse .git/config or use git commands
	response := struct {
		FullName string `json:"full_name"`
	}{}

	err = client.Get("user/repos", &response)
	if err != nil {
		return "", err
	}

	return response.FullName, nil
}