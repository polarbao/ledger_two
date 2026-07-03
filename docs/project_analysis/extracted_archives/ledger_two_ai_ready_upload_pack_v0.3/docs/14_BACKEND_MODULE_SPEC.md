# 后端模块级实现规格 v0.3

## 1. 后端目标

后端使用 Go 实现单体 REST API 服务，负责：

1. 用户初始化与登录。
2. 账单 CRUD。
3. 共同支出分摊。
4. 结算计算。
5. 首页 Dashboard 聚合。
6. 统计报表。
7. CSV/JSON 导出。
8. SQLite 备份。
9. 托管前端静态文件，生产部署时可选。

## 2. 技术栈锁定

Demo 推荐：

```text
Go 1.22+
SQLite
chi 或 gin，优先 chi
sqlc 或 database/sql，Demo 可先 database/sql + 手写 repo
goose migration
bcrypt 或 argon2id
cookie session
```

### 2.1 chi vs gin

| 方案 | 优点 | 缺点 | Demo 建议 |
|---|---|---|---|
| chi | 标准库风格、轻量、中间件清晰 | 社区示例略少于 gin | 推荐 |
| gin | 生态大、示例多、上手快 | handler 风格更框架化 | 可选 |

Demo 版本推荐 chi，方便保持清晰架构。

### 2.2 sqlc vs GORM vs database/sql

| 方案 | 优点 | 缺点 | Demo 建议 |
|---|---|---|---|
| database/sql | 透明、依赖少 | SQL 与 struct 映射手写多 | Demo 可用 |
| sqlc | 类型安全、SQL 透明 | 需要额外生成步骤 | 长期推荐 |
| GORM | CRUD 快 | 容易隐藏 SQL，金额统计不透明 | 不推荐作为账务核心 |

Demo 可先使用 `database/sql`，后续再迁移到 sqlc。

## 3. 推荐目录结构

```text
backend/
  go.mod
  cmd/server/main.go
  internal/config/config.go
  internal/db/db.go
  internal/http/router.go
  internal/http/response.go
  internal/middleware/auth.go
  internal/auth/
    handler.go
    service.go
    session.go
  internal/user/
    model.go
    repo.go
    service.go
    handler.go
  internal/category/
  internal/tag/
  internal/account/
  internal/transaction/
    model.go
    dto.go
    repo.go
    service.go
    handler.go
  internal/settlement/
    dto.go
    service.go
    handler.go
  internal/report/
  internal/dashboard/
  internal/export/
  internal/backup/
  internal/audit/
  migrations/
    001_init.sql
    002_seed.sql
  data/.gitkeep
```

## 4. 统一响应结构

### 4.1 成功响应

```json
{
  "success": true,
  "data": {}
}
```

