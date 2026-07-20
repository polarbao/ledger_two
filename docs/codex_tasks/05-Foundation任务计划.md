# Foundation Task31-Task40 任务计划

状态：供审核  
目标：在 v1.1 具体业务开发前，先补齐 MVP/v1.0 缺失的基础框架。

## Task31：文档事实源收口

```text
请先阅读：
- docs/00-文档索引.md
- docs/reviews/2026-06-17-Task30实现与文档差异审阅.md
- docs/prd/00-产品定位与版本路线.md
- docs/18-演示后AI编码任务与提示词.md

任务目标：
1. 更新 README、docs 索引、PRD roadmap、tech/ui README，使其统一为 Task30 已完成后的 Foundation before v1.1 状态。
2. 明确 v1.1 具体业务规划仍未审核。
3. 将后续 AI 任务入口指向 docs/codex_tasks/。
4. 不修改业务代码。

禁止事项：
- 不要实现 v1.1 业务功能。
- 不要删除历史文档，只能归档、改入口或说明过期状态。

验收标准：
- 文档不再互相冲突。
- AI 阅读顺序明确。
- git diff 只包含文档。
```

## Task32：配置与部署安全框架

```text
请先阅读：
- docs/tech/14-配置安全与部署一致性.md
- docs/tech/08-NAS部署方案.md
- docker-compose.yml
- backend/internal/config/config.go

任务目标：
1. 统一环境变量命名。
2. production 环境缺少强 JWT/Session 密钥时拒绝启动。
3. 明确 COOKIE_SECURE、COOKIE_SAMESITE 和 HTTP/HTTPS 模式。
4. 更新 .env.example、Docker Compose 和部署文档。
5. 增加配置校验测试。

禁止事项：
- 不要提交真实密钥。
- 不要破坏当前本地开发默认启动。

验收标准：
- development 可本地启动。
- production 缺少强密钥启动失败。
- Docker Compose 与 config.go 变量一致。
```

## Task33：LedgerContext 与 RBAC

```text
请先阅读：
- docs/tech/15-账本上下文与RBAC权限框架.md
- backend/internal/http/middleware/auth_middleware.go
- backend/internal/ledger/*
- backend/internal/transaction/service.go
- backend/internal/settlement/service.go

任务目标：
1. 新增统一 LedgerContext 解析。
2. 新增 RolePolicy / MembershipGuard。
3. 将关键业务 API 从自行 fallback ledger 逐步改为依赖 LedgerContext。
4. 增加 owner/editor/viewer 权限测试。

禁止事项：
- 不要实现邀请机制。
- 不要重写全部业务模块。
- 不要改变现有账单数据结构，除非文档明确。

验收标准：
- 非成员访问 ledger 返回 403/404。
- viewer 无法写入。
- 多账本数据隔离测试通过。
```

## Task34：API 契约与 OpenAPI

```text
请先阅读：
- docs/tech/16-API契约OpenAPI与错误码规范.md
- backend/internal/http/router/router.go
- frontend/src/api/client.ts

任务目标：
1. 整理当前 API contract。
2. 新增 docs/api/openapi.yaml 草案。
3. 明确 /api 与 /api/v1 的兼容策略。
4. 统一错误码和分页说明。
5. 前端 API 类型与文档字段对齐。

禁止事项：
- 不要破坏现有 /api 路径。
- 不要大范围改业务逻辑。

验收标准：
- OpenAPI 草案覆盖核心 API。
- 新增 API 必须写入文档。
```

## Task35：分类、标签、支付账户管理基础

```text
请先阅读：
- docs/prd/11-v1.1前基础框架补齐.md
- docs/ui/12-v1.1前基础界面框架.md
- docs/ui/13-设置页信息架构重组.md
- docs/tech/15-账本上下文与RBAC权限框架.md

任务目标：
1. 建立 category/tag/account 独立模块或清晰 service 边界。
2. 支持新增、编辑、排序、归档、恢复。
3. 历史账单继续显示已归档项。
4. 设置页增加对应管理入口。
5. 增加权限测试和 UI 状态。

禁止事项：
- 不要物理删除已使用分类/标签/账户。
- 不要实现复杂合并和批量迁移，除非另有任务。

验收标准：
- owner 可管理。
- viewer 不可管理。
- 归档后新增账单不默认展示。
- 历史账单展示不丢失。
```

