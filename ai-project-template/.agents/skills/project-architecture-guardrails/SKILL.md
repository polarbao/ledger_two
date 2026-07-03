---
name: project-architecture-guardrails
description: Use for {{PROJECT_NAME}} architecture boundary decisions, module responsibility checks, dependency direction, public API changes, and ADR creation.
---

# {{PROJECT_NAME}} Architecture Guardrails

Read first:

1. `AGENTS.md`
2. `.agents/docs/project-profile.md`
3. `.agents/docs/architecture-boundary.md`
4. `.agents/docs/doc-state.md`
5. Relevant formal PRD/DEV/DOC files under `{{FORMAL_DOCS_PATH}}`

## Rules

- Classify evidence as A/B/C/D when current behavior and target design differ.
- Preserve dependency direction and public API boundaries.
- Do not introduce new cross-layer dependencies without impact analysis.
- For long-lived or irreversible choices, create or update an ADR/DOC_DECISION under `{{FORMAL_DOCS_PATH}}`.

## Output

- Current State
- Target State
- Boundary impact
- Risks
- Verification plan
