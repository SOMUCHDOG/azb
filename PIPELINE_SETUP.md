# GitHub Actions CI/CD Pipeline Setup Guide

This guide walks you through setting up the automated CI/CD pipelines for the Azure Boards CLI project using the GitHub CLI (`gh`).

## Prerequisites

1. **GitHub CLI installed**
   ```bash
   # macOS
   brew install gh

   # Linux
   curl -sS https://webi.sh/gh | sh

   # Windows
   winget install GitHub.cli
   ```

2. **Authenticate with GitHub**
   ```bash
   gh auth login
   ```
   Follow the prompts to authenticate with your GitHub account.

3. **Verify repository access**
   ```bash
   gh repo view
   ```
   This should display information about your current repository.

---

## Step-by-Step Setup

### Step 1: Push the Pipeline Configuration to GitHub

First, push the current branch with all the CI/CD configuration files:

```bash
# Push the pipelines branch to GitHub
git push -u origin pipelines
```

### Step 2: Verify Workflow Files

Check that GitHub has detected the workflow files:

```bash
# List all workflows in the repository
gh workflow list
```

You should see two workflows:
- `CI` (test.yml)
- `Release` (release.yml)

### Step 3: Create a Pull Request to Main

Create a PR to merge the pipelines branch into main:

```bash
# Create a pull request
gh pr create \
  --title "ci: add GitHub Actions pipelines and semantic release automation" \
  --body "$(cat <<'EOF'
## Summary
- Adds CI workflow for linting, testing, and build verification
- Adds release workflow with semantic versioning and GoReleaser
- Configures multi-platform builds (macOS, Linux, Windows)
- Adds version command to CLI
- Includes example test files and updated documentation

## What's Automated
- **Continuous Integration**: Runs on every push and PR
  - Code linting with golangci-lint
  - Test execution with race detection
  - Build verification for all platforms

- **Continuous Deployment**: Runs on push to main
  - Semantic versioning based on commit messages
  - Automatic git tag creation
  - Multi-platform binary builds
  - GitHub Release creation with binaries

## Test Plan
- [x] CI workflow will run automatically on this PR
- [ ] Verify lint job passes
- [ ] Verify test job passes
- [ ] Verify build job succeeds for all platforms
- [ ] After merge, verify release workflow on main

## Breaking Changes
None - this only adds CI/CD infrastructure

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

### Step 4: Monitor the CI Workflow

Watch the CI workflow run on your pull request:

```bash
# View the status of the latest workflow run
gh run list --workflow=test.yml --limit 5

# Watch a specific workflow run (use the run ID from the list)
gh run watch <run-id>

# Or watch the latest run
gh run watch
```

### Step 5: View Workflow Details

If there are any issues, you can view detailed logs:

```bash
# View the latest workflow run
gh run view

# View logs for a specific job
gh run view --log --job=<job-id>

# Download logs for offline viewing
gh run download <run-id>
```

### Step 6: Merge the Pull Request

Once the CI checks pass, merge the PR:

```bash
# Merge using the GitHub CLI
gh pr merge --squash --delete-branch

# Or merge with auto-merge when checks pass
gh pr merge --auto --squash --delete-branch
```

**Note**: When you merge to main, the Release workflow will run, but it won't create a release unless you use a conventional commit message that indicates a version bump (feat:, fix:, etc.).

### Step 7: Trigger Your First Release

To trigger your first automated release, create a commit on main with a conventional commit message:

```bash
# Switch to main and pull the latest
git checkout main
git pull

# Make a small change to trigger a release (e.g., update SPEC.md or add a comment)
# Then commit with a conventional message
git commit -m "feat: initial release with automated CI/CD pipelines"

# Push to main
git push
```

### Step 8: Monitor the Release Workflow

Watch the release workflow create your first automated release:

```bash
# Watch the release workflow
gh run list --workflow=release.yml --limit 5
gh run watch

# Once complete, view the release
gh release list

# View details of the latest release
gh release view --web
```

---

## Verifying the Setup

### Check Workflow Status

```bash
# View all recent workflow runs
gh run list --limit 10

# View status of a specific workflow
gh workflow view ci
gh workflow view release
```

### Check Workflow Logs

If a workflow fails, view the logs to diagnose:

```bash
# View the latest run with logs
gh run view --log

# View a specific job's logs
gh run view <run-id> --log --job=<job-name>
```

### Enable/Disable Workflows

If needed, you can manually control workflows:

```bash
# Disable a workflow
gh workflow disable release.yml

# Enable a workflow
gh workflow enable release.yml
```

### Manual Workflow Triggers

If you want to test workflows manually:

```bash
# Trigger a workflow manually (if configured with workflow_dispatch)
gh workflow run test.yml

# Watch the triggered workflow
gh run watch
```

---

## Setting Up Optional Services

### 1. Code Coverage (Codecov) - Optional

The CI workflow includes codecov integration. To enable it:

```bash
# Create a codecov account at https://codecov.io
# Then add the token as a secret
gh secret set CODECOV_TOKEN
# Paste your Codecov token when prompted
```

### 2. Go Report Card - Optional

The README includes a Go Report Card badge. To activate it:

1. Visit https://goreportcard.com
2. Enter your repository URL: `github.com/casey/azure-boards-cli`
3. Click "Generate Report"

The badge will automatically work once the report is generated.

---

## Troubleshooting

### Workflows Not Running

```bash
# Check if workflows are enabled
gh workflow list

