---
name: ovav-release
description: OVAV release workflow (versioning, changelog, tag, deploy).
trigger: release, version, tag, deploy, changelog
---

# ovav-release

OVAV release workflow. Versioned by SemVer, governed by OWS.

## Workflow

```bash
# 1. Pre-release
ovav worktree owv                          # full validation
ovav worktree owl                          # check worktree state

# 2. Bump version
# Edit VERSION file (single source of truth)
# Edit CHANGELOG.md (per release)

# 3. Commit + merge to develop
git add VERSION CHANGELOG.md
git commit -m "chore(release): v<new-version>"
ovav worktree owd

# 4. Cut release branch
ovav worktree create release-v<new-version>

# 5. Tag
git tag -s v<new-version> -m "v<new-version>"

# 6. Push to origin (with auth)
git push origin develop
git push origin release-v<new-version>
git push origin v<new-version>

# 7. Deploy (if applicable)
ovav deploy stage
ovav deploy production
```

## Versioning

| Bump | When |
|---|---|
| Major (X.0.0) | Breaking changes, API redesign |
| Minor (0.X.0) | New features, backwards-compatible |
| Patch (0.0.X) | Bug fixes, internal changes |

## Rules

- Never push `--force` to release tags
- Always sign tags with GPG (`-s`)
- CHANGELOG entry mandatory (P3 compatibility)
- Run `owv` before any release commit
- `owv` must pass with 0 warnings before release

## Rollback

```bash
# Revert tag
git tag -d v<version>
git push origin --delete v<version>

# Revert release commit
ovav worktree owr  # rescue
ovav worktree owa  # abort
```

## Failure modes

- "GPG signature failed" → configure signing key
- "protected branch" → use `release-v*` naming
- "OWS denied merge" → `ovav validate` for details
