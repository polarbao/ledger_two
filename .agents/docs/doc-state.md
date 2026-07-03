# Documentation State

Use this order when docs conflict:

1. Current source, build files, tests, runtime config, verified command output.
2. Latest stage reports that list concrete commands and observed results.
3. Current formal PRD/DEV/DOC/ROADMAP/ADR documents under `docs`.
4. Current Codex task card under `docs/codex_tasks`.
5. Historical demos, archived docs, old task cards, or old discussions under `docs/project_analysis/extracted_archives and ai_workspace`.
6. Deprecated or conflicting material.

Label evidence as A/B/C/D for high-risk decisions.

## Evidence levels

```text
A = current code/config/tests; safe implementation basis
B = formal target design; direction only
C = historical reference; background only
D = deprecated/conflicting; not implementation basis
```

For architecture, protocol, production, or implementation-state questions, split conclusions into:

```text
Current State
Target State
Historical State
Pending Confirmation
```
