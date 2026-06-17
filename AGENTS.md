# LedgerTwo AI Coding Rules

本文档是 LedgerTwo 仓库中 Codex、Gemini、Cursor、Copilot 等 AI 编码助手的根级工作规则。

AI 助手在修改本仓库代码前，必须先阅读本文件，并遵守本文档中的阶段边界、代码规范、任务执行流程和安全要求。

---

## 1. Project Status

LedgerTwo 已完成 `docs/18_POST_DEMO_AI_CODING_TASKS.md` 中 Task01-Task30 的开发任务，当前仓库处于：

```text
v1.0 / post-MVP / Task30 completed
```

当前阶段不是直接开发 v1.1 业务功能，而是：

```text
Foundation before v1.1
```

也就是在 v1.1 正式业务开发前，先补足 MVP 阶段缺失或不足的基础框架，包括但不限于：

- 文档事实源收口
- 配置与部署安全
- LedgerContext
- RBAC / 权限框架
- API 契约与 OpenAPI
- 分类、标签、支付账户基础管理框架
- 前端 LedgerProvider 与 Query Key 规范
- 设置页信息架构
- 测试与质量门禁
- 附件访问控制
- 审计与系统诊断

v1.1 具体业务功能尚未最终审核，不得擅自实现。

---

## 2. Read First

AI 助手在开始任何代码修改前，必须按顺序阅读：

1. `AGENTS.md`
2. `docs/00_DOCUMENT_INDEX.md`
3. `docs/prd/README.md`
4. `docs/prd/00-product-roadmap.md`
5. `docs/prd/11-foundation-framework-before-v1.1.md`
6. `docs/tech/README.md`
7. `docs/tech/13-foundation-framework-before-v1.1.md`
8. `docs/ui/README.md`
9. `docs/codex_tasks/README.md`
10. `docs/codex_tasks/00-ai-development-workflow.md`
11. `docs/codex_tasks/05-foundation-task-plan.md`
12. `docs/codex_tasks/06-review-checklist.md`

如果当前任务指定了额外 PRD、Tech、UI 或 Review 文档，必须一并阅读。

---

## 3. Current Task Scope

当前允许执行的任务范围是：

```text
Foundation Task31-Task40
```

任务入口为：

```text
docs/codex_tasks/05-foundation-task-plan.md
```

推荐执行顺序：

1. Task31：文档事实源收口
2. Task32：配置与部署安全框架
3. Task33：LedgerContext 与 RBAC
4. Task34：API 契约与 OpenAPI
5. Task38：迁移、测试与质量门禁
6. Task36：前端 LedgerProvider 与 Query Key
7. Task35：分类、标签、支付账户管理基础
8. Task37：设置页信息架构重组
9. Task39：附件访问控制
10. Task40：审计与系统诊断中心

除非用户明确指定，不要跳过任务，也不要一次性实现多个任务。

---

## 4. Explicitly Not In Scope

在用户没有提供已审核 v1.1 PRD 和任务计划前，禁止实现以下功能：

- v1.1 具体业务功能
- 新的邀请机制完整业务
- 新的共同支付请求状态机
- 新的通知系统
- 新的预算系统
- 新的移动端 App
- 银行自动同步
- OCR 小票识别
- 股票、基金、资产负债管理
- 企业报销审批
- 复杂复式会计
- 公开注册或公开邀请链接
- 任意没有被当前任务明确要求的功能

如果当前基础框架任务需要为这些能力预留接口或数据结构，只能做清晰、最小、可测试的基础准备，不能直接实现完整业务流程。

---

## 5. Task Execution Rules

AI 助手执行任务时必须遵守：

1. 一次只执行一个任务。
2. 修改代码前，必须先输出实现计划和预计修改文件。
3. 等待用户确认后，再开始修改。
4. 不得扩大任务范围。
5. 不得重写整个项目。
6. 不得推翻当前技术栈。
7. 不得把未来 v1.1 业务混入基础框架任务。
8. 每次修改必须保持可审查、可回滚。
9. 必须补充或更新相关测试。
10. 必须运行当前任务相关验证命令。
11. 完成后必须输出完成内容、修改文件、验证命令、风险和下一步建议。

---

## 6. Repository Architecture Rules

LedgerTwo 当前主技术栈为：

- Backend：Go + SQLite + REST JSON
- Frontend：React + TypeScript + Vite
- State：TanStack Query + Zustand
- Form：React Hook Form + Zod
- Deploy：Docker Compose on NAS
- Database Migration：goose
- Storage：SQLite database + local backups + uploads directory

不得在没有明确任务要求的情况下替换核心技术栈。

---

## 7. Backend Rules

后端开发必须遵守：

