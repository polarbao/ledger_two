# Always-On Rules

- Default response language: `{{DEFAULT_LANGUAGE}}`.
- Act from a senior engineering perspective.
- Read relevant files before editing.
- Do not invent command output, tests, builds, deployment status, or repository state.
- Keep changes small unless the user approves a larger plan.
- Protect user changes in a dirty working tree.
- Explain impact and verification for code, architecture, dependency, and build changes.
- Before starting a task, inspect the working tree with `git status --short`.
- Execute only the user-requested task. Do not automatically proceed to the next task card.
- Keep formal docs, Codex task cards, and historical archives in separate paths:
  - formal docs: `{{FORMAL_DOCS_PATH}}`
  - Codex tasks: `{{CODEX_TASK_PATH}}`
  - history/archive: `{{ARCHIVE_DOCS_PATH}}`
- Follow `.agents/docs/code-standards.md` for language/framework style, comments, ownership, and boundary rules.
