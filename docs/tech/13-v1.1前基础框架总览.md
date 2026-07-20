# 技术：v1.1 前基础框架总览

状态：供审核  
目标：定义进入 v1.1 业务开发前必须完成的基础技术框架。

## 1. 总体原则

1. 不推倒重写。
2. 不提前实现未审核 v1.1 业务。
3. 以渐进式架构收口方式补齐基础框架。
4. 每个基础框架任务都必须可测试、可回滚、可小步提交。
5. 所有账务核心规则继续以后端为准。

## 2. 基础框架分层

```text
Foundation Layer
  ├── Documentation Source of Truth
  ├── Config & Deployment Safety
  ├── LedgerContext & RBAC
  ├── API Contract & OpenAPI
  ├── Data Migration & Backup Safety
  ├── Category / Tag / Account Management
  ├── Frontend App State & Query Keys
  ├── Testing & Quality Gates
  └── AI/Codex Development Rules
```

## 3. 后端目标架构

建议从当前结构逐步演进为：

```text
internal/
  app/                 dependency wiring, routes
  config/              env parsing, validation
  db/                  sqlite, migrations, transaction helper
  errors/              AppError, error codes
  http/
    middleware/        auth, ledger context, request id
    response/          success/error output
  identity/            users, login profile, future account lifecycle
  ledger/              ledger aggregate
  membership/          member roles, RBAC guard
  category/            category CRUD/order/archive
  tag/                 tag CRUD/order/archive
  account/             payment account CRUD/order/archive
  transaction/         transaction CRUD only
  sharedexpense/       shared expense + sync states, later
  split/               split calculator
  settlement/          settlement and transfer suggestions
  template/            transaction templates
  recurring/           recurring rules and reminders
  importer/            CSV parse/analyze/commit/import rules
  attachment/          upload metadata and guarded access
  safety/              backup/restore/export
  audit/               audit log write/query
  report/              reports and dashboard aggregation
```

此结构不要求一次性完成，可以按 Foundation Task 渐进拆分。

## 4. 前端目标架构

建议从当前结构逐步演进为：

```text
src/
  app/
    router.tsx
    providers/
      AuthProvider.tsx
      LedgerProvider.tsx
      QueryProvider.tsx
  api/
    client.ts
    queryKeys.ts
    generated/         future OpenAPI generated types
  permissions/
    PermissionGate.tsx
    rolePolicy.ts
  features/
    ledger/
    membership/
    category/
    tag/
    account/
    transaction/
    sharedExpense/
    settlement/
    safety/
    settings/
  components/
    ui/
    layout/
  stores/
    ui.store.ts
    draft.store.ts
  utils/
```

## 5. 必须补齐的统一对象

### 5.1 LedgerContext

```go
type LedgerContext struct {
    UserID   string
    LedgerID string
    Role     string
}
```

用途：

- 业务 service 不直接从 header 或数据库猜 ledger。
- 统一判断 membership。
- 统一传入 repository 查询。
- 统一写审计日志。

### 5.2 RolePolicy

```text
owner:  管理账本、成员、配置、分类、标签、账户、导出、备份恢复
editor: 记账、编辑自己创建的账单、参与结算、按配置管理分类标签
viewer: 查看授权数据、统计和结算状态，不可写入
```

### 5.3 APIError

所有错误响应保持：

```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "当前角色无权执行此操作",
    "details": null
  }
}
```

### 5.4 Query Key

前端所有服务端状态必须以 ledgerId 作为关键维度：

```ts
queryKeys.transactions(ledgerId, filters)
queryKeys.dashboard(ledgerId, month)
queryKeys.categories(ledgerId)
queryKeys.members(ledgerId)
queryKeys.settlements(ledgerId, month)
```

## 6. 迁移策略

1. 不修改已应用 migration。
2. 新增 migration 必须有 `Up` 和 `Down`。
3. 所有破坏性 migration 前必须自动安全备份。
4. v1.0 数据升级到 Foundation 版本必须有回归测试。
5. migration 中禁止 silently drop 数据。

## 7. 验收标准

Foundation 技术框架完成时：

1. 所有业务 API 都能从统一 LedgerContext 获取 ledger 和 role。
2. 所有写操作都能被 RolePolicy 校验。
3. 配置变量命名一致，生产环境缺少密钥拒绝启动。
4. API contract 有文档和 OpenAPI 草案。
5. 分类、标签、账户管理有完整基础接口。
6. 前端 query key 与 ledgerId 强绑定。
7. CI 增加多账本/权限/迁移/导出/附件回归测试。
8. Codex/Gemini 任务能从文档直接执行。
