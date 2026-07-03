# Exporting The AI Project Template

本文档说明如何把当前仓库中的 `.agents` / `.codex` 模板导出到其他项目使用。

模板源路径：

```text
E:\__Code\__Work\slice_test_demo\slice_soft_demo\ai-project-template
```

## 1. 推荐导出方式

推荐分三层使用：

1. 仓库内模板源：当前项目中的 `ai-project-template`。
2. 本机全局模板库：例如 `E:\__Code\__Templates\codex-ai-project-template`。
3. 目标项目实例：复制到具体项目根目录后替换占位符。

不要直接把当前项目的 `.agents` / `.codex` 整包复制到其他项目。应复制模板目录 `ai-project-template`，因为它使用 `{{PLACEHOLDER}}` 占位符并保留通用结构。

## 2. 导出到本机模板库

建议先创建一个本机模板库目录：

```powershell
$source = "E:\__Code\__Work\slice_test_demo\slice_soft_demo\ai-project-template"
$templateRoot = "E:\__Code\__Templates\codex-ai-project-template"

New-Item -ItemType Directory -Force $templateRoot | Out-Null
Copy-Item -Path "$source\*" -Destination $templateRoot -Recurse -Force
```

导出后建议检查模板文件：

```powershell
Get-ChildItem -Recurse $templateRoot | Select-Object FullName
```

## 3. 导出到一个新项目

目标项目示例：

```powershell
$templateRoot = "E:\__Code\__Templates\codex-ai-project-template"
$target = "E:\__Code\__Work\your-new-project"

Copy-Item -Path "$templateRoot\*" -Destination $target -Recurse -Force
```

如果目标项目已经存在这些文件或目录：

```text
AGENTS.md
.agents/
.codex/
```

不要直接覆盖。改用临时目录对比：

```powershell
$staging = "E:\__Code\__Work\your-new-project\.ai-template-staging"
New-Item -ItemType Directory -Force $staging | Out-Null
Copy-Item -Path "$templateRoot\*" -Destination $staging -Recurse -Force
```

然后手动合并：

- 目标已有 `AGENTS.md`：保留原项目硬规则，补充模板中的分层和 Skill 路由。
- 目标已有 `.agents/skills`：避免 Skill 同名冲突，必要时重命名模板 Skill。
- 目标已有 `.codex/config.toml`：只合并官方 runtime 设置，不覆盖认证、provider、MCP、企业策略。

## 4. 必须替换的占位符

在目标项目根目录执行：

```powershell
rg "\{\{[A-Z0-9_]+\}\}"
```

常见占位符：

| 占位符 | 含义 | 示例 |
| --- | --- | --- |
| `{{PROJECT_NAME}}` | 项目名称 | `BillingService` |
| `{{REPOSITORY_NAME}}` | 仓库名 | `acme/billing-service` |
| `{{CURRENT_BRANCH_OR_REF}}` | 当前基线分支或主分支 | `main` |
| `{{CURRENT_PHASE}}` | 当前阶段 | `R2 hardening` |
| `{{PROJECT_DOMAIN}}` | 项目领域 | `industrial slicing software` |
| `{{MAIN_APP_PATH}}` | 当前实现主线 | `src` |
| `{{DEPRECATED_PATHS}}` | 废弃或历史路径 | `legacy` |
| `{{TEST_PATH}}` | 测试目录 | `tests` |
| `{{TECH_STACK}}` | 技术栈 | `C++20, Qt 5.15, CMake` |
| `{{BUILD_SYSTEM}}` | 构建系统 | `CMake` |
| `{{DEFAULT_LANGUAGE}}` | 默认回复语言 | `zh-CN` |
| `{{BUILD_COMMAND}}` | 构建命令 | `cmake --build build` |
| `{{TEST_COMMAND}}` | 测试命令 | `ctest --test-dir build` |
| `{{QUICK_VALIDATION_COMMAND}}` | 快速验证命令 | `scripts/run_ci_quick.ps1` |
| `{{UI_SMOKE_COMMAND}}` | UI smoke 命令 | `app.exe --self-test` |
| `{{INTEGRATION_TEST_COMMAND}}` | 集成验证命令 | `scripts/run_integration.ps1` |
| `{{BUILD_TOOL}}` | 构建工具 | `CMake` |
| `{{PACKAGE_MANAGER}}` | 包管理 | `vcpkg` |
| `{{ENGINEERING_ROLE}}` | 工程视角 | `senior C++/Qt architect` |
| `{{FORMAL_DOCS_PATH}}` | 当前正式 PRD/DEV/DOC 目录 | `docs/slice` |
| `{{CODEX_TASK_PATH}}` | Codex 任务卡目录 | `docs/codex_task` |
| `{{ARCHIVE_DOCS_PATH}}` | 历史归档目录 | `docs/archive` |
| `{{CURRENT_TASK_FILE}}` | 当前任务卡入口 | `docs/codex_task/current/TASKS_current.md` |
| `{{AI_WORKSPACE_PATH}}` | AI 会话归档目录 | `ai_workspace` |

重点文件：

- `AGENTS.md`
- `.agents/docs/build-and-test.md`
- `.agents/docs/code-standards.md`
- `.agents/docs/doc-state.md`
- `.codex/project-context.toml`
- `.codex/agents/*.toml`

