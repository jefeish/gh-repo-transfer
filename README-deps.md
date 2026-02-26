# GitHub Repository Organizational Dependencies Analyzer

A comprehensive tool for identifying GitHub platform and governance dependencies that would break when moving a repository between organizations.

## Overview

When migrating repositories between GitHub organizations, many dependencies can break due to organizational boundaries. This tool systematically analyzes repositories to identify all organizational dependencies based on the [dependency reference table](repo-dependencies-table.md).

## Features

### 🔍 **Comprehensive Dependency Analysis**

The tool checks for six categories of organizational dependencies:

1. **Organization-Specific Code Dependencies**
   - Internal repository references
   - Git submodules pointing to org repos
   - Organization package registries
   - Hard-coded organization references
   - Organization-specific container registries

2. **GitHub Actions & CI/CD Dependencies**
   - Organization secrets/variables
   - Self-hosted runners
   - Deployment environments
   - Organization-specific actions
   - Cross-repo workflow triggers

3. **Access Control & Permissions**
   - Team permissions
   - Individual collaborators
   - Organization membership requirements
   - CODEOWNERS references
   - Branch protection rules

4. **Security & Compliance Dependencies**
   - Organization security policies
   - Dependabot configuration
   - Code/secret scanning rules
   - Vulnerability alerts

5. **GitHub Apps & Integrations**
   - Installed GitHub Apps
   - Repository webhooks
   - Deploy keys
   - Personal Access Token dependencies

6. **Organizational Governance**
   - Repository rulesets
   - Required status checks
   - Merge restrictions
   - Issue/PR templates

## Installation

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd gh-repo-transfer
   ```

2. **Build the analyzer:**
   ```bash
   go build -o repo-deps-analyzer repo-deps-analyzer.go analysis-impl.go
   ```

3. **Make the script executable:**
   ```bash
   chmod +x repo-deps.sh
   ```

## Usage

### Single Repository Analysis

```bash
# Basic analysis with summary output
./repo-deps.sh octocat/Hello-World

# JSON output for programmatic processing
./repo-deps.sh octocat/Hello-World --format json

# Verbose output for debugging
./repo-deps.sh octocat/Hello-World --verbose
```

### Multiple Repository Analysis

```bash
# Analyze multiple repositories in an organization
./repo-deps.sh --org myorg repo1 repo2 repo3

# With JSON output for processing
./repo-deps.sh --org myorg repo1 repo2 repo3 --format json
```

### GitHub CLI Extension

You can also install this as a GitHub CLI extension:

```bash
gh extension install <extension-name>
gh repo-deps octocat/Hello-World
```

## Output Formats

### Summary Format (Default)
```
🔍 Organizational Dependencies Analysis
════════════════════════════════════════

📁 Repository: octocat/Hello-World

📊 Dependencies Summary:
├─ 💻 Code Dependencies: 3
├─ 🔄 Actions/CI Dependencies: 5
├─ 🔐 Access Control Dependencies: 8
├─ 🛡️  Security Dependencies: 2
├─ 🔗 Apps/Integrations Dependencies: 1
└─ 📋 Governance Dependencies: 4

🎯 Total Organizational Dependencies: 23

⚠️  Moving this repository to another organization will require addressing these dependencies.
```

### JSON Format
Detailed JSON output with complete dependency information for programmatic processing.

## Reference Documentation

- **[Dependency Types Reference](repo-dependencies-table.md)** - Complete table of all dependency types with GitHub locations
- **[Original Dependency List](repo-dependencies.md)** - Detailed explanation of each dependency category

## Migration Planning

Use this tool to:

1. **Pre-migration Assessment**: Identify all organizational dependencies before starting a migration
2. **Migration Planning**: Create a comprehensive checklist of items to address
3. **Risk Assessment**: Understand the complexity and potential issues of moving a repository
4. **Documentation**: Generate reports for stakeholders about migration requirements

## Examples

### Basic Repository Check
```bash
./repo-deps.sh microsoft/vscode-github-copilot --format summary
```

### Organization-wide Analysis
```bash
./repo-deps.sh --org github cli hub desktop --format json > org-dependencies.json
```

### Integration with CI/CD
```bash
# Check dependencies as part of migration pipeline
./repo-deps.sh $REPO_TO_MIGRATE --format json | jq '.total_dependencies'
```

## API Requirements

This tool requires:
- GitHub CLI (`gh`) installed and authenticated
- Appropriate permissions to read repository settings, actions, and security configurations
- Organization member permissions for full analysis

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new dependency types
4. Update the reference documentation
5. Submit a pull request

## License

[MIT License](LICENSE)

---

**Note**: This tool is read-only and never modifies repository settings. It only analyzes and reports on existing configurations.