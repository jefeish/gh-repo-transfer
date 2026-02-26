# Enhanced GitHub Repository Organizational Dependencies Analyzer

## 🎉 **Implementation Complete!**

The tool has been successfully enhanced with **real GitHub API calls** and **comprehensive file content analysis**. Here's what's been implemented:

## ✨ **New Features Implemented**

### 1. **Real GitHub API Integration**
- ✅ **Repository Information API** - Fetches actual repo settings
- ✅ **Contents API** - Analyzes real file contents  
- ✅ **Teams & Collaborators API** - Gets real access control data
- ✅ **Branch Protection API** - Analyzes actual protection rules
- ✅ **Webhooks & Deploy Keys API** - Detects real integrations
- ✅ **Environments API** - Lists deployment environments

### 2. **Comprehensive File Content Analysis**

#### **Code Dependencies Analysis**
- **📄 .gitmodules** - Detects org-specific submodules
- **📦 package.json, pom.xml, go.mod** - Finds org package registries  
- **🐳 Dockerfile, docker-compose.yml** - Identifies org container registries
- **🔍 Source files** - Searches for hardcoded org references

#### **Actions/CI Dependencies Analysis** 
- **⚙️ .github/workflows/*.yml** - Parses workflow files for:
  - Organization secrets and variables (`secrets.ORG_SECRET`)
  - Self-hosted runners (non-GitHub hosted)
  - Organization-specific actions (`orgname/action-name`)
  - Cross-repo triggers and dependencies

#### **Access Control Analysis**
- **👥 Teams API** - Lists teams with repository access
- **👤 Collaborators API** - Gets individual collaborators  
- **📋 CODEOWNERS** - Parses code review requirements
- **🛡️ Branch Protection** - Analyzes protection rules

#### **Security & Compliance Analysis**
- **🔐 Security Settings API** - Repository security configuration
- **🤖 dependabot.yml** - Dependabot configuration detection
- **📜 SECURITY.md** - Security policy presence

#### **Apps & Integrations Analysis**
- **🔗 Webhooks API** - External integrations
- **🔑 Deploy Keys API** - SSH deployment keys
- **📱 GitHub Apps** - Installed applications (org-level permission required)

#### **Governance Analysis**
- **📏 Repository Settings** - Merge restrictions and policies
- **📝 Issue Templates** - `.github/ISSUE_TEMPLATE/`
- **🔄 PR Templates** - `.github/pull_request_template.md`
- **✅ Required Status Checks** - CI/CD requirements

## 🚀 **Enhanced Capabilities**

### **Real Content Parsing**
```bash
# The tool now actually reads and parses files like:
- package.json → Detects npm.pkg.github.com registries
- pom.xml → Finds maven.pkg.github.com repositories  
- .github/workflows/*.yml → Extracts secrets, runners, actions
- CODEOWNERS → Parses team and user requirements
```

### **Comprehensive API Integration**
```bash
# Makes real GitHub API calls to:
GET /repos/{owner}/{repo}                    # Basic repo info
GET /repos/{owner}/{repo}/contents/{path}     # File contents
GET /repos/{owner}/{repo}/teams              # Team access  
GET /repos/{owner}/{repo}/collaborators      # Individual access
GET /repos/{owner}/{repo}/branches           # Branch protection
GET /repos/{owner}/{repo}/hooks              # Webhooks
GET /repos/{owner}/{repo}/keys               # Deploy keys
GET /repos/{owner}/{repo}/environments       # Deployment environments
```

## 📊 **Usage Examples**

### **Basic Analysis**
```bash
./repo-deps.sh owner/repository-name
```

### **Detailed Analysis with Verbose Output**
```bash
./repo-deps.sh owner/repository-name --verbose --format summary
```

### **JSON Output for Automation**
```bash
./repo-deps.sh owner/repository-name --format json > dependencies.json
```

### **Multi-Repository Organizational Analysis**
```bash
./repo-deps.sh --org myorg repo1 repo2 repo3 --format summary
```

## 🎯 **Real Detection Capabilities**

The enhanced tool now actually detects:

- **🔍 Hardcoded References**: `github.com/orgname/repo` in README files
- **📦 Private Registries**: `npm.pkg.github.com/@orgname` in package.json
- **🐳 Container Registries**: `ghcr.io/orgname/image` in Dockerfiles  
- **⚙️ Workflow Dependencies**: `secrets.ORG_SECRET` in GitHub Actions
- **👥 Team Access**: Actual teams with repository permissions
- **🔐 Security Policies**: Real Dependabot and security configurations

## 🛠️ **Architecture**

```
simple-repo-deps.go        # Main CLI and analysis orchestration
repo-analysis.go           # Core API calls and file parsing
additional-analysis.go     # Security, apps, and governance analysis
repo-deps.sh              # Convenient shell wrapper
```

## ⚠️ **Authentication Notes**

- **Public repositories**: Basic analysis works without authentication
- **Private repositories**: Requires GitHub CLI authentication (`gh auth login`)  
- **Organization data**: Some endpoints require organization member permissions
- **Advanced features**: Admin permissions needed for some security settings

The tool gracefully handles authentication errors and continues with available data.

## 🎉 **Ready for Production Use**

The enhanced analyzer is now a comprehensive tool for:
- **📋 Pre-migration planning** - Know exactly what will break
- **🔍 Organizational dependency auditing** - Complete visibility  
- **📊 Migration impact assessment** - Quantify complexity
- **🤖 CI/CD integration** - Automated dependency checking

Perfect for organizations planning repository migrations or conducting governance audits! 🚀