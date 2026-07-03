---
name: project-doc-state-resolver
description: Use for {{PROJECT_NAME}} questions about whether a feature is implemented, only designed, historical, deprecated, or conflicting across docs and source code.
---

# {{PROJECT_NAME}} Doc State Resolver

Read first:

1. `AGENTS.md`
2. `.agents/docs/doc-state.md`
3. Current source/build/tests before historical docs
4. Current formal docs under `{{FORMAL_DOCS_PATH}}`
5. Current task cards under `{{CODEX_TASK_PATH}}`

## Resolution Order

1. Current source, build files, tests, runtime config, verified command output.
2. Current accepted design docs, PRDs, ADRs, or plans.
3. Historical demos, archived docs, or old discussions.
4. Deprecated or conflicting material.

Output clear evidence labels: A/B/C/D.