## Task36：前端 LedgerProvider 与 Query Key

```text
请先阅读：
- docs/ui/12-v1.1前基础界面框架.md
- docs/codex_tasks/03-React与TypeScript前端代码规范.md
- frontend/src/components/layout/AppShell.tsx
- frontend/src/api/client.ts
- frontend/src/stores/ledger.store.ts

任务目标：
1. 新增 queryKeys 工厂。
2. 所有 ledger scoped query 带 ledgerId。
3. 引入 LedgerProvider 或等价上下文。
4. 切换账本时不使用 window.location.reload。
5. 增加 PermissionGate。

禁止事项：
- 不要重做整体视觉风格。
- 不要修改后端业务逻辑。

验收标准：
- 切换账本后 Dashboard/Transactions/Settings 正确刷新。
- viewer 无写入按钮。
- 前端测试覆盖 query key。
```

## Task37：设置页信息架构重组

```text
请先阅读：
- docs/ui/13-设置页信息架构重组.md
- frontend/src/pages/SettingsPage.tsx
- frontend/src/components/ledger/LedgerSettings.tsx

任务目标：
1. 将 SettingsPage 拆分为更清晰的卡片或二级页面。
2. 数据安全、账本成员、分类标签账户、周期规则、导入导出、系统诊断分区展示。
3. 保留现有备份恢复、导入、周期规则入口。
4. 移动端无横向滚动。

禁止事项：
- 不要删除现有功能入口。
- 不要实现未审核邀请机制。

验收标准：
- 设置页可读性提升。
- 高风险操作仍二次确认。
- 不同角色看到不同入口状态。
```

## Task38：迁移、测试与质量门禁

```text
请先阅读：
- docs/tech/17-数据迁移测试与质量门禁.md
- .github/workflows/ci.yml
- backend/migrations/

任务目标：
1. 增加迁移测试。
2. 增加多账本隔离测试。
3. 增加权限矩阵测试。
4. 增加导出和附件权限测试。
5. 更新 CI 文档。

禁止事项：
- 不要删除或 skip 核心测试。
- 不要修改已应用 migration。

验收标准：
- go test ./... 通过。
- 前端 lint/test/build 通过。
- Docker build 通过。
```

## Task39：附件访问控制

```text
请先阅读：
- docs/tech/15-账本上下文与RBAC权限框架.md
- docs/prd/11-v1.1前基础框架补齐.md
- backend/internal/http/router/router.go
- backend/internal/transaction/service.go

任务目标：
1. 设计并实现受保护附件访问 API。
2. 附件访问必须校验关联账单可见性。
3. private 账单附件不对其他成员可见。
4. 静态 /uploads 访问策略改为受控或限制。
5. 增加附件权限测试。

禁止事项：
- 不要破坏已有附件路径数据。
- 不要把附件存入数据库 BLOB。

验收标准：
- 无权限访问附件返回 403/404。
- 有权限用户可以预览/下载。
```

## Task40：审计与系统诊断中心

```text
请先阅读：
- docs/ui/13-设置页信息架构重组.md
- docs/tech/14-配置安全与部署一致性.md
- backend/internal/safety/service.go

任务目标：
1. 增强 /api/healthz 或新增诊断接口。
2. 设置页展示系统诊断状态。
3. 统一审计日志写入规范。
4. 可选增加审计日志只读查询接口。

禁止事项：
- 不要暴露 secret、token、password_hash。
- 不要把完整绝对路径直接返回前端。

验收标准：
- 诊断页面可帮助排查配置、数据库、备份、上传目录问题。
- 不泄露敏感信息。
```
