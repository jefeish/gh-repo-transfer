# gh-repo-transfer

A GitHub client utility for transferring or archiving repos, with special focus on ORG-level dependency 

## Overview

This extension helps you understand the governance and configuration of GitHub repositories by inspecting various settings including:

- **Branch Protection Rules** - Required status checks, review requirements, admin enforcement
- **Collaborators & Teams** - Repository access levels and permissions
- **Security Settings** - Vulnerability alerts, automated fixes, secret scanning
- **Repository Settings** - Merge options, branch policies, feature toggles
- **Issue Management** - Labels, milestones, and project configuration

## Installation

```bash
gh extension install jefeish/gh-repo-transfer
```

## Usage

### Basic Usage

```bash
# Inspect current repository (if in a git repository)
gh repo-transfer

# Inspect a specific repository
gh repo-transfer owner/repo-name
```

### Output Formats

```bash
# JSON output (default)
gh repo-transfer owner/repo --format json

# YAML output
gh repo-transfer owner/repo --format yaml

# Human-readable table format
gh repo-transfer owner/repo --format table
```

### Filtering Sections

```bash
# Only inspect branch protection
gh repo-transfer owner/repo --sections branches

# Multiple sections
gh repo-transfer owner/repo --sections branches,security,collaborators

# Available sections: branches, collaborators, teams, security, settings, labels, milestones
```

### Verbose Output

```bash
# Enable verbose logging
gh repo-transfer owner/repo --verbose
```

## Examples

### Inspect Branch Protection

```bash
gh repo-transfer microsoft/vscode --sections branches --format table
```

### Export Repository Governance

```bash
# Export to JSON file
gh repo-transfer owner/repo --format json > repo-governance.json

# Export to YAML file
gh repo-transfer owner/repo --format yaml > repo-governance.yaml
```

### Audit Multiple Repositories

```bash
# Create a script to audit multiple repositories
#!/bin/bash
for repo in "org/repo1" "org/repo2" "org/repo3"; do
  echo "Auditing $repo..."
  gh repo-transfer $repo --format json > "$(echo $repo | tr '/' '_')-audit.json"
done
```

## Sample Output

### JSON Format

```json
{
  "repository": {
    "owner": "microsoft",
    "name": "vscode"
  },
  "branch_protection": [
    {
      "pattern": "main",
      "enforce_admins": true,
      "required_status_checks": ["CI", "Hygiene"],
      "required_pull_request_reviews": true,
      "required_approving_review_count": 1
    }
  ],
  "security_settings": {
    "vulnerability_alerts": true,
    "automated_security_fixes": true,
    "secret_scanning": true
  }
}
```

### Table Format

```
Repository Governance Report
═══════════════════════════

📁 Repository: microsoft/vscode

⚙️  Repository Settings
├─ Private: ❌ No
├─ Default Branch: main
├─ Issues: ✅ Yes
└─ Allow Squash Merge: ✅ Yes

🔒 Security Settings
├─ Vulnerability Alerts: ✅ Yes
├─ Automated Security Fixes: ✅ Yes
└─ Secret Scanning: ✅ Yes

🛡️  Branch Protection Rules
└─ Branch: main
   ├─ Enforce Admins: ✅ Yes
   ├─ Require PR Reviews: ✅ Yes
   └─ Required Approving Reviews: 1
```

## Development

### Prerequisites

- Go 1.21 or later
- GitHub CLI (`gh`) installed and authenticated

### Building

```bash
# Build the extension
make build

# Install for development
make install-dev

# Run tests
make test

# Format and lint code
make check
```

### Project Structure

```
.
├── main.go          # Main CLI logic and command definitions
├── api.go           # GitHub API interaction functions
├── output.go        # Output formatting (JSON, YAML, table)
├── go.mod           # Go module dependencies
├── Makefile         # Build and development tasks
└── README.md        # Documentation
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Permissions Required

This extension requires the following GitHub permissions to function:

- **Repository access** - Read repository settings and metadata
- **Collaborator access** - Read collaborator and team information
- **Security settings** - Read security and vulnerability settings (may require additional permissions for private repositories)

## Troubleshooting

### Authentication Issues

```bash
# Ensure you're authenticated with GitHub CLI
gh auth status

# If not authenticated, login
gh auth login
```

### Permission Errors

Some repository settings may require additional permissions. If you encounter permission errors:

1. Ensure you have the necessary access to the repository
2. For organization repositories, you may need organization member permissions
3. Some security settings require admin access to view

### Verbose Mode

Use `--verbose` flag to see detailed information about what the tool is doing:

```bash
gh repo-transfer owner/repo --verbose
```

This will show warnings for any settings that couldn't be retrieved due to permission limitations.


