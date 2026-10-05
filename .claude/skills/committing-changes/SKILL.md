---
name: committing-changes
description: Commit working-tree changes following $pend's convention — atomic commits, `<type>: ` subjects, main-vs-branch rules, and the pre-commit checks. Use when asked to commit or save work, or when a finished chunk of work sits uncommitted.
---

# Committing Changes

Cut your work into commits that are cheap to read later — in `git log`, in `git bisect`, in review.

## Workflow

### 1. See What Changed

Run `git status` and `git diff` (plus `git diff --staged` once something is staged). Read the diff; don't commit from memory of what you edited.

### 2. Run the Checks

This is a monorepo (`server/` + `client/`) — run the checks for whichever package the diff touches; run both if it touches both.

Touched `server/`? All three must pass, run from inside `server/`:

```bash
go build ./...
gofmt -l .        # must print nothing
go vet ./...
```

Touched `client/`? Both must pass, run from inside `client/`:

```bash
pnpm lint
pnpm build
```

Fix any failure first. A late commit beats a commit that breaks the build.

### 3. Split into Atomic Commits

One reviewable idea per commit.

- Unrelated edits get separate commits, even in the same file — stage by hunk with `git add -p`.
- Don't split one coherent change (e.g. a function and its test) into pieces.

### 4. Pick `main` or a Branch

| Change | Where |
|--------|-------|
| Typo, one-line style tweak, doc-only edit | Straight to `main` |
| Feature, multi-file refactor, behavior change | Branch `<type>/<short-description>` (e.g. `feat/mobile-ux-polish`), then `git merge --no-ff` into `main` |

Already on a branch started for other work? Keep committing there.

### 5. Write the Message

**Subject**: `<type>: <description>` — English, 72–100 characters including the prefix, no scope.

| Type | When |
|------|------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Neither fixes a bug nor adds a feature |
| `docs` | Documentation only |
| `test` | Adding or fixing tests |
| `style` | Formatting, no behavior change |
| `chore` | Build, dependencies, tooling |

Always use the prefix; the few unprefixed commits in history are drift.

**Body**: say *why*, not *what* — what would break without this, what alternative was rejected, what constraint shaped it. Write prose paragraphs, not bullet changelogs. Skip it when the diff is self-explanatory.

**No footer**: end the message at the body. No `Co-Authored-By` trailer, no "🤖 Generated with Claude Code" line and no `Claude-Session:` link (this repo's convention, even though older commits carry a session link).

### 6. Commit and Confirm

For each group: stage it, switch branch if step 4 says so, then `git commit`. When done, show `git log --oneline -n <count>` so the split is easy to check.

## Examples

Subject only, because the diff is the whole story:

```
test: pin that a filtered page holds a full page of rows, and a pager sized to the matches
```

Subject and body, because the *why* isn't in the diff:

```
refactor: serve the embedded static tree through FileServerFS and one build-time ETag

The handler held every asset a second time in a map, hashed each one and
guessed its Content-Type from the extension -- all of which FileServerFS
already does off the embed.FS. What it cannot do is produce an ETag:
embed.FS reports a zero ModTime, so there is no Last-Modified to
revalidate against. One digest over the whole tree covers it.
```

## Tips

- **Unsure about the branch or the split?** Ask instead of guessing.
- **Check `git log`** for the texture to match before writing a body.
