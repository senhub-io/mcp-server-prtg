# Release Process

This document describes the standard release workflow for mcp-server-prtg following GitFlow best practices.

## Release Workflow Overview

```
feature branch → dev (beta release) → PR → main (production release)
```

## Step-by-Step Process

### 1. Development Phase

1. Create a feature branch from `dev`:
   ```bash
   git checkout dev
   git pull origin dev
   git checkout -b feature/your-feature-name
   ```

2. Develop and commit changes following commit conventions
3. Ensure all tests pass locally: `make test`

### 2. Beta Release on Dev

**IMPORTANT: Always create a beta release on dev before merging to main.**

1. **Merge feature → dev**:
   ```bash
   git checkout dev
   git merge feature/your-feature-name
   ```

2. **Push to origin/dev**:
   ```bash
   git push origin dev
   ```

3. **Wait for GitHub Actions** to pass (Go Test workflow)

4. **Create and push tag** to trigger beta release:
   ```bash
   git tag X.Y.Z -m "Beta release X.Y.Z - Description"
   git push origin X.Y.Z
   ```

5. **GitHub Actions automatically**:
   - Detects tag is on dev branch
   - Creates `X.Y.Z-beta` tag
   - Builds binaries for all platforms
   - Creates pre-release on GitHub with binaries

6. **Test the beta release** with real users/systems

### 3. Production Release via Pull Request

1. **Create PR** from dev to main:
   ```bash
   gh pr create --base main --head dev --title "Release X.Y.Z" --body "Release notes..."
   ```

2. **Review and approve PR** (code review, checks pass)

3. **Merge PR** to main:
   ```bash
   gh pr merge <PR-NUMBER> --merge
   ```

4. **Create production release**:
   ```bash
   git checkout main
   git pull origin main
   make release VERSION=X.Y.Z
   ```

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (X.0.0): Breaking changes
- **MINOR** (X.Y.0): New features, backward compatible
- **PATCH** (X.Y.Z): Bug fixes, backward compatible

### Examples:
- `1.2.2`: Patch release (bug fixes)
- `1.3.0`: Minor release (new features)
- `2.0.0`: Major release (breaking changes)

## Beta Release Tags

- Dev branch tags: `X.Y.Z` → automatically becomes `X.Y.Z-beta`
- Main branch tags: `X.Y.Z` → remains `X.Y.Z` (production)

**The GitHub Actions workflow `dev-beta-release.yml` handles beta releases automatically.**

## Makefile Automation

The Makefile provides automation for releases:

```bash
# Build all binaries
make build-all

# Create release (with tag)
make release VERSION=X.Y.Z

# Run quality checks
make quality-check
```

## GitHub Workflows

### 1. Go Test (`go-test.yml`)
- Triggers on: push, pull request
- Runs on: all branches
- Purpose: Run tests, ensure code quality

### 2. Dev Beta Release (`dev-beta-release.yml`)
- Triggers on: tag push (format X.Y.Z)
- Runs on: dev branch only
- Purpose: Create beta pre-releases with binaries

### 3. Main Release (`main-release.yml`)
- Triggers on: tag push (format X.Y.Z)
- Runs on: main branch only
- Purpose: Create production releases with binaries

## Release Checklist

Before creating a release, ensure:

- [ ] All features are tested locally
- [ ] All tests pass: `make test`
- [ ] Code quality checks pass: `make quality-check`
- [ ] Documentation is updated
- [ ] CHANGELOG is updated (if applicable)
- [ ] Beta release tested on dev
- [ ] PR reviewed and approved
- [ ] Version number follows semantic versioning

## Emergency Hotfix Process

For critical production issues:

1. Create hotfix branch from main:
   ```bash
   git checkout main
   git checkout -b hotfix/critical-issue
   ```

2. Fix the issue and test

3. Create PR to main (skip beta for critical fixes)

4. After merge, backport to dev:
   ```bash
   git checkout dev
   git merge main
   git push origin dev
   ```

## Notes for Release-Manager Agent

When using the `release-manager` agent for releases:

1. **ALWAYS create beta release on dev first**
2. **NEVER merge directly to main** - use PR workflow
3. **Wait for GitHub Actions** to complete before proceeding
4. **Ask for confirmation** before any remote operations (push, tag, release)
5. **Tag format**: Use `X.Y.Z` (no "v" prefix)
6. **Use Makefile** commands when available

## Example: Complete Release Flow

```bash
# 1. Feature development
git checkout -b feature/prtg-api-v2
# ... develop ...
git commit -m "feat: add PRTG API v2 integration"

# 2. Merge to dev
git checkout dev
git merge feature/prtg-api-v2
git push origin dev

# 3. Wait for tests to pass
gh run watch --exit-status

# 4. Create beta release
git tag 1.2.2 -m "Beta release 1.2.2"
git push origin 1.2.2
# Wait for beta build to complete

# 5. Create PR to main
gh pr create --base main --head dev --title "Release 1.2.2"

# 6. Merge PR (after review)
gh pr merge <PR#> --merge

# 7. Create production release
git checkout main
git pull origin main
make release VERSION=1.2.2
```

## Troubleshooting

### Beta release not triggered
- Check tag is on dev branch: `git branch --contains <tag>`
- Check workflow logs: `gh run list --workflow=dev-beta-release.yml`

### Build fails
- Check Go version in workflow matches project: `go.mod`
- Check ldflags are properly escaped
- Review workflow logs: `gh run view <run-id>`

### Release automation issues
- Ensure `gh` CLI is authenticated: `gh auth status`
- Check repository permissions for GitHub Actions
- Verify `GITHUB_TOKEN` has appropriate scopes

## References

- [GitFlow Workflow](https://nvie.com/posts/a-successful-git-branching-model/)
- [Semantic Versioning](https://semver.org/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub CLI Documentation](https://cli.github.com/manual/)
