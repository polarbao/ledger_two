# API Conventions

状态：Task34.3 规范基线  
适用阶段：Foundation before v1.1  
关联文档：

- `docs/api/API_INVENTORY.md`
- `docs/api/openapi.yaml`
- `docs/tech/10-error-codes.md`
- `docs/tech/16-api-contract-openapi-error.md`

## 1. 强制响应结构

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
    "message": "金额必须大于 0",
    "details": null
  }
}
```

规则：

1. handler 不得新增自定义错误 JSON。
2. 新业务错误必须先登记错误码。
3. 生产环境不得把 SQL、文件绝对路径、token、password_hash、secret 返回给前端。
4. 前端必须以 `error.code` 做稳定分支，不解析 `message`。

## 2. 错误码冻结枚举

### 2.1 通用错误码

| Code | HTTP | 使用场景 |
|---|---:|---|
| BAD_REQUEST | 400 | 请求体格式错误、Content-Type 不支持、JSON 解析失败 |
| VALIDATION_ERROR | 400 | 字段校验失败、参数范围错误 |
| UNAUTHORIZED | 401 | 未登录、Cookie 缺失或 token 无效 |
| FORBIDDEN | 403 | 已登录但无权限、非账本成员、角色不允许 |
| NOT_FOUND | 404 | 资源不存在或对当前用户不可见 |
| CONFLICT | 409 | 状态冲突、重复名称、重复成员、重复初始化 |
| RATE_LIMITED | 429 | 后续可选，当前未实现 |
| INTERNAL_ERROR | 500 | 未预期服务端错误 |
| SERVICE_UNAVAILABLE | 503 | 数据库、备份目录、外部存储不可用 |

### 2.2 业务错误码

| Domain | Code | HTTP | 使用场景 |
|---|---|---:|---|
| init/auth | APP_ALREADY_INITIALIZED | 409 | 系统已初始化 |
| init/auth | APP_NOT_INITIALIZED | 409 | 系统尚未初始化 |
| init/auth | INVALID_CREDENTIALS | 401 | 用户名或密码错误 |
| init/auth | SESSION_EXPIRED | 401 | 登录过期 |
| init/auth | PASSWORD_TOO_WEAK | 400 | 密码强度不足 |
| transaction | TRANSACTION_NOT_FOUND | 404 | 账单不存在或不可见 |
| transaction | TRANSACTION_AMOUNT_INVALID | 400 | 金额为空、为 0、负数或超过限制 |
| transaction | TRANSACTION_TYPE_INVALID | 400 | 不支持的账单类型 |
| transaction | TRANSACTION_VISIBILITY_INVALID | 400 | 不支持的可见性 |
| transaction | TRANSACTION_NOT_EDITABLE | 403 | 当前用户不可编辑 |
| transaction | TRANSACTION_ALREADY_DELETED | 409 | 账单已删除 |
| transaction | TRANSACTION_HAS_SETTLEMENT_EFFECT | 409 | 操作影响结算，需要确认 |
| split | SPLIT_METHOD_INVALID | 400 | 不支持的分摊方式 |
| split | SPLIT_PARTICIPANTS_INVALID | 400 | 参与人不属于账本或为空 |
| split | SPLIT_AMOUNT_MISMATCH | 400 | 分摊金额不等于总金额 |
| split | SPLIT_RATIO_MISMATCH | 400 | 分摊比例不等于 100% |
| split | PAYER_NOT_FOUND | 400 | 付款人不存在或不属于账本 |
| settlement | SETTLEMENT_AMOUNT_INVALID | 400 | 结算金额无效 |
| settlement | SETTLEMENT_NOT_REQUIRED | 409 | 当前没有需要结算金额 |
| settlement | SETTLEMENT_DIRECTION_INVALID | 400 | 付款人和收款人方向错误 |
| settlement | SETTLEMENT_NOT_FOUND | 404 | 结算记录不存在 |
| metadata | CATEGORY_NOT_FOUND | 404 | 分类不存在 |
| metadata | CATEGORY_ARCHIVED | 409 | 分类已归档，不可用于新账单 |
| metadata | TAG_NOT_FOUND | 404 | 标签不存在 |
| metadata | ACCOUNT_NOT_FOUND | 404 | 支付账户不存在 |
| metadata | DUPLICATE_NAME | 409 | 同账本内名称重复 |
| import/export/backup | EXPORT_FAILED | 500 | 导出失败 |
| import/export/backup | BACKUP_FAILED | 500 | 备份失败 |
| import/export/backup | BACKUP_NOT_FOUND | 404 | 备份文件不存在 |
| import/export/backup | BACKUP_PATH_INVALID | 500 | 备份目录不可写或不存在 |
| import/export/backup | IMPORT_FILE_INVALID | 400 | 导入文件格式不支持 |
| import/export/backup | IMPORT_DUPLICATE_ITEM | 409 | 导入项重复 |
| import/export/backup | IMPORT_PREVIEW_EXPIRED | 409 | 导入预览已过期 |
| import/export/backup | IMPORT_BATCH_NOT_FOUND | 404 | 导入批次不存在或对当前用户不可见 |
| import/export/backup | IMPORT_ROW_INVALID | 400 | 导入行必需字段缺失、金额或时间无法解析 |
| import/export/backup | IMPORT_ROW_REQUIRES_CONFIRMATION | 409 | suspicious 行尚未由用户确认导入或跳过 |
| import/export/backup | IMPORT_COMMIT_CONFLICT | 409 | 导入批次状态不允许提交、已提交或已过期 |

## 3. Details 字段

字段级错误：

```json
{
  "field_errors": {
    "amount_cents": "金额必须大于 0",
    "occurred_at": "时间格式必须为 ISO8601"
  }
}
```

导入错误：

```json
{
  "rows": [
    {
      "row": 3,
      "code": "VALIDATION_ERROR",
      "message": "金额为空"
    }
  ]
}
```

约束：

1. `details` 可以为空。
2. `details` 不放内部堆栈。
3. 导入、批量操作可以返回行级错误，但不能写入半批脏数据。

## 4. 分页规范

短期已有列表接口可继续返回数组。新增或改造后的分页列表统一返回：

```json
{
  "items": [],
  "page": 1,
  "page_size": 20,
  "total": 100
}
```

查询参数：

| Param | Type | Default | Rule |
|---|---|---:|---|
| page | integer | 1 | 最小 1 |
| page_size | integer | 20 | 最小 1，最大 100 |

约束：

1. 第一页为 `page=1`。
2. `total` 是符合筛选条件的总数，不是当前页数量。
3. 当前无分页的接口，在 OpenAPI 中标为 transitional；引入分页时需要兼容旧前端。

## 5. 筛选规范

通用命名：

| Param | 示例 | 说明 |
|---|---|---|
| month | `2026-07` | 月份，格式 `YYYY-MM` |
| date_from | `2026-07-01` | 起始日期，闭区间 |
| date_to | `2026-07-31` | 结束日期，闭区间 |
| type | `expense` | 账单类型 |
| visibility | `private` | 可见性 |
| category_id | UUID | 分类 |
| account_id | UUID | 支付账户 |
| tag | `餐饮` | 单个标签 |
| keyword | `咖啡` | 标题、备注等模糊搜索 |
| status | `pending` | 状态筛选 |

金额筛选如后续需要，统一：

| Param | 说明 |
|---|---|
| amount_min_cents | 最小金额，整数分 |
| amount_max_cents | 最大金额，整数分 |

## 6. 排序规范

查询参数：

```text
sort=occurred_at:desc
```

规则：

1. 格式为 `{field}:{direction}`。
2. `direction` 只能是 `asc` 或 `desc`。
3. 默认排序必须写入 OpenAPI。
4. 不允许前端传任意数据库字段。

短期允许字段：

| Resource | Fields |
|---|---|
| transactions | `occurred_at`, `created_at`, `amount_cents` |
| settlements | `occurred_at`, `created_at`, `amount_cents` |
| templates | `name`, `created_at` |
| recurring_rules | `next_due_date`, `created_at` |
| backups | `created_at`, `size_bytes` |

## 7. 金额、时间和 ID

金额：

- API 使用整数分。
- 字段命名优先 `amount_cents`、`share_amount_cents`、`paid_amount_cents`。
- UI 负责元和分转换。
- 禁止 float 表示金额。

时间：

- 请求和响应使用 ISO8601 字符串。
- 月份使用 `YYYY-MM`。
- 日期使用 `YYYY-MM-DD`。

ID：

- API 暴露 string UUID。
- 路径参数统一写 `{id}` 或领域名 `{transactionId}`、`{ledgerId}`。

## 8. Ledger Context

当前：

- 显式账本通过 `X-Ledger-Id` 传入。
- Foundation 兼容期内，部分接口仍 fallback 到用户第一个账本。

冻结目标：

1. 业务写接口必须显式传 `X-Ledger-Id` 或使用 path ledger。
2. 非成员返回 `FORBIDDEN`。
3. viewer 写操作返回 `FORBIDDEN`。
4. 前端 query key 必须包含 ledgerId。

## 9. Metadata 管理规则

Task35 后端基础接口：

```text
GET   /api/metadata/{kind}/
POST  /api/metadata/{kind}/
POST  /api/metadata/{kind}/reorder
PATCH /api/metadata/{kind}/{id}
POST  /api/metadata/{kind}/{id}/archive
POST  /api/metadata/{kind}/{id}/restore
```

规则：

1. `kind` 仅允许 `categories`、`tags`、`accounts`。
2. 只有 owner 可以新增、编辑、归档、恢复。
3. viewer 和 editor 默认不可管理元数据。
4. 归档不物理删除，历史账单仍可显示。
5. 新增账单选择器只返回未归档分类和账户。
6. 分类同账本同 type 下名称唯一；标签和账户同账本名称唯一。
7. 管理列表返回 `sort_order` 和 `usage_count`，用于排序操作和归档前风险提示；不得据此阻止归档。

## 10. 新 API 准入

新增或修改 API 必须同步：

1. 更新 `docs/api/API_INVENTORY.md`。
2. 更新 `docs/api/openapi.yaml`。
3. 若新增错误码，更新本文件和 `docs/tech/10-error-codes.md`。
4. 若新增列表接口，明确分页、筛选和排序。
5. 若新增写接口，明确角色权限和审计要求。

## 11. Task50 v1.3 冻结补充

以下规则由 Task50P.4 冻结，但在 Task50 代码完成前仍属于目标契约：

1. 账本内业务请求缺少显式账本返回 400 `LEDGER_REQUIRED`，不再 fallback 到首个账本。
2. 账本管理 path 与可选 `X-Ledger-Id` 不一致返回 400 `LEDGER_CONTEXT_MISMATCH`。
3. 非成员访问账本统一返回 403 `LEDGER_ACCESS_DENIED`；成员在可访问账本中请求不存在对象返回 404 `LEDGER_OBJECT_NOT_FOUND`。
4. archived 写入返回 409 `LEDGER_ARCHIVED`；恢复是 archived 唯一 lifecycle mutation。
5. rename/archive/restore/add/update/remove/leave/transfer 必须携带 `If-Match: "ledger:<id>:v<version>"`；冲突返回 409 `LEDGER_VERSION_CONFLICT`。
6. 成员上限、Owner 移交、ready 导入阻断分别使用 `LEDGER_MEMBER_LIMIT_REACHED`、`LEDGER_OWNER_TRANSFER_REQUIRED`、`LEDGER_READY_IMPORT_EXISTS`。
7. 整库 backup/restore/diagnostics 不读取 Ledger Context，仅由实例管理员授权；拒绝使用 403 `INSTANCE_ADMIN_REQUIRED`。
8. 完整 DTO、HTTP 映射和响应 envelope 以 `openapi-v1.3-ledger-draft.yaml` 为准。