### 4.2 失败响应

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "金额必须大于 0"
  }
}
```

## 5. 错误码

| code | HTTP | 说明 |
|---|---:|---|
| `UNAUTHORIZED` | 401 | 未登录 |
| `FORBIDDEN` | 403 | 无权限 |
| `NOT_FOUND` | 404 | 数据不存在 |
| `VALIDATION_ERROR` | 400 | 参数错误 |
| `CONFLICT` | 409 | 状态冲突 |
| `INTERNAL_ERROR` | 500 | 内部错误 |

## 6. 核心数据模型

金额字段统一命名为 `amount_cents` 或数据库中的 `amount`，单位为分。

### 6.1 Transaction

```go
type Transaction struct {
    ID              string
    Type            string
    Title           string
    Amount          int64
    Currency        string
    OccurredAt      time.Time
    OwnerUserID     string
    CreatedByUserID string
    PayerUserID     string
    AccountID       sql.NullString
    CategoryID      sql.NullString
    Visibility      string
    Note            sql.NullString
    Status          string
    CreatedAt       time.Time
    UpdatedAt       time.Time
    DeletedAt       sql.NullTime
}
```

### 6.2 Split

```go
type TransactionSplit struct {
    ID            string
    TransactionID string
    UserID        string
    ShareAmount   int64
    ShareRatio    sql.NullInt64
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

## 7. 权限规则

### 7.1 查看账单

```go
func CanViewTransaction(currentUserID string, tx Transaction, participantIDs []string) bool {
    if tx.Status == "deleted" {
        return false
    }
    if tx.OwnerUserID == currentUserID || tx.CreatedByUserID == currentUserID || tx.PayerUserID == currentUserID {
        return true
    }
    if tx.Visibility == "partner_readable" {
        return true
    }
    if tx.Visibility == "shared" {
        return contains(participantIDs, currentUserID)
    }
    return false
}
```

### 7.2 编辑账单

Demo 版本：谁创建谁编辑。

```go
func CanEditTransaction(currentUserID string, tx Transaction) bool {
    return tx.Status != "deleted" && tx.CreatedByUserID == currentUserID
}
```

## 8. Auth 模块

### 8.1 API

```http
POST /api/auth/login
POST /api/auth/logout
GET  /api/auth/me
POST /api/auth/change-password
```

### 8.2 LoginRequest

```go
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}
```

### 8.3 MeResponse

```go
type MeResponse struct {
    ID          string `json:"id"`
    Username    string `json:"username"`
    DisplayName string `json:"display_name"`
    Role        string `json:"role"`
}
```

### 8.4 Session 策略

Demo 版本可使用服务端内存 session，但推荐直接用签名 Cookie：

1. Cookie 名称：`ledger_two_session`
2. 内容：用户 ID + 过期时间 + HMAC 签名
3. HttpOnly：true
4. SameSite：Lax
5. Secure：生产 HTTPS 时 true

## 9. Init 模块

### 9.1 API

```http
GET  /api/init/status
POST /api/init/setup
```

### 9.2 初始化规则

1. 如果 users 表为空，则允许 setup。
2. setup 必须创建两个用户。
3. setup 必须写入默认分类。
4. setup 必须写入默认账户。
5. setup 完成后写入 `app_settings.initialized=true`。

## 10. Transaction 模块

### 10.1 API

```http
GET    /api/transactions
POST   /api/transactions
GET    /api/transactions/{id}
PATCH  /api/transactions/{id}
DELETE /api/transactions/{id}
```

### 10.2 CreateTransactionRequest

```go
type CreateTransactionRequest struct {
    Type        string   `json:"type"`
    Title       string   `json:"title"`
    AmountCents int64    `json:"amount_cents"`
    Currency    string   `json:"currency"`
    OccurredAt  string   `json:"occurred_at"`
    PayerUserID string   `json:"payer_user_id"`
    AccountID   *string  `json:"account_id"`
    CategoryID  *string  `json:"category_id"`
    Visibility  string   `json:"visibility"`
    TagNames    []string `json:"tag_names"`
    Note        string   `json:"note"`
}
```

### 10.3 校验规则

1. `amount_cents > 0`
2. `type` 必须是 `expense` 或 `income`，共同支出走 shared API。
3. `title` 为空时，使用分类名称作为默认标题。
4. `visibility` 为空时默认 `private`。
5. `occurred_at` 必须是 ISO8601。
6. `payer_user_id` 必须是系统两个用户之一。

### 10.4 删除规则

DELETE 不物理删除，执行：

```sql
UPDATE transactions
SET status = 'deleted', deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
```

并写 audit log。

## 11. Shared Expense 模块

### 11.1 API

```http
POST /api/shared-expenses
GET  /api/shared-expenses/{id}
PATCH /api/shared-expenses/{id}
```

### 11.2 CreateSharedExpenseRequest

```go
type CreateSharedExpenseRequest struct {
    Title       string   `json:"title"`
    AmountCents int64    `json:"amount_cents"`
    Currency    string   `json:"currency"`
    OccurredAt  string   `json:"occurred_at"`
    PayerUserID string   `json:"payer_user_id"`
    CategoryID  *string  `json:"category_id"`
    SplitMethod string   `json:"split_method"`
    Participants []SplitInput `json:"participants"`
    TagNames    []string `json:"tag_names"`
    Note        string   `json:"note"`
}

type SplitInput struct {
    UserID string `json:"user_id"`
    ShareAmountCents int64 `json:"share_amount_cents"`
}
```

### 11.3 equal 分摊算法

两人平均分摊时，奇数分按稳定规则给支付人承担多 1 分：

```text
amount = 10001
payer = A
A share = 5001
B share = 5000
```

算法：

```go
base := amount / 2
rem := amount % 2
payerShare := base + rem
otherShare := base
```

### 11.4 payer_only 算法

```text
付款人承担全部金额
对方 share = 0
不进入待结算
```

但仍然可以作为 shared 可见账单。

## 12. Settlement 模块

### 12.1 API

```http
GET  /api/settlements/balance?month=2026-06
POST /api/settlements
GET  /api/settlements?month=2026-06
```

### 12.2 余额计算

每个用户净额：

```text
net = 实际支付共同支出金额 - 实际应承担共同支出金额 - 已向别人结算金额 + 别人向我结算金额
```

对两人账本：

```text
A_net = A_paid_shared - A_share - A_paid_settlement_to_B + B_paid_settlement_to_A
B_net = B_paid_shared - B_share - B_paid_settlement_to_A + A_paid_settlement_to_B
```

如果 `A_net > 0`，B 应向 A 支付 `A_net`。

### 12.3 CreateSettlementRequest

```go
type CreateSettlementRequest struct {
    FromUserID string `json:"from_user_id"`
    ToUserID   string `json:"to_user_id"`
    AmountCents int64 `json:"amount_cents"`
    OccurredAt string `json:"occurred_at"`
    Note       string `json:"note"`
}
```

### 12.4 结算规则

1. `from_user_id != to_user_id`
2. `amount_cents > 0`
3. 结算生成 `settlements` 表记录。
4. 同时生成一条 `transactions.type=settlement` 的可见流水，便于流水追踪。
5. 不修改历史 shared_expense。

## 13. Dashboard 模块

### 13.1 API

```http
GET /api/dashboard?month=2026-06&scope=all_visible
```

### 13.2 Response

```go
type DashboardResponse struct {
    Month string `json:"month"`
    TotalExpenseCents int64 `json:"total_expense_cents"`
    TotalIncomeCents int64 `json:"total_income_cents"`
    MyPaidCents int64 `json:"my_paid_cents"`
    PartnerPaidCents int64 `json:"partner_paid_cents"`
    SharedBalance BalanceResponse `json:"shared_balance"`
    RecentTransactions []TransactionListItem `json:"recent_transactions"`
    CategorySummary []SummaryItem `json:"category_summary"`
    TagSummary []SummaryItem `json:"tag_summary"`
    UserStats []UserStatItem `json:"user_stats"`
}
```

## 14. Export 模块

### 14.1 API

```http
GET /api/export/transactions.csv
GET /api/export/full.json
```

### 14.2 CSV 字段

```text
id,type,title,amount_cents,currency,occurred_at,payer,category,visibility,tags,note
```

## 15. Backup 模块

Demo 版本实现手动备份即可：

```http
POST /api/admin/backup
GET  /api/admin/backups
```

SQLite 备份使用：

```sql
VACUUM INTO '/app/backups/ledger-two-YYYYMMDD-HHMMSS.db';
```

## 16. 后端测试必须覆盖

1. 金额为 0 创建失败。
2. private 账单对方不可见。
3. shared expense equal 分摊正确。
4. 奇数分金额分摊正确。
5. A 垫付、B 垫付、结算后净额正确。
6. 删除 shared expense 后净额重新计算正确。
7. settlement 不修改历史账单。
