# LedgerTwo 文档索引与 AI 实现阅读顺序

> 当前事实源提示：Task01-Task49 已完成。Task49X 核心代码、运行开关、本机 schema 19、微信 XLSX/支付宝 CSV 真实 preview、移动端视觉验收和 NAS staging 自动回滚脚本已完成；支付宝当前仍只导出 CSV，不再等待支付宝 XLSX。剩余发布门禁为 NAS schema 19 staging、production 一致性备份与逐批导入确认。后续优先读取 `docs/project_analysis/2026-07-13-task49x-nas-schema19-readiness.md`、`docs/project_analysis/2026-07-12-local-wsl-xlsx-csv-preview-acceptance.md`、`docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`、专项 PRD/DEV 和既有 v1.2 导入契约。
>
> Fresh Light 设计规范、本地 29 Frame 审阅包和协同开发计划已完成；UI-FL-01 至 UI-FL-10 于 2026-07-15 全部关闭。Task53 分类、标签与导入智能归类专项已完成 P1-P6 产品、技术、API、Migration、Fixture、UI/Figma 和详细任务细化，其代码、migration 022 和部署暂不启动。Task50.1、Task50.2、Task50.3A 与 Task50.3B 已完成；成员增删/调角、Owner 原子移交、成员离开、ETag、历史参与者名称和历史余额已通过自动化验收，下一实现任务为 Task50.3C。WSL staging 仍为 schema 19，NAS 未执行 migration 020/021。

本文档用于让人类开发者、Codex、Cursor、Copilot 或其他 AI 编码模型快速理解项目并按正确顺序实现代码。

## 1. 当前结论

### 1.1 Task30 后当前结论

当前项目已经具备 v1.2 发布候选能力。后续重点不是继续堆功能，而是：

1. 保持 v1.2 业务范围冻结。
2. 按 `docs/releases/` 执行发布候选检查与升级验收。
3. 在部署窗口同步本机验收版本到 NAS。
4. 处理阻断级缺陷和非阻断性能债务。
5. v1.3 开工前重新冻结多账本、多成员和多人分摊范围。
6. v1.2 发布收口后按 UI-FL-01 至 UI-FL-10 分阶段迁移 Fresh Light，不覆盖既有业务验收事实。

当前产品与开发规划入口：

```text
docs/README.md
docs/prd/README.md
docs/prd/00-product-roadmap.md
docs/prd/20-product-retrospective-and-positioning.md
docs/prd/21-roadmap-short-mid-long.md
docs/prd/22-prd-v1.1-trust-and-daily-use.md
docs/prd/23-feature-priority-and-deferral-decisions.md
docs/prd/24-short-mid-module-breakdown.md
docs/prd/25-prd-v1.1-module-specs.md
docs/prd/26-prd-v1.2-import-module-specs.md
docs/prd/29-prd-v1.2-module-business-service-breakdown.md
docs/prd/30-prd-v1.2-xlsx-import-special.md
docs/prd/31-prd-v1.3-multi-ledger.md
docs/prd/32-v1.3-task50-acceptance-fixtures.md
docs/prd/33-task51-scenario-evidence-and-scope-questions.md
docs/prd/34-prd-v1.3-category-tag-intelligence.md
docs/prd/27-acceptance-case-matrix.md
docs/prd/28-transaction-caliber-supplement.md
docs/tech/README.md
docs/tech/00-current-architecture-after-task30.md
docs/tech/18-short-mid-architecture-slices.md
docs/tech/19-short-mid-implementation-readiness.md
docs/tech/20-v1.2-import-implementation-contract.md
docs/tech/21-v1.2-import-migration-review.md
docs/tech/22-v1.2-import-task47-implementation-plan.md
docs/tech/23-v1.2-deployment-environment-isolation.md
docs/tech/24-v1.2-xlsx-import-implementation-plan.md
docs/tech/26-v1.3-category-tag-intelligence-contract.md
docs/tech/27-v1.3-category-tag-migration-review.md
docs/api/API_INVENTORY.md
docs/api/API_CONVENTIONS.md
docs/api/openapi.yaml
docs/api/openapi-v1.2-import-draft.yaml
docs/api/openapi-v1.3-ledger-draft.yaml
docs/api/openapi-v1.3-category-tag-draft.yaml
docs/fixtures/imports/README.md
docs/fixtures/category-tag/README.md
docs/ui/README.md
docs/ui/14-v1.1-v1.2-module-flows.md
docs/ui/15-ledgertwo-ux-optimization-program.md
docs/ui/16-v1.3-multi-ledger-flows.md
docs/ui/17-v1.3-category-tag-intelligence-flows.md
docs/ui/figma/README.md
docs/ui/figma/ledger-two-fresh-light-implementation-spec-2026-07-13.md
docs/ui/figma/task50-v1.3-multi-ledger/README.md
docs/ui/figma/task50-v1.3-multi-ledger/task50-frame-manifest.json
docs/ui/figma/task53-v1.3-category-tag/README.md
docs/ui/figma/task53-v1.3-category-tag/task53-frame-manifest.json
docs/codex_tasks/05-foundation-task-plan.md
docs/codex_tasks/08-product-roadmap-dev-plan.md
docs/codex_tasks/09-task41-49-detailed-plan.md
docs/codex_tasks/10-task33-40-detailed-plan.md
docs/codex_tasks/11-v1.2-release-hardening-plan.md
docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md
docs/codex_tasks/13-fresh-light-ui-interaction-plan.md
docs/codex_tasks/14-v1.3-task50-predevelopment-plan.md
docs/codex_tasks/15-v1.3-task50-detailed-implementation-plan.md
docs/codex_tasks/16-v1.3-task50-3-readiness-and-post-task50-entry.md
docs/codex_tasks/17-task51-predevelopment-plan.md
docs/codex_tasks/18-task53-category-tag-predevelopment-plan.md
docs/codex_tasks/19-v1.3-task53-detailed-implementation-plan.md
docs/project_analysis/2026-07-16-category-tag-competitive-research.md
docs/project_analysis/2026-07-16-task53-predevelopment-readiness.md
docs/project_analysis/2026-07-15-task50-current-state-gap-matrix.md
docs/project_analysis/2026-07-15-task50-p6-development-readiness.md
docs/project_analysis/2026-07-16-task50-2-ledger-context-acceptance.md
docs/project_analysis/2026-07-16-task50-3a-lifecycle-acceptance.md
docs/project_analysis/2026-07-16-task50-3b-member-acceptance.md
```

