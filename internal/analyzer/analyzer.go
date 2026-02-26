package analyzer

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/jefeish/gh-repo-transfer/internal/dependencies"
	"github.com/jefeish/gh-repo-transfer/internal/types"
)

// AnalyzeOrganizationalDependencies performs comprehensive analysis across all 6 categories
func AnalyzeOrganizationalDependencies(client api.RESTClient, owner, repo string, verbose bool) (*types.OrganizationalDependencies, error) {
	if verbose {
		fmt.Fprintf(os.Stderr, "Starting organizational dependencies analysis for %s/%s\n", owner, repo)
	}

	deps := &types.OrganizationalDependencies{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
	}

	// 1. Organization-Specific Code Dependencies
	if verbose {
		fmt.Fprintf(os.Stderr, "Analyzing code dependencies...\n")
	}
	if err := dependencies.AnalyzeCodeDependencies(client, owner, repo, deps); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze code dependencies: %v\n", err)
		}
	}

	// 2. GitHub Actions & CI/CD Dependencies
	if verbose {
		fmt.Fprintf(os.Stderr, "Analyzing CI/CD dependencies...\n")
	}
	if err := dependencies.AnalyzeActionsCIDependencies(client, owner, repo, deps); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze Actions/CI dependencies: %v\n", err)
		}
	}

	// 3. Access Control & Permissions
	if verbose {
		fmt.Fprintf(os.Stderr, "Analyzing access control dependencies...\n")
	}
	if err := dependencies.AnalyzeAccessPermissions(client, owner, repo, deps); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze access control dependencies: %v\n", err)
		}
	}

	// 4. Security & Compliance Dependencies
	if verbose {
		fmt.Fprintf(os.Stderr, "Analyzing security compliance dependencies...\n")
	}
	if err := dependencies.AnalyzeSecurityCompliance(client, owner, repo, deps); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze security compliance dependencies: %v\n", err)
		}
	}

	// 5. GitHub Apps & Integrations
	if verbose {
		fmt.Fprintf(os.Stderr, "Analyzing apps and integrations dependencies...\n")
	}
	if err := dependencies.AnalyzeAppsIntegrations(client, owner, repo, deps); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze apps and integrations dependencies: %v\n", err)
		}
	}

	// 6. Organizational Governance
	if verbose {
		fmt.Fprintf(os.Stderr, "Analyzing governance dependencies...\n")
	}
	if err := dependencies.AnalyzeOrgGovernance(client, owner, repo, deps); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to analyze governance dependencies: %v\n", err)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Organizational dependencies analysis completed\n")
	}

	return deps, nil
}