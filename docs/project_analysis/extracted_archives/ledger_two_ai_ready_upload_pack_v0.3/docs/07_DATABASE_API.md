# 07 数据库与 API 设计文档：LedgerTwo v0.2

## 1. 数据库原则

1. 金额使用整数分。
2. 时间使用 ISO 8601 字符串。
3. 账单删除使用软删除。
4. 共同支出通过 splits 表保存分摊。
5. 结算记录独立保存。
6. 预留多账本字段 `ledger_id`，MVP 可只有一个默认账本。

## 2. 核心表

### 2.1 users

```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar_url TEXT,
    role TEXT NOT NULL DEFAULT 'user',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

### 2.2 ledgers

```sql
CREATE TABLE ledgers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    default_currency TEXT NOT NULL DEFAULT 'CNY',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

### 2.3 accounts

```sql
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    initial_balance INTEGER NOT NULL DEFAULT 0,
    is_archived INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id)
);
```

### 2.4 categories

```sql
CREATE TABLE categories (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    owner_user_id TEXT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    icon TEXT,
    color TEXT,
    parent_id TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_system INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id),
    FOREIGN KEY (parent_id) REFERENCES categories(id)
);
```

### 2.5 tags

```sql
CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    name TEXT NOT NULL,
    owner_user_id TEXT,
    color TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id)
);
```

### 2.6 transactions

```sql
CREATE TABLE transactions (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    occurred_at TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    payer_user_id TEXT,
    account_id TEXT,
    category_id TEXT,
    visibility TEXT NOT NULL DEFAULT 'private',
    split_method TEXT,
    note TEXT,
    status TEXT NOT NULL DEFAULT 'normal',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (owner_user_id) REFERENCES users(id),
    FOREIGN KEY (created_by_user_id) REFERENCES users(id),
    FOREIGN KEY (payer_user_id) REFERENCES users(id),
    FOREIGN KEY (account_id) REFERENCES accounts(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);
```

### 2.7 transaction_splits

```sql
CREATE TABLE transaction_splits (
    id TEXT PRIMARY KEY,
    transaction_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    share_amount INTEGER NOT NULL,
    share_ratio INTEGER,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### 2.8 transaction_tags

```sql
CREATE TABLE transaction_tags (
    transaction_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (transaction_id, tag_id),
    FOREIGN KEY (transaction_id) REFERENCES transactions(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);
```

### 2.9 settlements

```sql
CREATE TABLE settlements (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    from_user_id TEXT NOT NULL,
    to_user_id TEXT NOT NULL,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'CNY',
    occurred_at TEXT NOT NULL,
    note TEXT,
    created_by_user_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (from_user_id) REFERENCES users(id),
    FOREIGN KEY (to_user_id) REFERENCES users(id),
    FOREIGN KEY (created_by_user_id) REFERENCES users(id)
);
```

### 2.10 audit_logs

```sql
CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    ledger_id TEXT NOT NULL,
    actor_user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    created_at TEXT NOT NULL,
    FOREIGN KEY (ledger_id) REFERENCES ledgers(id),
    FOREIGN KEY (actor_user_id) REFERENCES users(id)
);
```

## 3. 索引

```sql
CREATE INDEX idx_transactions_ledger_month ON transactions(ledger_id, occurred_at);
CREATE INDEX idx_transactions_payer ON transactions(ledger_id, payer_user_id, occurred_at);
CREATE INDEX idx_transactions_category ON transactions(ledger_id, category_id, occurred_at);
CREATE INDEX idx_transactions_type ON transactions(ledger_id, type, occurred_at);
CREATE INDEX idx_splits_transaction ON transaction_splits(transaction_id);
CREATE INDEX idx_splits_user ON transaction_splits(user_id);
CREATE INDEX idx_settlements_users ON settlements(ledger_id, from_user_id, to_user_id, occurred_at);
```

## 4. API 设计

统一前缀：`/api/v1`。

### 4.1 Auth

```http
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
POST /api/v1/auth/change-password
```

### 4.2 Dashboard

```http
GET /api/v1/dashboard?month=2025-04&scope=shared
```

返回：

```json
{
  "success": true,
  "data": {
    "month": "2025-04",
    "total_expense": 328000,
    "my_paid": 188000,
    "partner_paid": 140000,
    "shared_balance": {
      "settled": false,
      "from_user_id": "lynn",
      "to_user_id": "polar",
      "amount": 18650
    },
    "recent_transactions": [],
    "category_summary": []
  }
}
```

### 4.3 Transactions

```http
GET    /api/v1/transactions
POST   /api/v1/transactions
GET    /api/v1/transactions/{id}
PATCH  /api/v1/transactions/{id}
DELETE /api/v1/transactions/{id}
```

查询参数：

```http
GET /api/v1/transactions?month=2025-04&type=expense&category_id=xxx&keyword=午餐&page=1&page_size=20
```

### 4.4 Shared Expense

```http
POST /api/v1/shared-expenses
GET  /api/v1/shared-expenses
GET  /api/v1/shared-expenses/{id}
```

创建请求：

```json
{
  "title": "晚餐",
  "amount": 20000,
  "currency": "CNY",
  "occurred_at": "2025-04-12T19:00:00+08:00",
  "payer_user_id": "polar",
  "category_id": "food",
  "split_method": "equal",
  "participants": [
    { "user_id": "polar", "share_amount": 10000 },
    { "user_id": "lynn", "share_amount": 10000 }
  ],
  "tag_names": ["晚餐"],
  "note": ""
}
```

### 4.5 Settlement

```http
GET  /api/v1/settlements
POST /api/v1/settlements
GET  /api/v1/settlements/balance?month=2025-04
```

### 4.6 Reports

```http
GET /api/v1/reports/trend?from=2025-01&to=2025-12
GET /api/v1/reports/category-summary?month=2025-04
GET /api/v1/reports/member-summary?month=2025-04
GET /api/v1/reports/tag-summary?month=2025-04
```

### 4.7 Settings

```http
GET    /api/v1/categories
POST   /api/v1/categories
PATCH  /api/v1/categories/{id}
DELETE /api/v1/categories/{id}

GET    /api/v1/tags
POST   /api/v1/tags
PATCH  /api/v1/tags/{id}
DELETE /api/v1/tags/{id}

GET    /api/v1/accounts
POST   /api/v1/accounts
PATCH  /api/v1/accounts/{id}
DELETE /api/v1/accounts/{id}
```

### 4.8 Export & Backup

```http
GET  /api/v1/export/transactions.csv
GET  /api/v1/export/full.json
POST /api/v1/admin/backups
GET  /api/v1/admin/backups
GET  /api/v1/admin/backups/{filename}
```

## 5. 跨端预留接口

### 5.1 Token 登录预留

MVP 使用 Cookie Session，但预留：

```http
POST /api/v1/auth/token
```

用于后续移动端 App。

### 5.2 文件上传预留

```http
POST /api/v1/uploads
GET  /api/v1/uploads/{id}
DELETE /api/v1/uploads/{id}
```

### 5.3 同步接口预留

```http
GET /api/v1/sync/changes?since=timestamp
```

用于 PWA 离线缓存和移动端增量同步。
