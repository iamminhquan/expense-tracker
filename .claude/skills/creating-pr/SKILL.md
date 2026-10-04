---
name: creating-pr
description: Create a clean, review-ready pull request with a good title, structured description, linked issues, and appropriate reviewers.
user-invocable: true
---

# Creating a PR

Package your work into a pull request that's easy for others to review and merge.

## Workflow

### 1. Prepare the Branch

Before creating the PR, make sure your branch is in good shape:

```bash
# Ensure branch is up to date with the base (usually main)
git fetch origin
git rebase origin/main  # or merge, depending on project preference

# Review what will be in the PR
git log origin/main..HEAD --oneline
git diff origin/main --stat
```

Squash fixup commits if the project prefers clean history. Keep logical commits separate if the project prefers granular history.

### 2. Write the Title

Use a clear, structured format: `<type>: <short description>`

| Type | When |
|------|------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `docs` | Documentation only |
| `test` | Adding or fixing tests |
| `chore` | Build, CI, dependencies, or tooling |
| `perf` | Performance improvement |

Examples:
- `feat: add dark mode toggle to settings page`
- `fix: prevent duplicate form submissions on checkout`
- `refactor: extract auth middleware into shared module`

### 3. Write the Description

Structure it like this:

```markdown
## Summary

1–3 sentences explaining what this PR does and why it matters.

Closes #123

## Changes

- Added `ThemeToggle` component with system/light/dark options
- Updated `Layout` to read theme from context
- Added theme persistence to localStorage

## Test Plan

- [ ] Toggle between light/dark/system themes
- [ ] Refresh page — theme persists
- [ ] Check no flash of unstyled content on load
```

Keep it focused and scannable. Reviewers should understand the "what" and "why" quickly.

### 4. Self-Review

Before requesting review:
- Read every line of the diff yourself
- Remove debug code (`console.log`, `TODO`, commented-out code)
- Verify tests pass
- Verify types and linting pass
- Check for files that shouldn't be committed (`.env`, lockfile conflicts, IDE files)

### 5. Create the PR

```bash
git push -u origin HEAD
gh pr create --title "<title>" --body "$(cat <<'EOF'
## Summary
...

## Changes
...

## Test Plan
...
EOF
)"
```

Or use your platform's UI if you prefer.

### 6. Request Review

- Tag the appropriate reviewers (code owners, domain experts, people who've touched this area)
- If the PR is large (>400 lines), add a comment suggesting the best order to review files
- If the PR depends on another PR, note it in the description
- Apply labels as appropriate (feature, bug, breaking change, etc.)

## Tips for a Smooth Review

- **Small PRs get reviewed faster** — aim for <300 lines changed
- **If a PR is too big, split it into stacked PRs** that build on each other
- **Screenshots/videos for UI changes** make review much faster — show what the user sees
- **Draft PRs are useful for early feedback** before the work is complete
- **Respond to feedback promptly** — keeps momentum and shows you're engaged
