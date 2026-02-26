# Repository Transfer with Team Assignment

## Overview

The `--assign` option for the `gh repo-transfer transfer` command automatically recreates team access permissions from the source repository to the target organization after a repository transfer. This ensures that team-based access control is preserved during repository migrations between organizations.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Usage](#usage)
- [Implementation Details](#implementation-details)
- [API Endpoints](#api-endpoints)
- [Error Handling](#error-handling)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Features

### Core Functionality
- **Automatic Team Discovery**: Scans the source repository for existing team assignments
- **Team Creation**: Creates teams in the target organization if they don't already exist
- **Permission Preservation**: Maintains the same permission levels (read, write, admin, etc.)
- **Bulk Processing**: Supports multiple repository transfers with team assignment
- **Non-destructive**: Will not overwrite existing teams in the target organization

### Permission Mapping
The tool automatically maps GitHub permissions from source to target:

| Source Permission | Target Permission | Description |
|------------------|------------------|-------------|
| `read` | `pull` | Read-only access |
| `write` | `push` | Read and write access |
| `admin` | `admin` | Full administrative access |
| `maintain` | `maintain` | Maintain access (GitHub Enterprise) |
| `triage` | `triage` | Triage access (GitHub Enterprise) |
| Custom roles | Preserved as-is | Organization-specific custom roles |

## Prerequisites

### Required Permissions
- **Source Repository**: Admin permissions on the repository being transferred
- **Target Organization**: 
  - Owner or admin permissions in the target organization
  - Permission to create teams
  - Permission to manage repository access

### GitHub CLI Setup
Ensure you're authenticated with the GitHub CLI:
```bash
gh auth login
```

## Usage

### Basic Syntax
```bash
gh repo-transfer transfer [owner/repo] --target-org [target-org] --assign
```

### Command Options
- `--assign, -a`: Enable team assignment from source repository
- `--target-org, -t`: Target organization for the transfer (required)
- `--dry-run, -d`: Preview actions without executing
- `--verbose, -v`: Enable verbose output
- `--enforce, -e`: Skip dependency validation checks

### Single Repository Transfer
```bash
# Transfer repository and assign teams
gh repo-transfer transfer myorg/myproject --target-org neworg --assign

# With verbose output to see detailed progress
gh repo-transfer transfer myorg/myproject --target-org neworg --assign --verbose
```

### Multiple Repository Transfer
```bash
# Transfer multiple repositories with team assignment
gh repo-transfer transfer myorg/project1 myorg/project2 myorg/project3 \
  --target-org neworg --assign --verbose
```

### Dry Run Mode
```bash
# Preview what would happen without executing
gh repo-transfer transfer myorg/myproject --target-org neworg --assign --dry-run
```

## Implementation Details

### Workflow Process

1. **Repository Validation**: Validates source repository exists and user has admin permissions
2. **Target Organization Validation**: Confirms target organization exists and user has appropriate permissions
3. **Dependency Analysis**: Analyzes potential transfer blockers (unless `--enforce` is used)
4. **Repository Transfer**: Executes the repository transfer
5. **Team Discovery**: Retrieves team assignments from source repository
6. **Team Creation**: Creates teams in target organization (if they don't exist)
7. **Team Assignment**: Assigns teams to the transferred repository

### Team Name Handling
- Team names are converted to slugs for API compatibility
- Spaces are replaced with hyphens
- Names are converted to lowercase
- Example: "Frontend Team" becomes "frontend-team"

### Conflict Resolution
- If a team already exists in the target organization, it will be reused
- No existing teams are modified or overwritten
- Team descriptions indicate they were "migrated from source repository"

## API Endpoints

The `--assign` feature utilizes the following GitHub REST API endpoints:

### Team Discovery
```http
GET /repos/{owner}/{repo}/teams
```
Retrieves all teams with access to the source repository.

### Team Existence Check
```http
GET /orgs/{target-org}/teams/{team-slug}
```
Checks if a team already exists in the target organization.

### Team Creation
```http
POST /orgs/{target-org}/teams
```
```json
{
  "name": "Team Name",
  "description": "Team migrated from source repository",
  "privacy": "closed"
}
```

### Repository Team Assignment
```http
PUT /orgs/{target-org}/teams/{team-slug}/repos/{target-org}/{repo}
```
```json
{
  "permission": "push"
}
```

## Error Handling

### Fatal Errors
These errors will stop the transfer process:
- Source repository not found or insufficient permissions
- Target organization not found or insufficient permissions
- API authentication failures
- Repository transfer failures

### Non-Fatal Warnings
These errors will be logged but won't stop the process:
- Individual team creation failures
- Individual team assignment failures
- Teams with unsupported permission types

### Recovery Strategies
- **Partial Failures**: The tool continues processing remaining teams even if some fail
- **Retry Logic**: Manual retry of failed team assignments is possible
- **Verbose Logging**: Use `--verbose` flag to get detailed error information

## Examples

### Example 1: Basic Transfer with Team Assignment
```bash
$ gh repo-transfer transfer acme/web-app --target-org newacme --assign --verbose

Preparing to transfer repository: acme/web-app to newacme
✅ Target organization 'newacme' exists
✅ Source repository 'acme/web-app' is valid and transferable
Checking for transfer blockers...
✅ No transfer blockers found

🎉 Repository transfer simulation completed!
═════════════════════════════════════════
Source: acme/web-app
Target: newacme

📋 Assigning teams to transferred repository...
Retrieving team information from source repository...
Found 3 teams in source repository
✅ Created team 'Frontend Team' in target organization
✅ Successfully assigned team 'Frontend Team' with 'push' permission
✅ Created team 'Backend Team' in target organization
✅ Successfully assigned team 'Backend Team' with 'admin' permission
Team 'DevOps Team' already exists in target organization
✅ Successfully assigned team 'DevOps Team' with 'push' permission
✅ Team assignment completed

✅ All validations passed - transfer would succeed
```

### Example 2: Dry Run with Multiple Repositories
```bash
$ gh repo-transfer transfer acme/frontend acme/backend --target-org newacme --assign --dry-run

🔍 DRY RUN: Batch repository transfer simulation
═════════════════════════════════════════════════

[1/2] Processing acme/frontend
✅ No transfer blockers found

[2/2] Processing acme/backend  
✅ No transfer blockers found

acme/frontend                             ✅ SUCCESS (VALIDATED)
acme/backend                              ✅ SUCCESS (VALIDATED)

Summary:
  Total repositories: 2
  Would succeed: 2
  Would fail: 0
  Target: newacme
```

## Troubleshooting

### Common Issues

#### "Team creation failed: insufficient permissions"
**Cause**: User lacks permission to create teams in the target organization.
**Solution**: Ensure you have owner or admin permissions in the target organization.

#### "Team already exists but assignment failed"
**Cause**: Team exists but user cannot modify repository access.
**Solution**: Verify you have admin permissions on the transferred repository.

#### "No teams found in source repository"
**Cause**: Source repository has no team-based access control.
**Solution**: This is normal - the command completes successfully with no teams to assign.

#### "Repository transfer failed"
**Cause**: Various issues including insufficient permissions or repository constraints.
**Solution**: Use `--verbose` flag to get detailed error information.

### Debug Commands

```bash
# Check current repository teams
gh api repos/owner/repo/teams

# Check target organization teams
gh api orgs/target-org/teams

# Verify authentication and permissions
gh auth status
```

### Verbose Output
Use the `--verbose` flag to get detailed information about:
- Team discovery process
- Team creation attempts
- Permission assignments
- API responses and errors

### Getting Help
```bash
# Display command help
gh repo-transfer transfer --help

# Display overall tool help
gh repo-transfer --help
```

## Best Practices

1. **Always test with `--dry-run`** before executing transfers
2. **Use `--verbose`** for detailed logging in production environments
3. **Backup team configurations** before major organizational changes
4. **Verify permissions** in both source and target organizations
5. **Plan for team naming conflicts** in the target organization
6. **Review team assignments** after transfer completion

## Security Considerations

- The tool inherits GitHub API permissions from your authenticated session
- Teams are created with "closed" privacy by default
- Original team members are not automatically transferred
- Repository access is granted to teams, not individual users
- Custom organization roles are preserved when possible

## Version Information

This feature was implemented as part of the `gh repo-transfer` extension and requires:
- GitHub CLI v2.0+
- Go 1.19+
- Valid GitHub authentication with appropriate permissions

---

*Generated on February 10, 2026*
*GitHub Copilot - Repository Transfer Tool Documentation*