# Check repository settings
gh repo view --web
# Navigate to Settings → Actions → General
# Ensure "Allow all actions and reusable workflows" is selected
```

### Permission Issues

```bash
# Check if the repository has the correct permissions
gh api repos/casey/azure-boards-cli/actions/permissions

# Update permissions if needed (requires admin access)
gh api -X PUT repos/casey/azure-boards-cli/actions/permissions \
  -f enabled=true \
  -f allowed_actions=all
```

### Release Workflow Not Creating Releases

The release workflow only creates releases when:
1. Commits are pushed to the `main` branch
2. Commit messages follow conventional commit format
3. The commit type triggers a version bump (feat, fix, perf, refactor, or BREAKING CHANGE)

```bash
# Check recent commits on main
git log --oneline main

# Verify commit message format
# Good: "feat: add new feature" ✓
# Bad:  "added new feature" ✗
```

### Semantic Release Not Detecting Changes

```bash
# View the release workflow logs
gh run list --workflow=release.yml
gh run view --log

# Look for the semantic-release output
# It will show which commits it analyzed and why it did/didn't create a release
```

### Build Failures

```bash
# View build logs
gh run view --log --job=build

# Common issues:
# 1. Go version mismatch - check go.mod
# 2. Missing dependencies - run `go mod tidy`
# 3. Platform-specific code - check build constraints
```

---

## Repository Settings Recommendations

### Branch Protection Rules

Set up branch protection for main:

```bash
# View in browser to configure
gh repo view --web
# Go to: Settings → Branches → Add rule

# Or use the GitHub API
gh api -X PUT repos/casey/azure-boards-cli/branches/main/protection \
  -f required_status_checks[strict]=true \
  -f required_status_checks[contexts][]=lint \
  -f required_status_checks[contexts][]=test \
  -f required_status_checks[contexts][]=build \
  -f enforce_admins=false \
  -f required_pull_request_reviews=null \
  -f restrictions=null
```

Recommended settings:
- ✅ Require status checks to pass before merging
- ✅ Require branches to be up to date before merging
- ✅ Include administrators (optional)
- ✅ Require linear history (optional)

### Enable Auto-Merge

```bash
# View repository settings
gh repo view --web
# Go to: Settings → General → Pull Requests
# Enable: "Allow auto-merge"
```

---

## Monitoring and Notifications

### Set Up Notifications

```bash
# Configure GitHub notifications for workflow failures
gh repo view --web
# Go to: Settings → Notifications
# Enable: "Actions" notifications
```

### View Release History

```bash
# List all releases
gh release list

# View a specific release
gh release view v1.0.0

# Download release assets
gh release download v1.0.0
```

### View Workflow Analytics

```bash
# View workflow runs in the browser
gh workflow view test.yml --web

# View a specific run
gh run view <run-id> --web
```

---

## Conventional Commit Examples

For the release automation to work, use these commit message formats:

```bash
# New feature (minor version bump: 1.0.0 → 1.1.0)
git commit -m "feat: add OAuth authentication support"
git commit -m "feat(auth): add support for Azure AD authentication"

# Bug fix (patch version bump: 1.0.0 → 1.0.1)
git commit -m "fix: handle empty work item response correctly"
git commit -m "fix(list): prevent panic when API returns null"

# Performance improvement (patch version bump)
git commit -m "perf: optimize work item query caching"

# Breaking change (major version bump: 1.0.0 → 2.0.0)
git commit -m "feat!: remove support for legacy API v1"
# OR
git commit -m "feat: migrate to new Azure DevOps API

BREAKING CHANGE: the legacy v1 API is no longer supported"

# No version bump (documentation, tests, chores)
git commit -m "docs: update README installation instructions"
git commit -m "test: add unit tests for config package"
git commit -m "chore: update dependencies"
git commit -m "ci: improve workflow performance"
```

---

## Quick Reference Commands

```bash
# Workflow Management
gh workflow list                    # List all workflows
gh workflow view <workflow>         # View workflow details
gh workflow run <workflow>          # Trigger workflow manually
gh workflow enable <workflow>       # Enable workflow
gh workflow disable <workflow>      # Disable workflow

# Run Management
gh run list                         # List recent workflow runs
gh run view                         # View latest run
gh run view <run-id>                # View specific run
gh run watch                        # Watch latest run in real-time
gh run watch <run-id>               # Watch specific run
gh run rerun <run-id>               # Rerun a failed workflow
gh run download <run-id>            # Download run artifacts/logs

# Release Management
gh release list                     # List all releases
gh release view <tag>               # View release details
gh release view --web               # Open latest release in browser
gh release download <tag>           # Download release assets

# Repository Secrets
gh secret list                      # List repository secrets
gh secret set <name>                # Set a secret
gh secret delete <name>             # Delete a secret

# Pull Requests
gh pr create                        # Create a pull request
gh pr list                          # List pull requests
gh pr view                          # View current PR
gh pr checks                        # View PR check status
gh pr merge                         # Merge PR
```

---

## Next Steps

After setting up the pipelines:

1. ✅ Verify CI runs on every PR
2. ✅ Test the release workflow by merging a feat/fix commit to main
3. ✅ Download and test the generated binaries
4. ✅ Set up branch protection rules
5. ✅ Configure Codecov (optional)
6. ✅ Add badges to README
7. ✅ Document the release process for contributors

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub CLI Documentation](https://cli.github.com/manual/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Semantic Release](https://semantic-release.gitbook.io/)
- [GoReleaser Documentation](https://goreleaser.com/intro/)
- [golangci-lint Linters](https://golangci-lint.run/usage/linters/)
