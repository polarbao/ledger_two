# {{PROJECT_NAME}} Codex Layer

`.codex/` is the project-scoped Codex execution layer.

## Boundaries

- `AGENTS.md` owns durable project instructions and Skill routing.
- `.agents/docs/**` owns longer project facts.
- `.agents/skills/**/SKILL.md` owns task-specific workflows.
- `{{FORMAL_DOCS_PATH}}` owns current formal PRD/DEV/DOC/ROADMAP/REPORT files.
- `{{CODEX_TASK_PATH}}` owns Codex task cards, prompts, and current task entry.
- `{{ARCHIVE_DOCS_PATH}}` owns historical docs and completed task references.
- `.codex/config.toml` owns official Codex runtime settings.
- `.codex/agents/*.toml` owns optional custom subagent profiles.
- `.codex/project-context.toml` owns human-readable project metadata.
- `.codex/hooks/**` documents optional hooks; hooks are disabled by default.