1. 使用 Go。
2. 使用 SQLite 作为当前部署模型。
3. 金额必须使用整数分，禁止用 float 存储或计算持久化金额。
4. 使用 REST JSON API。
5. HTTP handler 只负责请求解析、响应输出和调用 service。
6. 业务规则必须放在 service 层。
7. 数据库访问必须放在 repository 层。
8. 不得在 handler 中直接写复杂 SQL。
9. 不得让前端成为权限判断的唯一来源。
10. 所有账本级业务 API 必须在后端校验 ledger membership。
11. 所有角色权限必须以后端 RBAC 为准。
12. 删除账单必须 soft delete。
13. 共同支出必须生成 split 记录。
14. 结算必须生成 settlement 记录，不得修改历史共同支出抵消金额。
15. 修改金额、删除账单、结算、导入、导出、备份、恢复等高风险操作必须写 audit log。
16. 不得把密码、JWT secret、session secret、数据库路径等敏感信息返回给前端。
17. 错误响应必须使用统一结构。
18. 数据库 migration 必须可追踪、可审查，不得随意修改已应用 migration。

---

## 8. Frontend Rules

前端开发必须遵守：

1. 使用 React + TypeScript + Vite。
2. 使用 TanStack Query 管理服务端状态。
3. 使用 Zustand 管理 UI 状态或本地轻量状态。
4. 使用 React Hook Form + Zod 管理表单和校验。
5. API 金额单位始终为分。
6. UI 展示金额单位为元。
7. 金额转换必须集中在工具函数中。
8. 账本级数据的 Query Key 必须包含 active ledger 信息。
9. 不得使用 `window.location.reload()` 作为常规状态刷新手段。
10. 切换账本应通过 LedgerProvider、状态更新和 query invalidation 完成。
11. 权限相关按钮可以在前端隐藏，但后端仍必须强制校验。
12. 所有主页面必须有 loading、empty、error 状态。
13. 高风险操作必须有二次确认。
14. 移动端页面不得出现横向滚动。
15. 不得为了快速实现而把大量业务逻辑写进页面组件。

---

## 9. Domain Rules

业务领域必须明确区分以下概念：

- Login User：登录用户账号
- Ledger：账本
- Ledger Member：账本成员
- Role：账本内角色
- Payer：实际付款人
- Participant：参与消费人
- Owner：账单归属人
- Creator：账单创建人
- Payment Account：支付账户，例如现金、微信、支付宝、银行卡
- Category：分类
- Tag：标签
- Split：分摊记录
- Settlement：结算记录
- Audit Log：审计日志

不得混用“用户账号”和“支付账户”。

不得混用“付款人”和“承担人”。

不得混用“可见性”和“是否参与结算”。

---

## 10. Ledger and Permission Rules

Foundation 阶段必须逐步收口以下规则：

1. 所有账本级数据必须带 ledger context。
2. 所有账本级 API 必须校验当前用户是否属于该账本。
3. owner 可以管理账本、成员、配置和高风险数据操作。
4. editor 可以记账和编辑自己有权限编辑的业务数据。
5. viewer 默认只读。
6. invited 或 pending invite 状态用户不得访问账本数据。
7. private 账单不得泄露给无权限成员。
8. shared 账单是否参与结算必须以后端业务规则为准。
9. 前端角色状态仅用于 UI 展示，不得作为最终权限依据。
10. 多账本切换时，不得 fallback 到错误账本。

---

## 11. Configuration and Security Rules

配置和安全必须遵守：

1. 不得提交真实 `.env`。
2. 不得提交真实数据库。
3. 不得提交备份文件。
4. 不得提交上传附件。
5. 生产环境不得使用开发默认 JWT/session secret。
6. 配置变量名称必须在代码、`.env.example`、`docker-compose.yml` 和文档中保持一致。
7. HTTP 局域网访问和 HTTPS 反向代理访问下的 Cookie 策略必须明确。
8. Cookie 必须使用 HttpOnly。
9. 生产公网或反向代理场景必须使用 HTTPS。
10. 备份、恢复、导出等高风险操作必须有审计日志。
11. 附件如果包含小票、截图、发票等隐私数据，不得通过无权限静态公开 URL 直接访问。

---

## 12. Database and Migration Rules

数据库变更必须遵守：

1. 使用 goose migration。
2. 新增字段必须说明默认值和兼容旧数据策略。
3. 不得随意修改已经被应用的 migration。
4. 破坏性 migration 必须有备份、回滚或人工恢复说明。
5. migration 前必须考虑 SQLite 兼容性。
6. 金额字段必须为 INTEGER。
7. 时间字段必须使用稳定格式。
8. soft delete 不能被物理删除替代。
9. v1.1 前的基础框架 migration 必须优先保障 v1.0 数据可升级。
10. 涉及权限、成员、账本、附件、分类、标签的 migration 必须补测试。

