# LedgerTwo 压缩文档包与 AI 模板接入分析

日期：2026-07-02

## 1. 本次处理范围

输入文件：

- `docs/ledger_two_foundation_docs_2026-06-17.zip`
- `docs/ledger_two_ai_ready_upload_pack_v0.3.zip`
- `docs/ledger_two_local_upload_pack.zip`
- `ai-project-template/`

已解压到：

```text
docs/project_analysis/extracted_archives/
  ledger_two_foundation_docs_2026-06-17/
  ledger_two_ai_ready_upload_pack_v0.3/
  ledger_two_local_upload_pack/
```

## 2. 三个压缩包与当前 docs 的冲突判断

### 2.1 ledger_two_local_upload_pack.zip

结论：只适合作为历史初始化包，不建议覆盖当前仓库。

原因：

- 包内主要是早期根目录上传包，包含 `README.md`、`AGENTS.md`、`docker-compose.yml`、`docs/01-12`、基础 `.vscode/.codex/deploy` 等。
- 当前仓库已经完成 Task30 后演进，根 README、部署、AI 规则和 docs 内容都比该包更新。
- 哈希比对显示同名文件均与当前仓库不同；如果直接覆盖，会回退当前文档和配置。

可用价值：

- 可作为最早期项目骨架和上传路径参考。
- 不应作为后续开发计划来源。

### 2.2 ledger_two_ai_ready_upload_pack_v0.3.zip

结论：是 v0.3 AI-ready 实现包，和当前 `docs/00-17` 高度重叠，不建议覆盖当前仓库。

原因：

- 包内包含 `docs/00_DOCUMENT_INDEX.md` 到 `docs/17_AI_CODING_TASKS.md`，对应早期 Demo/AI 编码任务体系。
- 当前仓库已有这些文件，并额外增加了模块化 `docs/prd`、`docs/tech`、`docs/ui`、Task30 后文档和 v1.0 状态。
- 包内根 `README.md`、`AGENTS.md`、`docker-compose.yml`、`.env.example` 与当前代码阶段不一致。

可用价值：

- 可作为 Demo 阶段需求边界和 AI 任务切片的历史依据。
- 对当前后续开发只能提供背景，不能作为事实源。

### 2.3 ledger_two_foundation_docs_2026-06-17.zip

结论：可以作为后续开发计划补充，尤其适合 Task30 后进入 Foundation before v1.1 的文档事实源。

原因：

- 包内明确说明适用阶段是 Task01-Task30 已完成、v1.1 业务需求未审核前。
- 它不是早期 Demo 包，而是对 Task30 后当前代码与 PRD/Tech/UI 差异的审阅和基础框架补齐计划。
- 新增 `docs/codex_tasks/`、Foundation PRD、Foundation Tech、Foundation UI、Task31-Task40 计划，正好补齐当前仓库后续开发入口。

已安全并入的新增内容：

```text
docs/codex_tasks/
docs/reviews/2026-06-17-task30-current-progress-vs-docs-review.md
docs/prd/11-foundation-framework-before-v1.1.md
docs/prd/12-current-progress-gap-analysis.md
docs/tech/00-current-architecture-after-task30.md
docs/tech/13-foundation-framework-before-v1.1.md
docs/tech/14-configuration-security-deployment.md
docs/tech/15-ledger-context-rbac.md
docs/tech/16-api-contract-openapi-error.md
docs/tech/17-data-migration-test-quality.md
docs/ui/12-foundation-framework-ui.md
docs/ui/13-settings-management-redesign.md
```

未直接覆盖的同名文件：

- `docs/00_DOCUMENT_INDEX.md`
- `docs/18_POST_DEMO_AI_CODING_TASKS.md`
- `docs/prd/00-product-roadmap.md`
- `docs/prd/README.md`
- `docs/tech/README.md`
- `docs/ui/README.md`

这些文件存在事实源冲突，后续建议作为 Task31 独立处理，避免一次性覆盖造成上下文丢失。

## 3. 当前 docs 是否形成完整 PRD / DEV 体系

结论：当前已经具备较完整的 PRD / Tech / UI / AI task 文档体系，但仍需要 Task31 做事实源收口。

已有体系：

- 总体文档：`docs/00_DOCUMENT_INDEX.md`、`docs/README.md`
- 原始主文档：`docs/01_PRD.md` 到 `docs/18_POST_DEMO_AI_CODING_TASKS.md`
- 模块化 PRD：`docs/prd/*`
- 模块化技术文档：`docs/tech/*`
- 模块化 UI 文档：`docs/ui/*`
- 审阅报告：`docs/reviews/*`
- 后续 DEV / AI 任务入口：`docs/codex_tasks/*`

主要缺口：

1. 旧文档仍有 Demo/v0.3/v0.2 表述，和 README/CHANGELOG 的 v1.0 状态不完全一致。
2. 当前没有传统命名为 `docs/dev` 或 `DEV_xx` 的目录，但 `docs/codex_tasks/` 已经承担后续 DEV 任务卡功能。
3. 后续开发前应优先执行 `docs/codex_tasks/05-foundation-task-plan.md` 中的 Task31：文档事实源收口。

判断：

- 可以支持后续开发。
- 不建议继续只依赖早期 `17_AI_CODING_TASKS.md`。
- 后续新任务应以 `docs/codex_tasks/` + 对应 PRD/Tech/UI 文档作为入口。

## 4. ai-project-template 接入结果

已接入：

```text
.agents/
.codex/agents/
.codex/hooks/
.codex/project-context.toml
.codex/README.md
```

已项目化替换：

- 项目名：LedgerTwo
- 当前阶段：Foundation before v1.1，Task01-Task30 已完成
- 正式文档入口：`docs`
- 当前任务入口：`docs/codex_tasks/05-foundation-task-plan.md`
- 历史归档：`docs/project_analysis/extracted_archives` 和 `ai_workspace`
- 技术栈：Go、SQLite、React、TypeScript、Vite、TanStack Query、Zustand、Docker Compose
- 验证命令：`./run_tests.sh`

已更新：

- `AGENTS.md`：加入 `.agents/docs`、`docs/codex_tasks` 和 Task30 后事实源说明。
- `docs/README.md`：加入 `codex_tasks`、`project_analysis` 和 Foundation before v1.1 入口说明。
- `.codex/config.toml`：保留 `approval_policy=on-request`、`sandbox_mode=workspace-write`，补充多 agent 配置。

验证：

- `.codex/*.toml` 通过 Python `tomllib` 解析。
- `.codex/agents/*.toml` 均具备 `name`、`description`、`developer_instructions`。
- `.agents` / `.codex` / 新增 docs 中已清空模板占位符。

## 5. 后续建议

P0：

1. 执行 Task31：文档事实源收口。
2. 不覆盖旧包中的早期 README、AGENTS、docker-compose 或 docs。
3. 将 `docs/project_analysis/extracted_archives` 视为历史证据，不作为当前事实源。

P1：

1. 执行 Task32：统一配置与部署安全，重点处理 `DB_DSN/DB_PATH`、`JWT_SECRET/SESSION_SECRET` 口径。
2. 执行 Task33：LedgerContext 与 RBAC。
3. 执行 Task34：API 契约与 OpenAPI 草案。

P2：

1. 逐步拆分过大的交易和前端表单模块。
2. 强化多账本隔离、权限矩阵、附件访问控制、迁移回归测试。
