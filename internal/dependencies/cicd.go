package dependencies

import (
	"encoding/base64"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// AnalyzeActionsCIDependencies analyzes GitHub Actions and CI/CD dependencies
func AnalyzeActionsCIDependencies(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Analyze workflow files
	if err := analyzeWorkflows(client, owner, repo, deps); err != nil {
		// Non-fatal error - .github/workflows might not exist
	}

	// Analyze required workflows from repository rulesets
	if err := analyzeRequiredWorkflows(client, owner, repo, deps); err != nil {
		// Non-fatal error - rulesets might not be accessible
	}

	// Analyze environments (requires special API access)
	if err := analyzeEnvironments(client, owner, repo, deps); err != nil {
		// Non-fatal error - environments might not be accessible
	}

	return nil
}

// analyzeWorkflows analyzes GitHub Actions workflow files
func analyzeWorkflows(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	var contents []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/contents/.github/workflows", owner, repo), &contents)
	if err != nil {
		return err // .github/workflows doesn't exist
	}

	for _, item := range contents {
		if item.Type == "file" && (strings.HasSuffix(item.Name, ".yml") || strings.HasSuffix(item.Name, ".yaml")) {
			if err := analyzeWorkflowFile(client, owner, repo, item.Path, deps); err != nil {
				continue // Skip files that can't be read
			}
		}
	}

	return nil
}

func analyzeWorkflowFile(client api.RESTClient, owner, repo, workflowPath string, deps *types.OrganizationalDependencies) error {
	var content struct {
		Content string `json:"content"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, workflowPath), &content)
	if err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return err
	}

	workflowContent := string(decoded)
	workflowName := path.Base(workflowPath)

	// Check for organization secrets
	analyzeOrganizationSecrets(workflowContent, workflowName, deps)
	
	// Check for organization variables
	analyzeOrganizationVariables(workflowContent, workflowName, deps)
	
	// Check for self-hosted runners
	analyzeSelfHostedRunners(workflowContent, workflowName, deps)
	
	// Check for organization-specific actions
	analyzeOrganizationSpecificActions(workflowContent, workflowName, owner, deps)
	
	// Check for cross-repo workflow triggers
	analyzeCrossRepoTriggers(workflowContent, workflowName, owner, deps)

	return nil
}

func analyzeOrganizationSecrets(content, workflowName string, deps *types.OrganizationalDependencies) {
	// Look for secrets.PATTERN usage
	secretPattern := regexp.MustCompile(`secrets\.([A-Z_][A-Z0-9_]*)`)
	matches := secretPattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			secretName := match[1]
			secretRef := fmt.Sprintf("%s (in %s)", secretName, workflowName)
			// Check if already added
			found := false
			for _, existing := range deps.ActionsCIDependencies.OrganizationSecrets {
				if existing == secretRef {
					found = true
					break
				}
			}
			if !found {
				deps.ActionsCIDependencies.OrganizationSecrets = append(deps.ActionsCIDependencies.OrganizationSecrets, secretRef)
			}
		}
	}
}

func analyzeOrganizationVariables(content, workflowName string, deps *types.OrganizationalDependencies) {
	// Look for vars.PATTERN usage
	varPattern := regexp.MustCompile(`vars\.([A-Z_][A-Z0-9_]*)`)
	matches := varPattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			varRef := fmt.Sprintf("%s (in %s)", varName, workflowName)
			// Check if already added
			found := false
			for _, existing := range deps.ActionsCIDependencies.OrganizationVariables {
				if existing == varRef {
					found = true
					break
				}
			}
			if !found {
				deps.ActionsCIDependencies.OrganizationVariables = append(deps.ActionsCIDependencies.OrganizationVariables, varRef)
			}
		}
	}
}

func analyzeSelfHostedRunners(content, workflowName string, deps *types.OrganizationalDependencies) {
	// Look for runs-on with self-hosted or custom runner labels
	runnerPatterns := []string{
		`runs-on:\s*self-hosted`,
		`runs-on:\s*\[.*self-hosted.*\]`,
		`runs-on:\s*([a-zA-Z][a-zA-Z0-9\-_]*)`, // Custom runner names (not ubuntu-latest, windows-latest, etc.)
	}
	
	for _, pattern := range runnerPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		
		for _, match := range matches {
			runnerInfo := ""
			if strings.Contains(match[0], "self-hosted") {
				runnerInfo = "self-hosted"
			} else if len(match) > 1 && !isGitHubHostedRunner(match[1]) {
				runnerInfo = match[1]
			}
			
			if runnerInfo != "" {
				runnerRef := fmt.Sprintf("Self-hosted runner: %s (in %s)", runnerInfo, workflowName)
				// Check if already added
				found := false
				for _, existing := range deps.ActionsCIDependencies.SelfHostedRunners {
					if existing == runnerRef {
						found = true
						break
					}
				}
				if !found {
					deps.ActionsCIDependencies.SelfHostedRunners = append(deps.ActionsCIDependencies.SelfHostedRunners, runnerRef)
				}
			}
		}
	}
}

func analyzeOrganizationSpecificActions(content, workflowName, owner string, deps *types.OrganizationalDependencies) {
	// Look for actions from the same organization
	actionPattern := regexp.MustCompile(fmt.Sprintf(`uses:\s*%s/([^@\s]+)`, owner))
	matches := actionPattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			actionName := match[1]
			actionRef := fmt.Sprintf("%s/%s (in %s)", owner, actionName, workflowName)
			// Check if already added
			found := false
			for _, existing := range deps.ActionsCIDependencies.OrgSpecificActions {
				if existing == actionRef {
					found = true
					break
				}
			}
			if !found {
				deps.ActionsCIDependencies.OrgSpecificActions = append(deps.ActionsCIDependencies.OrgSpecificActions, actionRef)
			}
		}
	}
}

func analyzeCrossRepoTriggers(content, workflowName, owner string, deps *types.OrganizationalDependencies) {
	// Look for workflow_run or repository_dispatch events targeting same org repos
	triggerPatterns := []string{
		fmt.Sprintf(`repository_dispatch.*%s/`, owner),
		fmt.Sprintf(`workflow_run.*%s/`, owner),
	}
	
	for _, pattern := range triggerPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(content) {
			triggerRef := fmt.Sprintf("Cross-repo trigger (in %s)", workflowName)
			// Check if already added
			found := false
			for _, existing := range deps.ActionsCIDependencies.CrossRepoWorkflowTriggers {
				if existing == triggerRef {
					found = true
					break
				}
			}
			if !found {
				deps.ActionsCIDependencies.CrossRepoWorkflowTriggers = append(deps.ActionsCIDependencies.CrossRepoWorkflowTriggers, triggerRef)
			}
		}
	}
}

// analyzeEnvironments analyzes repository environments for organizational dependencies
func analyzeEnvironments(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Note: This requires special API access and might not be available to all users
	var environments struct {
		Environments []struct {
			Name string `json:"name"`
		} `json:"environments"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/environments", owner, repo), &environments)
	if err != nil {
		return err // Environments not accessible
	}

	for _, env := range environments.Environments {
		envRef := fmt.Sprintf("Environment: %s", env.Name)
		deps.ActionsCIDependencies.EnvironmentDependencies = append(deps.ActionsCIDependencies.EnvironmentDependencies, envRef)
	}

	return nil
}

// isGitHubHostedRunner checks if a runner name is a GitHub-hosted runner
func isGitHubHostedRunner(runner string) bool {
	githubRunners := []string{
		"ubuntu-latest", "ubuntu-22.04", "ubuntu-20.04",
		"windows-latest", "windows-2022", "windows-2019",
		"macos-latest", "macos-14", "macos-13", "macos-12",
	}
	
	for _, gh := range githubRunners {
		if runner == gh {
			return true
		}
	}
	return false
}

// analyzeRequiredWorkflows analyzes workflow requirements from repository rulesets
func analyzeRequiredWorkflows(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Get list of rulesets
	var rulesets []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Source      string `json:"source"`
		Enforcement string `json:"enforcement"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/rulesets", owner, repo), &rulesets)
	if err != nil {
		return err // Repository rulesets not accessible
	}

	for _, ruleset := range rulesets {
		// Get detailed ruleset to extract workflow requirements
		var detailedRuleset struct {
			Rules []struct {
				Type       string `json:"type"`
				Parameters struct {
					Workflows []struct {
						Path         string `json:"path"`
						Ref          string `json:"ref"`
						RepositoryID int64  `json:"repository_id"`
					} `json:"workflows"`
				} `json:"parameters"`
			} `json:"rules"`
		}

		if err := client.Get(fmt.Sprintf("repos/%s/%s/rulesets/%d", owner, repo, ruleset.ID), &detailedRuleset); err != nil {
			continue
		}

		// Extract workflow requirements
		for _, rule := range detailedRuleset.Rules {
			if rule.Type == "workflows" {
				for _, workflow := range rule.Parameters.Workflows {
					workflowFile := path.Base(workflow.Path)
					workflowInfo := fmt.Sprintf("%s (ID: %d, repo: %s/%s, ruleset: %s)", 
						workflowFile, workflow.RepositoryID, ruleset.Source, repo, ruleset.Name)
					deps.ActionsCIDependencies.RequiredWorkflows = append(deps.ActionsCIDependencies.RequiredWorkflows, workflowInfo)
				}
			}
		}
	}

	return nil
}