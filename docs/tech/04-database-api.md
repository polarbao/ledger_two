# 技术：数据库与 API 设计

## 1. 数据库原则

- SQLite 作为 Demo 和早期版本数据库。
- 金额字段使用 INTEGER，单位为分。
- 时间字段使用 ISO8601 字符串或 SQLite 兼容时间格式。
- 删除采用 soft delete。
- 数据库迁移必须可重复执行和可追踪。

## 2. 核心表

- users。
- accounts。
- categories。
- tags。
- transactions。
- transaction_tags。
- transaction_splits。
- settlements。
- audit_logs。
- app_settings。

## 3. 后续扩展表

- ledgers。
- ledger_members。
- import_batches。
- import_items。
- import_rules。
- transaction_attachments。
- budgets。
- reminders。

## 4. API 响应格式

成功：

```json
{
  "success": true,
  "data": {}
}
```

失败：

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "金额不能为空"
  }
}
```

## 5. 核心 API

### Auth

- POST /api/auth/login
- POST /api/auth/logout
- GET /api/auth/me

### Init

- GET /api/init/status
- POST /api/init/setup

### Transactions

- GET /api/transactions
- POST /api/transactions
- GET /api/transactions/{id}
- PATCH /api/transactions/{id}
- DELETE /api/transactions/{id}

### Shared Expense

- POST /api/shared-expenses
- GET /api/shared-expenses
- GET /api/shared-expenses/{id}

### Settlement

- GET /api/settlements/balance
- GET /api/settlements
- POST /api/settlements

### Reports

- GET /api/reports/monthly-summary
- GET /api/reports/category-summary
- GET /api/reports/tag-summary
- GET /api/reports/member-summary

### Export / Backup

- GET /api/export/transactions.csv
- GET /api/export/full.json
- POST /api/admin/backup
- GET /api/admin/backups

## 6. API 契约要求

- 分页参数统一使用 page/page_size，后续可扩展 cursor。
- 筛选参数命名稳定。
- DTO 不直接暴露数据库内部字段。
- 所有跨端客户端复用同一套 API。
