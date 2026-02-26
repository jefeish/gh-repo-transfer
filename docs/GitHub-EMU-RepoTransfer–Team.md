# GitHub Enterprise Repo Transfer – Team Migration Toolkit

This document shows **all supported approaches** to preserve team access when transferring repositories between organizations.

---

## 1. End-to-End Migration Flow (Recommended)

```text
1. Export team permissions from source org
2. Transfer repository
3. Map source teams → destination teams
4. Reapply permissions
5. Fix branch protection & CODEOWNERS
6. Validate access
```

---

## 2. REST API Script (Export & Reapply Teams)

### 2.1 Export Teams From Source Repo

```bash
export GH_TOKEN=ghp_xxx
export SRC_ORG=source-org
export REPO=my-repo

curl -s \
  -H "Authorization: Bearer $GH_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/$SRC_ORG/$REPO/teams \
  | jq '[.[] | {team: .slug, permission: .permission}]' \
  > teams.json
```

Example output:

```json
[
  {"team": "backend", "permission": "write"},
  {"team": "devops", "permission": "admin"}
]
```

---

### 2.2 Transfer the Repository

```bash
export DST_ORG=destination-org

curl -X POST \
  -H "Authorization: Bearer $GH_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/$SRC_ORG/$REPO/transfer \
  -d '{"new_owner":"'$DST_ORG'"}'
```

⚠️ This removes all team access.

---

### 2.3 Reapply Teams in Destination Org

```bash
export DST_ORG=destination-org

jq -c '.[]' teams.json | while read row; do
  TEAM=$(echo $row | jq -r .team)
  PERM=$(echo $row | jq -r .permission)

  curl -X PUT \
    -H "Authorization: Bearer $GH_TOKEN" \
    -H "Accept: application/vnd.github+json" \
    https://api.github.com/orgs/$DST_ORG/teams/$TEAM/repos/$DST_ORG/$REPO \
    -d '{"permission":"'$PERM'"}'
done
```

Notes:

* Team **slug must exist** in destination org
* Permissions are reapplied exactly

---

## 3. GraphQL (Faster for Bulk Repos)

### Query Team Access

```graphql
query($org:String!, $repo:String!) {
  repository(owner:$org, name:$repo) {
    teams(first:100) {
      nodes {
        slug
        permission
      }
    }
  }
}
```

GraphQL is preferred for:

* Bulk migrations
* Rate-limit sensitive operations

---

## 4. Terraform (Best for Enterprise & Compliance)

### 4.1 Define Teams

```hcl
resource "github_team" "backend" {
  name = "backend"
  privacy = "closed"
}
```

### 4.2 Assign Repo Permissions

```hcl
resource "github_team_repository" "backend_access" {
  team_id   = github_team.backend.id
  repository = "my-repo"
  permission = "push"
}
```

### 4.3 Migration Strategy

```text
terraform import github_team.backend backend
terraform apply
```

Benefits:

* Drift detection
* Auditable access
* Repeatable migrations

---

## 5. Branch Protection Recovery

After transfer:

* Team reviewers are removed
* Rules may be disabled

### Reapply Example

```bash
curl -X PUT \
  -H "Authorization: Bearer $GH_TOKEN" \
  https://api.github.com/repos/$DST_ORG/$REPO/branches/main/protection \
  -d '{
    "required_pull_request_reviews": {
      "required_approving_review_count": 1
    }
  }'
```

---

## 6. CODEOWNERS Fix

### Before

```text
/src @source-org/backend
```

### After

```text
/src @destination-org/backend
```

Required after every org-to-org transfer.

---

## 7. Validation Checklist

```text
✔ Repo visible in destination org
✔ Correct teams assigned
✔ Permissions verified
✔ Branch protection enabled
✔ CODEOWNERS resolving
✔ CI still running
```

---

## 8. Common Failure Modes

| Issue           | Cause             | Fix               |
| --------------- | ----------------- | ----------------- |
| 404 adding team | Team slug missing | Create team first |
| No reviewers    | CODEOWNERS stale  | Update org name   |
| Access drift    | Manual edits      | Terraform         |

---

## 9. Recommendation Summary

| Scale           | Tool      |
| --------------- | --------- |
| 1–5 repos       | REST API  |
| 10–100 repos    | GraphQL   |
| Enterprise-wide | Terraform |

---

If needed, this toolkit can be adapted for:

* Azure AD / Okta team sync
* Multi-org Enterprise migrations
* Zero-downtime access transfers
