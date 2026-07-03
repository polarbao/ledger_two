# LedgerTwo Codex Layer

`.codex/` is the project-scoped Codex execution layer.

## Boundaries

- `AGENTS.md` owns durable project instructions and Skill routing.
- `.agents/docs/**` owns longer project facts.
- `.agents/skills/**/SKILL.md` owns task-specific workflows.
- `docs` owns current formal PRD/DEV/DOC/ROADMAP/REPORT files.
- `docs/codex_tasks` owns Codex task cards, prompts, and current task entry.
- `docs/project_analysis/extracted_archives and ai_workspace` owns historical docs and completed task references.
- `.codex/config.toml` owns official Codex runtime settings.
- `.codex/agents/*.toml` owns optional custom subagent profiles.
- `.codex/project-context.toml` owns human-readable project metadata.
- `.codex/hooks/**` documents optional hooks; hooks are disabled by default.
