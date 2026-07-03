# Applying The AI Project Template

This guide explains how to apply `ai-project-template` to a new repository.

## 1. When To Use This Template

Use this template when a project needs durable AI collaboration rules:

- repeatable coding/review/testing expectations;
- project-specific build and verification commands;
- task-specific Codex Skills;
- optional Codex custom agents;
- a clear split between always-on instructions and on-demand details.

Do not use this template as a dumping ground for long project manuals. Keep
`AGENTS.md` short, and move details into `.agents/docs`.

## 2. What The Template Installs

```text
target-project/
├── AGENTS.md
├── .agents/
│   ├── docs/
│   │   ├── README.md
│   │   ├── always-on-rules.md
│   │   ├── workflow-map.md
│   │   ├── project-profile.md
│   │   ├── architecture-boundary.md
│   │   ├── build-and-test.md
│   │   ├── code-standards.md
│   │   ├── doc-state.md
│   │   ├── chat-save.md
│   │   └── verification.md  (optional)
│   └── skills/
│       ├── project-dev-workflow/
│       ├── project-code-review/
│       ├── project-architecture-guardrails/
│       ├── project-build/
│       ├── project-context-handoff/
│       ├── project-doc-state-resolver/
│       ├── project-chat-save/
│       └── project-verification/  (optional)
├── {{FORMAL_DOCS_PATH}}/
├── {{CODEX_TASK_PATH}}/
├── {{ARCHIVE_DOCS_PATH}}/
└── .codex/
    ├── config.toml
    ├── README.md
    ├── project-context.toml
    ├── agents/
    │   ├── pm.toml
    │   ├── explorer.toml
    │   ├── builder.toml
    │   ├── tester.toml
    │   └── reporter.toml
    └── hooks/
        └── README.md
```

## 3. Copy Into A New Project

From this repository:

```powershell
$template = "E:\__Code\__Work\slice_test_demo\slice_soft_demo\ai-project-template"
$target = "E:\path\to\new-project"
Copy-Item -Path "$template\*" -Destination $target -Recurse -Force
```

If the target already has `AGENTS.md`, `.agents`, or `.codex`, do not overwrite
blindly. Merge manually and preserve the target project's existing hard rules.

## 4. Replace Placeholders

Search all placeholders:

```powershell
rg "\{\{[A-Z0-9_]+\}\}" E:\path\to\new-project
```

Common replacements:

| Placeholder | Meaning | Example |
| --- | --- | --- |
| `{{PROJECT_NAME}}` | Human project name | `BillingService` |
| `{{REPOSITORY_NAME}}` | Repo or org/repo name | `acme/billing-service` |
| `{{CURRENT_BRANCH_OR_REF}}` | Main branch or current baseline | `main` |
| `{{CURRENT_PHASE}}` | Current project phase | `R2 hardening` |
| `{{PROJECT_DOMAIN}}` | Project domain | `industrial slicing software` |
| `{{MAIN_APP_PATH}}` | Primary implementation path | `src` |
| `{{DEPRECATED_PATHS}}` | Historical/deprecated implementation paths | `legacy` |
| `{{TEST_PATH}}` | Test path | `tests` |
| `{{TECH_STACK}}` | Main stack | `TypeScript, Node.js, PostgreSQL` |
| `{{BUILD_SYSTEM}}` | Build system | `CMake` |
| `{{DEFAULT_LANGUAGE}}` | User-facing response language | `zh-CN` |
| `{{ENGINEERING_ROLE}}` | Desired engineering perspective | `senior backend engineer` |
| `{{BUILD_TOOL}}` | Build tool | `pnpm`, `cmake`, `gradle` |
| `{{PACKAGE_MANAGER}}` | Dependency manager | `pnpm`, `vcpkg`, `poetry` |
| `{{BUILD_COMMAND}}` | Main build command | `pnpm build` |
| `{{TEST_COMMAND}}` | Main test command | `pnpm test` |
| `{{QUICK_VALIDATION_COMMAND}}` | Fast local validation | `pnpm test -- --runInBand` |
| `{{UI_SMOKE_COMMAND}}` | UI smoke command, if any | `npm run test:ui-smoke` |
| `{{INTEGRATION_TEST_COMMAND}}` | Integration command, if any | `pnpm test:integration` |
| `{{FORMAL_DOCS_PATH}}` | Current formal PRD/DEV/docs | `docs/slice` |
| `{{CODEX_TASK_PATH}}` | Codex task-card path | `docs/codex_task` |
| `{{ARCHIVE_DOCS_PATH}}` | Old docs or demos | `docs/archive` |
| `{{CURRENT_TASK_FILE}}` | Current task-card entry | `docs/codex_task/current/TASKS_current.md` |
| `{{AI_WORKSPACE_PATH}}` | Durable AI context path | `ai_workspace` |