### 1.2 历史 v0.3 结论

v0.2 文档已经足够支持产品讨论、UI 原型和总体技术选型，但对于“直接交给 AI 模型连续生成可运行代码”还不够。主要缺口是：

1. 缺少明确的 MVP 裁剪边界，AI 容易一次性实现过多功能。
2. 缺少按模块的后端 handler / service / repository / migration 实现细则。
3. 缺少统一 DTO、错误码、校验规则和权限矩阵。
4. 缺少前端页面级组件树、表单字段、API 对接策略和响应式规则。
5. 缺少可执行测试用例和验收命令。
6. 缺少 AI 编码任务切片和提示词模板。
7. Mac Air 与 Windows PC 双开发环境的配置步骤不够细。

v0.3 文档包补齐以上缺口。Demo 版本按 v0.3 执行，AI 模型可以分模块完成代码编写，但仍建议每个阶段由人类开发者 review 数据库迁移、金额计算和权限控制。

## 2. 推荐阅读顺序

AI 编码模型必须按以下顺序阅读：

```text
00_DOCUMENT_INDEX.md
README.md
docs/README.md
docs/prd/README.md
docs/tech/README.md
docs/ui/README.md
docs/releases/README.md
docs/releases/v1.2.0-rc-release-notes.md
docs/releases/v1.2.0-rc-upgrade-guide.md
docs/releases/v1.2.0-rc-checklist.md
docs/project_analysis/2026-07-12-v1.2-nas-production-upgrade-acceptance.md
docs/project_analysis/2026-07-13-task49x-nas-schema19-readiness.md
docs/project_analysis/2026-07-12-local-wsl-xlsx-csv-preview-acceptance.md
docs/project_analysis/v1.2-task49x-ui-acceptance-2026-07-12/README.md
docs/project_analysis/2026-07-12-real-wechat-bill-import-readiness.md
docs/project_analysis/2026-07-12-v1.2-xlsx-special-predevelopment-review.md
docs/prd/00-product-roadmap.md
docs/prd/20-product-retrospective-and-positioning.md
docs/prd/21-roadmap-short-mid-long.md
docs/prd/22-prd-v1.1-trust-and-daily-use.md
docs/prd/23-feature-priority-and-deferral-decisions.md
docs/prd/24-short-mid-module-breakdown.md
docs/prd/25-prd-v1.1-module-specs.md
docs/prd/26-prd-v1.2-import-module-specs.md
docs/prd/29-prd-v1.2-module-business-service-breakdown.md
docs/prd/30-prd-v1.2-xlsx-import-special.md
docs/prd/31-prd-v1.3-multi-ledger.md
docs/prd/32-v1.3-task50-acceptance-fixtures.md
docs/prd/27-acceptance-case-matrix.md
docs/prd/28-transaction-caliber-supplement.md
docs/tech/18-short-mid-architecture-slices.md
docs/tech/19-short-mid-implementation-readiness.md
docs/tech/20-v1.2-import-implementation-contract.md
docs/tech/21-v1.2-import-migration-review.md
docs/tech/22-v1.2-import-task47-implementation-plan.md
docs/tech/24-v1.2-xlsx-import-implementation-plan.md
docs/ui/14-v1.1-v1.2-module-flows.md
docs/ui/15-ledgertwo-ux-optimization-program.md
docs/ui/figma/README.md
docs/ui/figma/ledger-two-fresh-light-implementation-spec-2026-07-13.md
docs/fixtures/imports/README.md
docs/codex_tasks/README.md
docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md
docs/codex_tasks/13-fresh-light-ui-interaction-plan.md
docs/codex_tasks/14-v1.3-task50-predevelopment-plan.md
docs/project_analysis/2026-07-15-task50-current-state-gap-matrix.md
```

