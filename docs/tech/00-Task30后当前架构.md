# 技术：Task30 后当前架构说明

状态：供审核  
目标：描述当前仓库在 Task30 完成后的真实架构状态，为后续基础框架补齐提供输入。

## 1. 当前总体架构

```text
Browser / PWA
  -> React + TypeScript + Vite frontend
  -> Go chi HTTP server
  -> REST JSON API
  -> Service layer
  -> Repository layer
  -> SQLite + goose migrations
  -> NAS local data/backups/uploads/logs
```

当前架构仍然适合私有化家庭记账，不需要整体推倒重写。

## 2. 当前后端架构

当前后端主要模块包括：

```text
internal/config
internal/db
internal/db/repo
internal/http/router
internal/http/handler
internal/http/middleware
internal/http/response
internal/errors
internal/transaction
internal/settlement
internal/dashboard
internal/reports
internal/safety
internal/ledger
```

已具备的能力：

1. 初始化和登录。
2. JWT Cookie 认证。
3. 统一响应结构。
4. goose migration。
5. 账单 CRUD。
6. 共同支出和多人分摊。
7. 结算净额和建议转账。
8. 模板、周期规则和周期提醒。
9. CSV 导入、导入规则和去重。
10. 附件上传路径记录。
11. 手动备份、恢复准备、CSV/JSON 导出。
12. 多账本和成员角色雏形。

## 3. 当前前端架构

当前前端主要能力：

```text
React Router
TanStack Query
Zustand stores
React Hook Form + Zod
TransactionFormDrawer
DraftListDrawer
AppShell
SettingsPage
LedgerSettings
ImportPage
RecurringRulesPage
Dashboard / Transactions / Settlement / Analytics
```

已具备：

1. 桌面侧边栏和移动底部导航。
2. active ledger store。
3. API client 自动附加 `X-Ledger-Id`。
4. 账单表单和模板。
5. 离线状态提示和草稿箱。
6. 设置页中的备份恢复、导出、导入、周期规则入口和账本成员管理。

## 4. 当前数据库架构

核心表包括：

```text
users
ledgers
ledger_members
accounts
categories
tags
transactions
transaction_splits
transaction_tags
settlements
audit_logs
app_settings
transaction_templates
recurring_rules
recurring_reminders
import_batches
import_items
import_rules
```

需要继续补足：

1. 分类/标签/账户归档和排序完整字段。
2. ledger invite 相关表，待 v1.1 审核后再进入实现。
3. 附件元数据表和权限访问 API。
4. schema version / migration audit 信息。

## 5. 当前部署架构

当前 Docker Compose 使用单容器部署，挂载：

```text
data/
backups/
uploads/
logs/
```

并配置 healthcheck 访问 `/api/healthz`。

需要补足：

1. 环境变量命名与后端 config 对齐。
2. 生产密钥强校验。
3. HTTP/HTTPS Cookie 策略说明。
4. 启动时配置诊断输出。

## 6. 当前架构风险

| 风险 | 说明 | 建议 |
|---|---|---|
| 文档事实源落后 | 代码已 v1.0，文档仍部分 v0.3/Demo | Task31 处理 |
| transaction 模块过重 | 聚合账单、模板、周期、导入、分类查询等多业务 | 后续分模块渐进拆分 |
| LedgerContext 分散 | 多个 service 自行查 ledger | Task33 统一 |
| 配置变量不一致 | Docker 和 config 命名需对齐 | Task32 统一 |
| 分类标签管理不足 | 长期记账必需基础能力 | Task35 补齐 |
| 附件静态访问 | private 附件可能绕开权限 | Task39 补齐 |
