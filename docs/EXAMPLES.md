# Usage Examples for gh repo-transfer

This document provides practical examples of using the `gh repo-transfer` extension.

## Installation

First, install the extension (once published):

```bash
gh extension install jefeish/gh-repo-transfer
```

For development/testing, you can install locally:

```bash
# In the project directory
make install-dev
```

## Basic Examples

### Inspect Current Repository

```bash
# If you're in a git repository directory
gh repo-transfer

# Explicitly specify a repository
gh repo-transfer microsoft/vscode
```

### Different Output Formats

```bash
# JSON output (default)
gh repo-transfer microsoft/vscode --format json

# YAML output
gh repo-transfer microsoft/vscode --format yaml

# Human-readable table
gh repo-transfer microsoft/vscode --format table
```

### Filter Specific Sections

```bash
# Only branch protection rules
gh repo-transfer microsoft/vscode --sections branches

# Multiple sections
gh repo-transfer microsoft/vscode --sections branches,security,collaborators

# All available sections: branches, collaborators, teams, security, settings, labels, milestones
```

## Advanced Usage

### Audit Script for Multiple Repositories

Create a script to audit multiple repositories:

```bash
#!/bin/bash
# audit-repos.sh

repos=(
    "microsoft/vscode"
    "golang/go"
    "kubernetes/kubernetes"
)

for repo in "${repos[@]}"; do
    echo "Auditing $repo..."
    gh repo-transfer "$repo" --format json > "${repo//\//_}-audit.json"
    echo "Results saved to ${repo//\//_}-audit.json"
done
```

### Generate Compliance Reports

```bash
# Generate comprehensive security report
gh repo-transfer myorg/myrepo --sections security,branches --format table

# Export detailed governance configuration
gh repo-transfer myorg/myrepo --format yaml > governance-config.yaml
```

### Verbose Logging

```bash
# Enable verbose output to see detailed information and warnings
gh repo-transfer myorg/myrepo --verbose
```

## Example Outputs

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
      "required_approving_review_count": 1,
      "dismiss_stale_reviews": true,
      "require_code_owner_reviews": true
    }
  ],
  "collaborators": [
    {
      "login": "maintainer1",
      "permission": "admin",
      "type": "User"
    }
  ],
  "security_settings": {
    "vulnerability_alerts": true,
    "automated_security_fixes": true,
    "secret_scanning": true,
    "secret_scanning_push_protection": true
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
├─ Archived: ❌ No
├─ Default Branch: main
├─ Allow Merge Commit: ✅ Yes
└─ Delete Branch on Merge: ✅ Yes

🔒 Security Settings
├─ Vulnerability Alerts: ✅ Yes
├─ Automated Security Fixes: ✅ Yes
└─ Secret Scanning: ✅ Yes

🛡️  Branch Protection Rules
└─ Branch: main
   ├─ Enforce Admins: ✅ Yes
   ├─ Require PR Reviews: ✅ Yes
   │  └─ Required Approving Reviews: 1
   └─ Required Status Checks:
      • CI
      • Hygiene
```

## Use Cases

### 1. Security Audit

Check security settings across repositories:

```bash
gh repo-transfer myorg/repo1 --sections security --format table
gh repo-transfer myorg/repo2 --sections security --format table
```

### 2. Branch Protection Review

Audit branch protection policies:

```bash
gh repo-transfer myorg/critical-repo --sections branches --verbose
```

### 3. Access Review

Review collaborator and team access:

```bash
gh repo-transfer myorg/private-repo --sections collaborators,teams --format json
```

### 4. Compliance Export

Generate compliance documentation:

```bash
# Export all governance settings
gh repo-transfer myorg/repo --format yaml > compliance-report.yaml

# Create summary for multiple repos
for repo in repo1 repo2 repo3; do
    gh repo-transfer "myorg/$repo" --format json > "${repo}-compliance.json"
done
```

## Troubleshooting

### Permission Issues

If you encounter permission errors:

```bash
# Check your GitHub CLI authentication
gh auth status

# Re-authenticate if needed
gh auth login --scopes repo
```

### Missing Information

Some information requires specific permissions:

- **Branch protection**: Requires repository read access
- **Collaborators**: Requires repository admin access
- **Teams**: Requires organization member permissions
- **Security settings**: May require repository admin access

Use `--verbose` flag to see detailed information about what could not be retrieved:

```bash
gh repo-transfer myorg/repo --verbose --sections security
```