---

## 13. API Rules

API 必须遵守：

1. 使用 JSON 响应。
2. 成功响应使用统一结构。
3. 失败响应使用统一错误结构。
4. 错误码必须稳定。
5. 不得把内部异常堆栈返回给前端。
6. 分页、筛选、排序参数命名必须稳定。
7. 文件上传使用 multipart/form-data。
8. API DTO 不得直接暴露数据库内部结构。
9. 账本级 API 必须明确 ledger context。
10. 后续需要逐步补充 OpenAPI 规格。

推荐响应结构：

```json
{
  "success": true,
  "data": {}
}
```

错误响应结构：

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "参数错误",
    "details": {}
  }
}
```

---

## 14. Testing Rules

每个任务必须根据影响范围运行测试。

后端相关任务至少运行：

```bash
cd backend
go test ./...
```

前端相关任务至少运行：

```bash
cd frontend
pnpm lint
pnpm test
pnpm build
```

部署相关任务至少运行：

```bash
docker compose build
```

通用检查：

```bash
git diff --check
```

涉及以下内容时必须增加或更新测试：

- 配置加载
- 登录与认证
- LedgerContext
- RBAC
- 多账本隔离
- owner/editor/viewer 权限
- private/shared 可见性
- 分类、标签、账户管理
- 共同支出
- 分摊
- 结算
- 导入
- 导出
- 备份
- 恢复
- 附件访问
- migration

---

## 15. Code Style Rules

代码风格必须遵守：

1. Go 代码必须通过 `gofmt`。
2. Go 文件命名使用小写和下划线。
3. Go package 名称保持简短、语义明确。
4. Go service 方法必须表达业务含义。
5. SQL 不得随意拼接用户输入。
6. React 组件使用 PascalCase。
7. TypeScript 类型、接口和 DTO 必须清晰命名。
8. Hooks 只能在 React 组件或自定义 Hook 顶层调用。
9. Query Key 必须稳定、可预测。
10. 不得引入无必要的大型依赖。
11. 不得把格式化变更和业务变更混在一个提交中。
12. 提交信息建议使用 Conventional Commits。

提交信息示例：

```text
docs: align foundation framework before v1.1
fix: normalize production session secret config
refactor: introduce ledger context resolver
test: add ledger membership permission tests
feat: add category management foundation
```

---

## 16. Branch and Commit Rules

每个任务使用独立分支。

推荐分支命名：

```text
foundation/task32-config-security
foundation/task33-ledger-context-rbac
foundation/task34-api-contract
foundation/task35-category-tag-account
foundation/task36-frontend-ledger-provider
foundation/task37-settings-redesign
foundation/task38-test-quality
foundation/task39-attachment-access-control
foundation/task40-audit-diagnostics
```

禁止一个 PR 混入多个 Foundation 任务。

禁止在同一个 PR 中混入 v1.1 未审核业务功能。

---

## 17. Required Output Format

AI 助手完成任务后，必须按以下格式输出：

```text
完成内容：
- ...

修改文件：
- ...

验证命令：
- ...

验证结果：
- ...

未完成 / 风险：
- ...

下一步建议：
- ...
```

如果测试未运行，必须明确说明原因。

如果某项任务只完成部分内容，必须明确说明未完成范围。

---

## 18. Review Rules

提交 PR 前，必须检查：

1. 是否只实现当前任务。
2. 是否误实现 v1.1 业务功能。
3. 是否破坏 v1.0 现有能力。
4. 是否新增或更新测试。
5. 是否更新相关文档。
6. 是否有真实密钥、数据库、备份、上传文件。
7. 是否有权限绕过。
8. 是否有金额 float。
9. 是否有未处理错误。
10. 是否有不必要的大范围重构。
11. 是否可回滚。
12. 是否能被人类 reviewer 快速理解。

---

## 19. Human Approval Requirement

以下情况必须等待人类确认，不得自行决定：

1. 修改产品边界。
2. 实现 v1.1 具体业务功能。
3. 新增公开注册。
4. 新增公开邀请链接。
5. 改动认证方式。
6. 改动数据库核心结构。
7. 执行破坏性 migration。
8. 删除历史数据。
9. 改变结算算法口径。
10. 改变 private/shared 可见性规则。
11. 引入新框架或大型依赖。
12. 改变部署方式。

---

## 20. Final Reminder

当前阶段目标是：

```text
先补基础框架，再开发 v1.1 业务功能。
```

AI 助手不得因为看到 v1.1 相关草案或历史讨论，就自行开始 v1.1 功能实现。

一切 v1.1 业务开发必须等待用户提供明确、已审核的 PRD、Tech、UI 和任务计划。
