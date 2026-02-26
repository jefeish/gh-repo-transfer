# Command: `deps`

## Overview

The `deps` command analyzes all **organizational dependencies** of one or more repositories — everything that would need to be assessed or addressed when moving a repository to a different GitHub organization. It is the primary discovery and pre-flight tool before using `transfer` or `archive`.

Results can be output as a human-readable table, JSON, or YAML, and optionally written to separate per-repository files for batch runs.

---

## Usage

```sh
gh repo-transfer deps [owner/repo...] [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--target-org` | `-t` | — | Target organization to validate dependencies against |
| `--format` | `-f` | `table` | Output format: `table`, `json`, `yaml` |
| `--per-repo` | `-p` | `false` | Write results to individual JSON files per repository |
| `--verbose` | `-v` | `false` | Enable verbose/debug output |

### Examples

```sh
# Analyze a single repository
gh repo-transfer deps owner/repo

# Analyze multiple repositories (batch mode with caching)
gh repo-transfer deps owner/repo1 owner/repo2 owner/repo3

# Analyze and validate against a target organization
gh repo-transfer deps owner/repo --target-org target-org

# Output as JSON
gh repo-transfer deps owner/repo --format json

# Write each repo's results to its own file
gh repo-transfer deps owner/repo1 owner/repo2 --per-repo
```

---

## What It Analyzes

The command examines **six dependency categories**:

| Category | Description |
|----------|-------------|
| **Code Dependencies** | References to org-internal packages, private registries, org-specific URLs |
| **CI/CD Dependencies** | GitHub Actions workflows referencing internal actions, runners, secrets, or environments |
| **Access & Permissions** | Teams, individual collaborators, deploy keys, outside collaborators |
| **Security & Compliance** | Branch protection rules, required status checks, secret scanning, GHAS settings |
| **Apps & Integrations** | Installed GitHub Apps, webhooks, OAuth integrations |
| **Governance** | Rulesets, CODEOWNERS, required reviewers, merge strategies |

When `--target-org` is provided, each category is also **validated** against the target organization's capabilities and returns one of:

- ✅ `OK` — No action required
- ⚠️ `Warning` — Should be reviewed, not a hard blocker
- ❌ `Blocker` — Must be resolved before transfer

### Batch Optimization

When multiple repositories from the **same organization** are specified, org-level data (teams, apps, rulesets, etc.) is fetched **once and cached**, significantly reducing GitHub API calls.

---

## Process Flow Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant CLI as gh repo-transfer deps
    participant Cache as Org Cache
    participant GH as GitHub API

    User->>CLI: deps owner/repo1 owner/repo2 [--target-org X]

    CLI->>CLI: Validate repo format (owner/repo)
    CLI->>CLI: Group repos by organization

    alt Multiple repos from same org
        CLI->>GH: GET /orgs/{org}/teams
        CLI->>GH: GET /orgs/{org}/apps
        CLI->>GH: GET /orgs/{org}/rulesets
        GH-->>Cache: Store org-level data
        note over Cache: Batch cache: org data fetched once
    end

    loop For each repository
        CLI->>GH: GET /repos/{owner}/{repo}
        GH-->>CLI: Repo metadata

        CLI->>GH: GET /repos/{owner}/{repo}/teams
        GH-->>CLI: Team access list

        CLI->>GH: GET /repos/{owner}/{repo}/collaborators
        GH-->>CLI: Collaborator list

        CLI->>GH: GET /repos/{owner}/{repo}/branches (protection)
        GH-->>CLI: Branch protection rules

        CLI->>GH: GET /repos/{owner}/{repo}/actions/workflows
        GH-->>CLI: Workflow files

        CLI->>GH: GET /repos/{owner}/{repo}/hooks
        GH-->>CLI: Webhooks

        CLI->>GH: GET /repos/{owner}/{repo}/installations
        GH-->>CLI: Installed apps

        CLI->>GH: GET /repos/{owner}/{repo}/rulesets
        GH-->>CLI: Repo-level rulesets

        CLI->>CLI: Analyze all six dependency categories
    end

    alt --target-org specified
        CLI->>GH: GET /orgs/{target-org}
        CLI->>GH: GET /orgs/{target-org}/teams
        CLI->>GH: GET /orgs/{target-org}/apps
        CLI->>GH: GET /orgs/{target-org}/actions/permissions
        GH-->>CLI: Target org capabilities

        CLI->>CLI: ValidateAgainstTarget()
        note over CLI: Classify each finding as OK / Warning / Blocker
    end

    alt --per-repo flag set
        CLI->>CLI: Write {owner}-{repo}.json for each repo
    else single repo
        CLI->>User: Output table / JSON / YAML
    else multiple repos
        CLI->>User: Combined output table / JSON / YAML
    end
```

---

## Output Structure

When using `--format json`, the output follows this structure:

```json
{
  "repository": "owner/repo",
  "code_dependencies": [...],
  "ci_cd_dependencies": [...],
  "access_permissions": [...],
  "security_compliance": [...],
  "apps_integrations": [...],
  "governance": [...],
  "validation": {
    "summary": {
      "total": 12,
      "ok": 8,
      "warnings": 3,
      "blockers": 1
    },
    ...
  }
}
```

---

## Notes

- The `deps` command is **read-only** — it never modifies any repository or organization.
- It is the recommended first step before running `transfer` or `archive`.
- Blockers identified by `deps --target-org` are the same checks enforced by `transfer` (unless `--enforce` is used).
