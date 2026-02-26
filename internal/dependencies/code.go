// Package dependencies contains analyzers for the 6 categories of organizational dependencies
package dependencies

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// AnalyzeCodeDependencies analyzes organization-specific code dependencies
func AnalyzeCodeDependencies(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	// Analyze Git submodules
	if err := analyzeGitSubmodules(client, owner, repo, deps); err != nil {
		// Non-fatal error - .gitmodules might not exist
	}

	// Analyze package files for organization registries
	if err := analyzePackageFiles(client, owner, repo, deps); err != nil {
		// Non-fatal error - package files might not exist
	}

	// Analyze Dockerfiles for organization container registries
	if err := analyzeDockerfiles(client, owner, repo, deps); err != nil {
		// Non-fatal error - Dockerfiles might not exist
	}

	return nil
}

// analyzeGitSubmodules checks for submodules pointing to the same organization
func analyzeGitSubmodules(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	var content struct {
		Content string `json:"content"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/contents/.gitmodules", owner, repo), &content)
	if err != nil {
		return err // .gitmodules doesn't exist
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return err
	}

	gitmodulesContent := string(decoded)
	lines := strings.Split(gitmodulesContent, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "url") {
			// Extract URL - all submodules are dependencies for migration purposes
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				url := strings.TrimSpace(parts[1])
				
				if isOrganizationalRepo(url, owner) {
					submoduleInfo := fmt.Sprintf("%s (same organization)", url)
					deps.CodeDependencies.GitSubmodules = append(deps.CodeDependencies.GitSubmodules, submoduleInfo)
				} else {
					submoduleInfo := fmt.Sprintf("%s (external dependency)", url)
					deps.CodeDependencies.GitSubmodules = append(deps.CodeDependencies.GitSubmodules, submoduleInfo)
				}
			}
		}
	}

	return nil
}

// analyzePackageFiles analyzes package files for organization-specific registries
func analyzePackageFiles(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	packageFiles := []string{
		"package.json",     // npm
		"pom.xml",         // Maven
		"build.gradle",    // Gradle
		"requirements.txt", // Python pip
		"Pipfile",         // Python pipenv
		"go.mod",          // Go modules
		".npmrc",          // npm config
	}

	for _, file := range packageFiles {
		if err := analyzePackageFile(client, owner, repo, file, deps); err != nil {
			// Non-fatal - file might not exist
			continue
		}
	}

	return nil
}

func analyzePackageFile(client api.RESTClient, owner, repo, filename string, deps *types.OrganizationalDependencies) error {
	var content struct {
		Content string `json:"content"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, filename), &content)
	if err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return err
	}

	fileContent := string(decoded)
	
	// Look for organization-specific registry patterns
	registryPatterns := []string{
		fmt.Sprintf(`registry.*%s`, owner),
		fmt.Sprintf(`%s\..*\.com`, owner),
		`@.*:registry=`,
		`npm\.pkg\.github\.com`,
	}

	for _, pattern := range registryPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(fileContent) {
			deps.CodeDependencies.OrgPackageRegistries = append(deps.CodeDependencies.OrgPackageRegistries, filename)
			break
		}
	}

	return nil
}

// analyzeDockerfiles checks for organization-specific container registries
func analyzeDockerfiles(client api.RESTClient, owner, repo string, deps *types.OrganizationalDependencies) error {
	dockerFiles := []string{
		"Dockerfile",
		"docker-compose.yml",
		"docker-compose.yaml",
	}

	for _, file := range dockerFiles {
		if err := analyzeDockerfile(client, owner, repo, file, deps); err != nil {
			// Non-fatal - file might not exist
			continue
		}
	}

	return nil
}

func analyzeDockerfile(client api.RESTClient, owner, repo, filename string, deps *types.OrganizationalDependencies) error {
	var content struct {
		Content string `json:"content"`
	}

	err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, filename), &content)
	if err != nil {
		return err
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Content)
	if err != nil {
		return err
	}

	fileContent := string(decoded)
	
	// Look for organization-specific container registries
	registryPatterns := []string{
		fmt.Sprintf(`%s\.azurecr\.io`, owner),
		fmt.Sprintf(`ghcr\.io/%s`, owner),
		fmt.Sprintf(`%s\..*\.amazonaws\.com`, owner),
		fmt.Sprintf(`gcr\.io/%s`, owner),
	}

	for _, pattern := range registryPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(fileContent) {
			deps.CodeDependencies.OrgSpecificContainerRegistries = append(deps.CodeDependencies.OrgSpecificContainerRegistries, 
				fmt.Sprintf("%s (in %s)", pattern, filename))
			break
		}
	}

	return nil
}

// isOrganizationalRepo checks if a repository URL belongs to the same organization
func isOrganizationalRepo(url, owner string) bool {
	// Handle GitHub URLs
	if strings.Contains(url, "github.com") {
		parts := strings.Split(url, "/")
		for i, part := range parts {
			if part == "github.com" && i+1 < len(parts) {
				repoOwner := parts[i+1]
				return repoOwner == owner
			}
		}
	}
	return false
}