## 5. Rename Skills

The template uses generic Skill names such as `project-dev-workflow`.

For one-off personal projects, generic names are acceptable. For team projects
or multiple repos on the same machine, rename them to avoid collisions:

```text
project-dev-workflow           -> billing-dev-workflow
project-code-review            -> billing-code-review
project-architecture-guardrails -> billing-architecture-guardrails
project-build                  -> billing-build
project-context-handoff        -> billing-context-handoff
project-doc-state-resolver     -> billing-doc-state-resolver
project-chat-save              -> billing-chat-save
```

After renaming folders, update each `SKILL.md` front matter:

```yaml
---
name: billing-dev-workflow
description: Use for BillingService feature planning...
---
```

Also update `AGENTS.md` Skill routing.

## 6. Fill Project Docs

Minimum required edits:

- `AGENTS.md`: project identity, build command, test command, Skill names.
- `.agents/docs/project-profile.md`: project identity, current phase, main paths, core capabilities, and risk areas.
- `.agents/docs/architecture-boundary.md`: current modules and dependency direction.
- `.agents/docs/build-and-test.md`: build tool, package manager, dependency policy, exact commands, and what each command proves.
- `.agents/docs/code-standards.md`: language/framework style, comments, ownership, async/threading, error handling, and external boundaries.
- `.agents/docs/doc-state.md`: source-of-truth order for current vs historical docs.
- `.agents/docs/chat-save.md`: AI conversation save rules.
- `.codex/project-context.toml`: human-readable metadata and verification commands.

Keep project facts in docs, not in `.codex/config.toml`.

## 7. Tune Codex Runtime Settings

`.codex/config.toml` should contain official Codex runtime settings only.

Recommended default:

```toml
approval_policy = "on-request"
sandbox_mode = "workspace-write"
web_search = "cached"
model_reasoning_effort = "high"
personality = "friendly"

[features]
hooks = false
multi_agent = true
shell_snapshot = true

[agents]
max_threads = 2
max_depth = 1
```

Adjust per project:

- Use `approval_policy = "on-request"` for normal development.
- Avoid `approval_policy = "never"` in repos with production data, deployment scripts, hardware access, or destructive commands.
- Keep hooks disabled until scripts are reviewed and trusted.
- Keep `max_depth = 1` unless recursive subagent fan-out is intentionally needed.

## 8. Validate The Installation

Run from the new project root:

```powershell
rg "\{\{[A-Z0-9_]+\}\}"
```

No placeholders should remain unless intentionally documented.

Validate TOML:

```powershell
@'
import pathlib, tomllib
for path in list(pathlib.Path(".codex").rglob("*.toml")):
    with path.open("rb") as f:
        tomllib.load(f)
    print(f"OK {path}")
'@ | python -
```

Validate custom agents:

```powershell
@'
import pathlib, tomllib
for path in pathlib.Path(".codex/agents").glob("*.toml"):
    data = tomllib.loads(path.read_text(encoding="utf-8"))
    missing = [k for k in ("name", "description", "developer_instructions") if not data.get(k)]
    if missing:
        raise SystemExit(f"{path} missing {missing}")
    print(f"AGENT OK {path.name}: {data['name']}")
'@ | python -
```

Review code-style placeholders:

```powershell
rg "\{\{[A-Z0-9_]+\}\}" .agents/docs/code-standards.md
```

Ask Codex to summarize active guidance:

```text
Read AGENTS.md, .agents/docs/README.md, {{FORMAL_DOCS_PATH}}, and {{CODEX_TASK_PATH}}.
Summarize the active AI guidance layers, current task entry, evidence rules, and remaining placeholders.
Do not modify source code yet.
```

## 9. Maintenance Rules

- Keep `AGENTS.md` short and always-on.
- Keep long rules in `.agents/docs`.
- Keep each Skill focused on one job.
- Keep project-specific facts out of global/user skills.
- Keep `.codex/config.toml` to supported runtime settings.
- Use `.codex/project-context.toml` or docs for human-readable metadata.
- Convert the template into a plugin only when you need installable distribution across teams or machines.
