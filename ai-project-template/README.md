# AI Project Template

This template provides a reusable Codex setup for a repository.

Copy the contents of this directory into another project root, then replace all
`{{PLACEHOLDER}}` values before using it.

Detailed usage guides:

- `EXPORTING_TEMPLATE.md`: export this template to a local template library or another repository.
- `APPLYING_TEMPLATE.md`: apply and customize the template inside a target repository.

## Layers

- `AGENTS.md`: durable project instructions, hard rules, output language, and Skill routing.
- `.agents/docs/**`: long project facts, boundaries, verification commands, and workflow notes.
- `.agents/skills/**/SKILL.md`: task-specific workflows.
- `{{FORMAL_DOCS_PATH}}`: current formal PRD/DEV/DOC/ROADMAP/REPORT entry.
- `{{CODEX_TASK_PATH}}`: current Codex task cards, prompts, and task archive.
- `{{ARCHIVE_DOCS_PATH}}`: historical docs and completed-stage references.
- `.codex/config.toml`: official Codex runtime configuration.
- `.codex/agents/*.toml`: optional custom subagent roles.
- `.codex/project-context.toml`: human-readable project metadata; not runtime config.
- `.codex/hooks/**`: optional hook scripts or notes; disabled by default.

## Replacement Checklist

- Replace `{{PROJECT_NAME}}`, `{{REPOSITORY_NAME}}`, `{{MAIN_APP_PATH}}`, and `{{TECH_STACK}}`.
- Replace `{{FORMAL_DOCS_PATH}}`, `{{CODEX_TASK_PATH}}`, `{{ARCHIVE_DOCS_PATH}}`, and `{{CURRENT_TASK_FILE}}`.
- Replace build/test commands in `.agents/docs/build-and-test.md` and `.codex/project-context.toml`.
- Replace language/framework style placeholders in `.agents/docs/code-standards.md`.
- Rename Skill folders from `project-*` if the target repository needs project-specific prefixes.
- Remove irrelevant historical-reference sections.
- Keep project facts out of global/user skills.

## Recommended First Prompt

```text
Read AGENTS.md, .agents/docs/README.md, {{FORMAL_DOCS_PATH}}, and {{CODEX_TASK_PATH}}.
Summarize this project's AI guidance layers, current task entry, evidence rules, and remaining placeholders.
Do not modify source code yet.
```
