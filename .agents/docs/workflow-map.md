# Workflow Map

## Default Flow

1. Restate the objective and constraints.
2. Read necessary files and docs.
3. Identify affected boundaries and risks.
4. Implement small changes directly when safe.
5. Pause for confirmation before large or high-risk changes.
6. Run or recommend relevant verification.
7. Summarize changes, verification, and residual risk.

## Task Card Flow

For Codex task-card based work:

1. Read `AGENTS.md`, `.agents/docs/README.md`, and the current task file.
2. Run `git status --short` before changing files.
3. Execute only the requested task card.
4. Use the task card's allowed files, forbidden actions, validation commands, and completion criteria.
5. Do not execute the next task unless the user explicitly asks.

Current task entry placeholder:

```text
docs/codex_tasks/05-foundation-task-plan.md
```

## Routing

- Planning: `project-dev-workflow`
- Review: `project-code-review`
- Architecture boundaries / ADRs: `project-architecture-guardrails`
- Build/dependencies/tests: `project-build`
- Handoff: `project-context-handoff`
- Doc conflicts: `project-doc-state-resolver`
- Chat save: `project-chat-save`
