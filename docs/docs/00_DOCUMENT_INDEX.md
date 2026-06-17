# LedgerTwo 文档索引与 AI 实现阅读顺序（Post-Task30 Foundation）

版本：Foundation before v1.1  
状态：供审核  
适用范围：Task01-Task30 已完成后，进入 v1.1 业务开发前的基础框架补齐阶段

## 1. 当前结论

LedgerTwo 已完成 `docs/18_POST_DEMO_AI_CODING_TASKS.md` 中 Task01-Task30 的开发任务，项目已经从 Demo/MVP 演进到 v1.0 可部署阶段。后续不能继续沿用早期“固定双人 Demo”作为唯一事实源，否则 Codex/Gemini 会继续读取到过期约束，导致新任务开发方向混乱。

当前阶段的关键结论：

1. v1.1 具体业务规划尚未完成产品审核，不能直接进入 v1.1 业务功能开发。
2. 进入 v1.1 前，应先补足 MVP/v1.0 阶段缺失的基础框架，包括配置安全、文档事实源、账本上下文、权限策略、API 契约、分类标签账户管理、前端账本状态、迁移测试和 AI 开发规范。
3. 后续 Codex/Gemini 开发必须以 `docs/codex_tasks/` 为任务入口，并先阅读本索引、当前 PRD、当前技术文档和任务文件。
4. 所有开发任务必须小步提交，禁止一次性重构全项目。

## 2. 文档分层

```text
docs/
  prd/           产品需求、阶段路线、验收标准
  tech/          技术架构、模块边界、API、数据、安全、测试
  ui/            页面结构、交互、响应式、空态/错态/确认态
  codex_tasks/   Codex/Gemini 编码任务、代码风格、质量门禁、提示词模板
  reviews/       当前实现与文档差异审阅
```

## 3. 推荐阅读顺序

### 3.1 人类产品/架构审核

```text
README.md
CHANGELOG.md
docs/reviews/2026-06-17-task30-current-progress-vs-docs-review.md
docs/prd/00-product-roadmap.md
docs/prd/11-foundation-framework-before-v1.1.md
docs/tech/00-current-architecture-after-task30.md
docs/tech/13-foundation-framework-before-v1.1.md
docs/ui/12-foundation-framework-ui.md
```

### 3.2 Codex/Gemini 开发任务阅读顺序

```text
docs/00_DOCUMENT_INDEX.md
docs/prd/README.md
docs/prd/00-product-roadmap.md
docs/prd/11-foundation-framework-before-v1.1.md
docs/tech/README.md
docs/tech/13-foundation-framework-before-v1.1.md
docs/codex_tasks/README.md
docs/codex_tasks/00-ai-development-workflow.md
docs/codex_tasks/01-repository-code-style.md
docs/codex_tasks/04-testing-quality-gates.md
docs/codex_tasks/05-foundation-task-plan.md
```

如果任务涉及后端，必须额外阅读：

```text
docs/tech/14-configuration-security-deployment.md
docs/tech/15-ledger-context-rbac.md
docs/tech/16-api-contract-openapi-error.md
docs/tech/17-data-migration-test-quality.md
docs/codex_tasks/02-backend-go-style.md
```

如果任务涉及前端，必须额外阅读：

```text
docs/ui/12-foundation-framework-ui.md
docs/ui/13-settings-management-redesign.md
docs/codex_tasks/03-frontend-react-ts-style.md
```

## 4. 当前阶段开发约束

1. v1.1 具体业务需求尚未审核通过，不允许实现未经确认的业务闭环。
2. 允许补足 v1.1 前置基础框架，例如 LedgerContext、RBAC、配置安全、OpenAPI、分类/标签管理基础 API、前端 query key 规范和测试门禁。
3. 不允许删除或破坏 v1.0 已完成的记账、分摊、结算、导入、备份、恢复、统计、PWA/离线草稿能力。
4. 所有金额继续使用整数分 `int64 cents`，前端展示为元。
5. 后端是结算、统计和权限判断的唯一可信来源。
6. private 账单不得被无权限成员通过 API、导出、附件 URL 或缓存泄露。
7. 所有高风险操作必须有审计日志或二次确认。
8. 不允许提交真实 `.env`、数据库、备份、上传文件和密钥。

## 5. 分支建议

本轮仅为文档和基础框架准备，建议分支名：

```bash
docs/foundation-before-v1.1
```

如果后续开始写代码，建议每个 Foundation Task 单独分支：

```bash
foundation/task31-doc-alignment
foundation/task32-config-security
foundation/task33-ledger-context-rbac
foundation/task34-api-contract
foundation/task35-category-tag-account
```

## 6. AI 输出格式

Codex/Gemini 完成每个任务后必须输出：

```text
完成内容：
- ...

修改文件：
- ...

验证命令：
- ...

未完成/风险：
- ...

下一步建议：
- ...
```

如果未运行测试或构建，必须明确说明原因，不得声称“已验证”。
