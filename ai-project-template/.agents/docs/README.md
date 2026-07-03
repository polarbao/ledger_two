# {{PROJECT_NAME}} Agent Docs

This directory uses progressive disclosure:

- `AGENTS.md` keeps always-on rules and routing short.
- `.agents/docs/**` stores longer project facts and policies.
- `.agents/skills/**/SKILL.md` stores task-specific workflows.
- `{{FORMAL_DOCS_PATH}}` stores current formal PRD/DEV/DOC/ROADMAP/REPORT documents.
- `{{CODEX_TASK_PATH}}` stores current Codex task cards and prompts.
- `{{ARCHIVE_DOCS_PATH}}` stores historical docs and completed-stage references.
- `.codex/**` stores Codex runtime settings and optional custom agents.

## File Map

- `always-on-rules.md`: hard rules that must survive refactors.
- `workflow-map.md`: default development workflow and Skill routing.
- `project-profile.md`: project identity, current phase, tech stack, and major capabilities.
- `architecture-boundary.md`: module boundaries and dependency direction.
- `build-and-test.md`: build, dependency, test, packaging, and deployment notes.
- `code-standards.md`: language/framework style, comments, ownership, error handling, and external boundaries.
- `doc-state.md`: how to resolve conflicting current/historical docs.
- `chat-save.md`: how to save useful AI conversation context when explicitly requested.
