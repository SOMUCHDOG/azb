# Branch Protection Configuration for Automated Releases

## Issue

The semantic-release workflow needs to push commits (version bumps and CHANGELOG updates) directly to the `main` branch. However, branch protection rules require all changes to go through pull requests.

## Solution

Use a Personal Access Token (PAT) with `repo` scope to allow the release workflow to bypass branch protection.

## Setup Steps

### 1. Create a Personal Access Token

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Or visit: https://github.com/settings/tokens/new

2. Configure the token:
   - **Note**: `Semantic Release Token for ado-admin`
   - **Expiration**: Choose based on your preference (90 days, 1 year, or no expiration)
   - **Scopes**:
     - ✅ **`repo`** (Full control of repositories)

3. Click **"Generate token"** and copy it immediately

### 2. Add Token as Repository Secret

Via GitHub CLI:
```bash
gh secret set SEMANTIC_RELEASE_TOKEN
# Paste your token when prompted
```

Via Web UI:
1. Go to repository → Settings → Secrets and variables → Actions
2. Click **"New repository secret"**
3. Name: `SEMANTIC_RELEASE_TOKEN`
4. Value: Paste your token
5. Click **"Add secret"**

### 3. Update Release Workflow

The workflow has been configured to use this PAT:

```yaml
- uses: actions/checkout@v4
  with:
    token: ${{ secrets.SEMANTIC_RELEASE_TOKEN }}

- name: Run semantic-release
  env:
    GITHUB_TOKEN: ${{ secrets.SEMANTIC_RELEASE_TOKEN }}
```

## What This Allows

With this configuration:

✅ **Automated releases work** - semantic-release can push version commits
✅ **Bypasses branch protection** - PAT has full repo permissions
✅ **Security maintained** - Token is stored as encrypted secret
✅ **Audit trail preserved** - All changes tracked in workflow logs

## Token Maintenance

- **Expiration**: If your token expires, releases will fail. Create a new token and update the secret.
- **Revocation**: If compromised, revoke the token and create a new one immediately.
- **Permissions**: Only grant `repo` scope - do not add unnecessary permissions.

## Verification

After configuration, test the release workflow:

```bash
# Manual trigger (if workflow_dispatch is configured)
gh workflow run release.yml --ref main

# Or push a commit with conventional commit message
git commit -m "feat: new feature" --allow-empty
git push origin main

# Watch the workflow
gh run watch
```

## Troubleshooting

### Token expires
- Create a new PAT with the same configuration
- Update the `SEMANTIC_RELEASE_TOKEN` secret with the new token

### Permission denied errors
- Verify the PAT has `repo` scope
- Check that the secret name is exactly `SEMANTIC_RELEASE_TOKEN`
- Ensure the token hasn't expired

### Workflow still fails
- Check workflow logs: `gh run view --log`
- Verify the secret is accessible in the workflow environment