## 5. Skill 重命名策略

模板默认 Skill 名称是通用名：

```text
project-dev-workflow
project-code-review
project-architecture-guardrails
project-build
project-context-handoff
project-doc-state-resolver
project-chat-save
```

单个个人项目可以保留。

如果同一台电脑上会打开多个项目，推荐改成项目名前缀：

```text
billing-dev-workflow
billing-code-review
billing-architecture-guardrails
billing-build
billing-context-handoff
billing-doc-state-resolver
billing-chat-save
```

需要同步修改：

1. `.agents/skills/<skill-folder>/SKILL.md` 的 `name`。
2. `.agents/skills/<skill-folder>/SKILL.md` 的 `description`。
3. `AGENTS.md` 中的 Skill routing。
4. `.agents/docs/README.md` 或 workflow map 中的 Skill 名称。

注意：不要让两个不同目录的 Skill 使用同一个 `name`。Codex 不会自动合并同名 Skill，容易造成选择混乱。

## 6. `.codex` 配置使用原则

`.codex/config.toml` 只放 Codex runtime 设置，例如：

```toml
approval_policy = "on-request"
sandbox_mode = "workspace-write"
web_search = "cached"

[features]
hooks = false
multi_agent = true

[agents]
max_threads = 2
max_depth = 1
```

不要在项目级 `.codex/config.toml` 中放：

- API key；
- 用户认证；
- provider 凭据；
- 个人路径；
- 企业安全策略；
- 大段项目事实。

项目事实应放：

- `AGENTS.md`
- `.agents/docs/**`
- `{{FORMAL_DOCS_PATH}}`
- `{{CODEX_TASK_PATH}}`
- `.codex/project-context.toml`

## 7. 代码风格与提交风格迁移

代码风格入口：

```text
.agents/docs/code-standards.md
```

迁移时要按目标项目替换：

- 命名规则；
- 注释规则；
- public API 文档格式；
- UI / framework 线程规则；
- external SDK / API / ABI 边界；
- 默认单位；
- 错误处理和日志规则。

提交风格入口：

```text
.codex/project-context.toml
```

建议添加：

```toml
[commit_style]
format = "type(scope): 中文说明"
examples = [
  "docs(codex): 增加AI协作规则与模板",
  "fix(slice): 收口通道化输出文件命名规则",
]
```

如果目标项目使用英文提交，就改成目标项目现有风格。原则是：模板服从目标仓库历史风格。

## 8. 校验导出结果

在目标项目根目录执行：

```powershell
rg "\{\{[A-Z0-9_]+\}\}"
```

确认没有未替换占位符。

校验 TOML：

```powershell
@'
import pathlib, tomllib
for path in pathlib.Path(".codex").rglob("*.toml"):
    with path.open("rb") as f:
        tomllib.load(f)
    print(f"OK {path}")
'@ | python -
```

校验 custom agents：

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

检查 Skill front matter：

```powershell
Get-ChildItem .agents\skills -Recurse -Filter SKILL.md |
  ForEach-Object {
    Write-Host "---- $($_.FullName)"
    Get-Content $_.FullName -TotalCount 8
  }
```

## 9. 首次在新项目中让 Codex 自检

在目标项目打开 Codex 后，建议第一条提示：

```text
请读取 AGENTS.md、.agents/docs/README.md、.codex/README.md、.codex/project-context.toml、
{{FORMAL_DOCS_PATH}} 和 {{CODEX_TASK_PATH}}，
总结当前项目 AI 协作规则、Skill 路由、代码风格、验证命令、当前任务入口，并列出仍未替换的占位符。
暂时不要修改业务代码。
```

如果是 C++ / Qt / 硬件 / SDK 项目，可以追加：

```text
请重点检查 .agents/docs/code-standards.md 是否已经覆盖 C++/Qt 命名、Doxygen、线程、SDK ABI、错误处理和提交风格。
```

## 10. 建议的提交方式

在目标项目完成替换和校验后，单独提交 AI 协作配置：

```powershell
git add AGENTS.md .agents .codex
git commit -m "docs(codex): 增加AI协作规则与模板"
```

如果目标项目不是中文提交风格，按目标仓库历史风格调整。

不要把业务代码改动和模板导入混在同一个提交里。

## 11. 后续维护

推荐维护节奏：

1. 先在 1-2 个小项目试用模板。
2. 记录哪些占位符重复替换、哪些规则总是要改。
3. 再考虑增加 PowerShell 初始化脚本。
4. 当需要跨机器、跨团队安装时，再考虑做成 Codex plugin。

当前阶段推荐保持为“可复制模板”，不要过早插件化。

## 12. 常见错误

- 复制当前项目 `.agents` / `.codex`，而不是复制模板目录。
- 忘记替换 `{{...}}` 占位符。
- 多个项目 Skill 使用同一个 `name`。
- 把 API key、个人路径或认证配置写入项目 `.codex/config.toml`。
- 在 hooks 未审查前启用 hooks。
- 把长项目手册塞进 `AGENTS.md`。
- 让 `.codex/project-context.toml` 承担 runtime config 职责。
- 业务代码修改和模板导入混在一个提交里。
