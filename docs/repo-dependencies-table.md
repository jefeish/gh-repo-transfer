# GitHub Platform/Governance Dependencies - Reference Table

These are the dependencies that tie the repository to its current organizational context

> **Note**: Dependencies are categorized as either:
> - **Direct**: Configuration, access, or integrations specifically set up for this individual repository 
> - **Indirect**: Organization-wide governance/policies that apply to all repositories (security policies, org rulesets, installed apps)

## 1. Organization-Specific Code Dependencies

|Type|Description|GitHub Location|
|---|---|---|
|**Internal Repository References**|Dependencies on other repositories within the same organization|Repository code/config files|
|**Git Submodules**|Submodules pointing to repos in the same organization|`.gitmodules` file|
|**Organization Package Registries**|Dependencies on org-specific npm/Maven/NuGet registries|`package.json`, `pom.xml`, `.npmrc`, etc.|
|**Hard-coded Organization References**|Code containing organization name, URLs, or org-specific endpoints|Source code files|
|**Organization-specific Container Registries**|Docker images from org's container registry|`Dockerfile`, `docker-compose.yml`|

## 2. GitHub Actions & CI/CD Dependencies

|Type|Description|GitHub Location|
|---|---|---|
|**Organization Secrets/Variables**|References to org-level secrets and variables in workflows|Org Settings → Secrets and variables|
|**Self-hosted Runners**|Organization or repository-level runners referenced in workflows|Org Settings → Actions → Runners|
|**Environment Dependencies**|Environment reviewers that are org members/teams, or environment secrets referencing org-level secrets|Repository → Settings → Environments|
|**Organization-specific Actions**|Custom actions hosted in the same organization|`.github/workflows/*.yml` files|
|**Cross-repo Workflow Triggers**|Workflows that trigger or depend on other repos in the org|`.github/workflows/*.yml` files|

## 3. Access Control & Permissions

|Type|Description|GitHub Location|
|---|---|---|
|**Teams**|All teams with access to the repository|Repository → Settings → Manage access|
|**Individual Collaborators**|Direct repository access|Repository → Settings → Manage access|
|**Organization Roles**|Users/teams with custom organization roles that grant repository access|Organization Settings → Organization roles|
|**Organization Membership**|Required organization membership|Organization → People|
|**CODEOWNERS**|Code review requirements|`.github/CODEOWNERS` file|

## 4. Security & Compliance Dependencies

|Type|Description|GitHub Location|
|---|---|---|
|**Security Campaigns** *(Indirect)*|Organization-level security initiatives and enforcement campaigns that apply to all repos|Org Settings → Security & analysis → Campaigns|

## 5. GitHub Apps & Integrations

|Type|Description|GitHub Location|
|---|---|---|
|**Installed GitHub Apps** *(Indirect)*|Organization-wide apps that may have access to all repositories|Org Settings → GitHub Apps|
|**Personal Access Tokens**|PATs with repository access - token holders must be in new org or regenerate tokens|User Settings → Developer settings → Personal access tokens|

## 6. Organizational Governance

|Type|Description|GitHub Location|
|---|---|---|
|**Policies** *(Indirect)*|Organization-wide policies that apply to all repositories|Organization → Settings → Policies|
|**Repository Rulesets** *(Indirect)*|Organization-level rules that apply to multiple repositories|Organization → Settings → Repository → Rules|
|**Issue Templates**|Standardized issue reporting|`.github/ISSUE_TEMPLATE/`|
|**Pull Request Templates**|Standardized PR process|`.github/pull_request_template.md`|