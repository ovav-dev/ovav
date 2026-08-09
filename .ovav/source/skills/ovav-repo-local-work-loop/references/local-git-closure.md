# Local Git Closure Rules

## Core Principle

Git closure is local-only. No remote/origin workflow is required or assumed.

## Forbidden Operations

- `git push` (any form)
- `git pull` (any form)
- `git fetch origin`
- `git branch -d` / `git branch -D` / `git branch --delete`
- branch creation/switching only with explicit task scope and user approval
- `git add .` (stage everything blindly)

## Allowed Operations

- `git status`
- `git diff`
- `git diff --cached`
- `git log`
- `git add <exact files>` (only current-segment files, never broad folders)

## Stage Rules

- Stage exact files only. Never stage broad folders.
- Block: zips, PDFs, logs, backups, .config, .local, node_modules, .pytest_cache, __pycache__, generated noise.
- Stage only files created or modified in the current segment.

## Commit Rules

- Do NOT commit unless user explicitly approves.
- When approved: exact commit with scope-accurate message.
- Never amend or force-push.

## Historical Drift Restoration

- Some validators may rewrite historical evidence files. Treat that drift as suspect unless current scope owns it.
- After validation, always restore any historical evidence drift.
- Check for drift: `git diff -- .ovav/artifacts/S*`
- If historical artifacts changed unintentionally, restore them.