如果处理早期 Demo 或 v0.3 任务，再补读：

```text
01_PRD.md
02_UI_INTERACTION_DESIGN.md
03_TECH_DESIGN.md
04_TECH_IMPLEMENTATION.md
07_DATABASE_API.md
13_DEMO_SCOPE_LOCK.md
14_BACKEND_MODULE_SPEC.md
15_FRONTEND_MODULE_SPEC.md
16_TEST_ACCEPTANCE_SPEC.md
17_AI_CODING_TASKS.md
09_DEV_ENV_MAC.md 或 10_DEV_ENV_WINDOWS.md
11_VSCODE_CODEX_WORKFLOW.md
```

## 3. 文档清单

| 文件 | 作用 |
|---|---|
| `01_PRD.md` | 产品需求、角色、页面、功能、验收 |
| `02_UI_INTERACTION_DESIGN.md` | UI 交互、桌面/移动端页面、交互状态 |
| `03_TECH_DESIGN.md` | 架构、选型、模块边界、安全、跨端预留 |
| `04_TECH_IMPLEMENTATION.md` | 后端/前端实现路线、工程结构、运行方式 |
| `05_FRONTEND_DESIGN.md` | 前端页面、组件、状态、表单、样式方案 |
| `06_NAS_DEPLOYMENT.md` | 群晖 NAS Docker 部署、备份、恢复 |
| `07_DATABASE_API.md` | 数据库 schema、索引、API 合同 |
| `08_MVP_ROADMAP.md` | 里程碑、版本计划、开发顺序 |
| `09_DEV_ENV_MAC.md` | Mac Air 开发环境详细配置 |
| `10_DEV_ENV_WINDOWS.md` | Windows PC + WSL2 开发环境详细配置 |
| `11_VSCODE_CODEX_WORKFLOW.md` | VSCode + Codex 工作流 |
| `12_LOCAL_UPLOAD_GUIDE.md` | 本地上传到 GitHub 仓库说明 |
| `13_DEMO_SCOPE_LOCK.md` | Demo 版本范围锁定，防止 AI 过度实现 |
| `14_BACKEND_MODULE_SPEC.md` | 后端模块级实现规格，AI 可直接按模块编码 |
| `15_FRONTEND_MODULE_SPEC.md` | 前端页面/组件级实现规格，AI 可直接按页面编码 |
| `16_TEST_ACCEPTANCE_SPEC.md` | 自动化测试、手工验收、核心业务测试用例 |
| `17_AI_CODING_TASKS.md` | AI 编码任务拆分与提示词模板 |

## 4. AI 实现约束

AI 编码模型必须遵守：

1. 当前阶段优先遵守 Task30 后 Foundation before v1.1 文档；早期 Demo 范围锁定作为历史约束，不得用于删除已经完成的新能力。
2. 金额全部用整数分 `amount_cent` / `amount_cents`，禁止 float。
3. 后端采用 Go + SQLite + REST JSON。
4. 前端采用 React + TypeScript + Vite。
5. 删除账单必须软删除。
6. 共同支出必须生成 split 记录。
7. 结算必须生成 settlement 记录，不允许直接修改历史账单抵消金额。
8. private 账单对方不可见。
9. 业务逻辑放 service 层，不要塞进 handler。
10. 所有金额修改写 audit log。

## 5. 推荐开发分支

```bash
git checkout -b docs/ai-ready-v0.3
```

或直接提交到 main：

```bash
git add .
git commit -m "docs: add AI-ready implementation specs and dev setup"
git push origin main
```
