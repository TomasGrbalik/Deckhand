---
name: release
description: Create a new release by tagging and pushing. Use when user says "release", "create a release", "tag a release", "publish a new version". Runs pre-release checks, determines version bump, creates annotated tag, and pushes to trigger the release workflow.
disable-model-invocation: true
---

# Release

Create a new deckhand release by determining the version bump, running checks, and pushing a tag.

## Current State

- Branch: !`git branch --show-current`
- Latest tag: !`git describe --tags --abbrev=0 2>/dev/null || echo "none"`
- Recent commits: !`git log --oneline --no-merges -20`

## Workflow

### Step 1: Validate prerequisites

Before anything else, verify:
1. On the `main` branch — refuse to release from other branches
2. Working tree is clean (`git status --porcelain` is empty)
3. Up to date with remote (`git fetch origin main && git diff HEAD origin/main --quiet`)
4. `.goreleaser.yaml` passes validation (`goreleaser check`)

If any check fails, stop and tell the user what to fix.

### Step 2: Determine version

If `$ARGUMENTS` contains a version (e.g., `v1.0.0`), use that directly. Otherwise:

1. Find the latest tag (or treat as first release if none exist)
2. Parse all commits since the last tag
3. Apply conventional commit rules:
   - Any `!:` or `BREAKING CHANGE:` footer → **MAJOR** bump
   - Any `feat:` → **MINOR** bump
   - Everything else → **PATCH** bump
4. Calculate the new version
5. Present the suggested version and commit summary to the user
6. Ask for confirmation before proceeding using AskUserQuestion

### Step 3: Pre-release checks

Run these checks and report results:
```
go test ./...
go build ./cmd/deckhand
goreleaser check
```

If any fail, stop and report the failure.

### Step 4: Tag and push

```bash
git tag -a <version> -m "Release <version>"
git push origin <version>
```

Report success and link to the GitHub Actions workflow run that will build the binaries.
