# 技术：统一错误码与异常处理

## 1. 文档目标

本文件定义 LedgerTwo 后端 API 的统一错误码、HTTP 状态码映射、前端错误展示策略和日志记录要求。

Task34 后，API 契约侧的冻结枚举、分页、筛选、排序和 Ledger Context 规则同时维护在：

```text
docs/api/API_CONVENTIONS.md
```

统一错误码的目标是：

1. 前端可以稳定识别错误类型。
2. AI/开发者实现模块时不会各自发明错误格式。
3. 后续 Web、PWA、移动端共用同一套错误契约。
4. 日志中能快速定位问题。

## 2. API 错误响应格式

所有失败响应统一为：

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

字段说明：

| 字段 | 类型 | 说明 |
|---|---|---|
| success | boolean | 固定 false |
| error.code | string | 稳定错误码，供前端判断 |
| error.message | string | 用户可读提示或开发环境提示 |
| error.details | object/null | 字段级错误、调试信息，生产环境谨慎返回 |

## 3. 通用错误码

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| BAD_REQUEST | 400 | 请求格式错误，例如 JSON 解析失败 |
| VALIDATION_ERROR | 400 | 参数校验失败 |
| UNAUTHORIZED | 401 | 未登录或 Session 失效 |
| FORBIDDEN | 403 | 已登录但无权限 |
| NOT_FOUND | 404 | 资源不存在或不可见 |
| CONFLICT | 409 | 状态冲突，例如重复初始化 |
| RATE_LIMITED | 429 | 请求过于频繁，后续可选 |
| INTERNAL_ERROR | 500 | 未预期服务端错误 |
| SERVICE_UNAVAILABLE | 503 | 数据库不可用、备份服务不可用等 |

## 4. 业务错误码

### 4.1 初始化与认证

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| APP_ALREADY_INITIALIZED | 409 | 系统已经初始化，不允许重复初始化 |
| APP_NOT_INITIALIZED | 409 | 系统尚未初始化，不能登录或记账 |
| INVALID_CREDENTIALS | 401 | 用户名或密码错误 |
| SESSION_EXPIRED | 401 | Session 过期 |
| PASSWORD_TOO_WEAK | 400 | 密码强度不足 |

### 4.2 账单

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| TRANSACTION_NOT_FOUND | 404 | 账单不存在或对当前用户不可见 |
| TRANSACTION_AMOUNT_INVALID | 400 | 金额为空、为 0、负数或超过限制 |
| TRANSACTION_TYPE_INVALID | 400 | 不支持的账单类型 |
| TRANSACTION_VISIBILITY_INVALID | 400 | 不支持的可见性 |
| TRANSACTION_NOT_EDITABLE | 403 | 当前用户不能编辑该账单 |
| TRANSACTION_ALREADY_DELETED | 409 | 账单已删除 |
| TRANSACTION_HAS_SETTLEMENT_EFFECT | 409 | 操作会影响结算，需二次确认 |

### 4.3 共同支出与分摊

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| SPLIT_METHOD_INVALID | 400 | 不支持的分摊方式 |
| SPLIT_PARTICIPANTS_INVALID | 400 | 参与人为空或不属于账本 |
| SPLIT_AMOUNT_MISMATCH | 400 | 分摊金额合计不等于账单金额 |
| SPLIT_RATIO_MISMATCH | 400 | 分摊比例合计不等于 100% |
| PAYER_NOT_FOUND | 400 | 付款人不存在或不属于账本 |

### 4.4 结算

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| SETTLEMENT_AMOUNT_INVALID | 400 | 结算金额无效 |
| SETTLEMENT_NOT_REQUIRED | 409 | 当前没有需要结算的金额 |
| SETTLEMENT_DIRECTION_INVALID | 400 | 付款人和收款人方向错误 |
| SETTLEMENT_NOT_FOUND | 404 | 结算记录不存在 |

### 4.5 分类、标签、账户

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| CATEGORY_NOT_FOUND | 404 | 分类不存在 |
| CATEGORY_ARCHIVED | 409 | 分类已归档，不能用于新账单 |
| TAG_NOT_FOUND | 404 | 标签不存在 |
| ACCOUNT_NOT_FOUND | 404 | 账户不存在 |
| DUPLICATE_NAME | 409 | 分类、标签或账户名称重复 |

### 4.6 导入、导出、备份

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| EXPORT_FAILED | 500 | 导出失败 |
| BACKUP_FAILED | 500 | 备份失败 |
| BACKUP_NOT_FOUND | 404 | 备份文件不存在 |
| BACKUP_PATH_INVALID | 500 | 备份目录不可写或不存在 |
| IMPORT_FILE_INVALID | 400 | 导入文件格式不支持 |
| IMPORT_DUPLICATE_ITEM | 409 | 导入项重复 |
| IMPORT_PREVIEW_EXPIRED | 409 | 导入预览已过期 |
| IMPORT_BATCH_NOT_FOUND | 404 | 导入批次不存在或对当前用户不可见 |
| IMPORT_ROW_INVALID | 400 | 导入行必需字段缺失、金额或时间无法解析 |
| IMPORT_ROW_REQUIRES_CONFIRMATION | 409 | suspicious 行尚未由用户确认导入或跳过 |
| IMPORT_COMMIT_CONFLICT | 409 | 导入批次状态不允许提交、已提交或已过期 |
| IMPORT_RECLASSIFY_CONFLICT | 409 | 分类器关闭、批次非 ready/已过期或重新分类期间发生并发变化 |

Task53.4 已实现并进入运行时事实源的错误码如下：

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| IMPORT_BULK_ADJUST_CONFLICT | 409 | 批量调整批次状态或并发快照冲突 |
| CATEGORY_FALLBACK_REQUIRED | 409 | 归档支出/收入兜底分类时未指定替代分类 |
| CATEGORY_FALLBACK_REPLACEMENT_INVALID | 409 | 兜底替代分类类型、状态或 system_key 不合法 |
| CATEGORY_TYPE_MISMATCH | 400 | 分类类型与导入行目标交易类型不兼容 |
| TAG_LIMIT_EXCEEDED | 400 | 最终标签超过 8 个 |
| CLASSIFICATION_CONFLICT | 409 | 显式规则或同级分类候选冲突 |
| CLASSIFICATION_RULE_STALE | 409 | 规则或持久化建议引用不可用元数据 |
| CLASSIFICATION_MERCHANT_REQUIRED | 400 | 学习规则的规范化商户为空 |

## 5. 前端展示策略

### 5.1 表单错误

字段级错误展示在字段下方，例如：

```text
金额必须大于 0
```

表单级错误展示在表单顶部，例如：

```text
保存失败，请检查账单信息后重试
```

### 5.2 权限错误

`FORBIDDEN`、`TRANSACTION_NOT_EDITABLE`：

```text
你没有权限执行此操作
```

### 5.3 未登录错误

`UNAUTHORIZED`、`SESSION_EXPIRED`：

- 清理前端当前用户状态。
- 跳转登录页。
- 展示：登录状态已过期，请重新登录。

### 5.4 服务异常

`INTERNAL_ERROR`、`SERVICE_UNAVAILABLE`：

```text
服务暂时不可用，请稍后重试。如果持续失败，请检查 NAS 服务状态。
```

### 5.5 高风险操作错误

影响结算或数据安全的错误，必须使用 Modal 确认，不只用 Toast。

## 6. 后端实现建议

建议定义：

```go
type APIError struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details any         `json:"details,omitempty"`
}
```

并提供统一响应函数：

```go
func WriteError(w http.ResponseWriter, err error)
func WriteJSON[T any](w http.ResponseWriter, data T)
```

业务层返回领域错误，例如：

```go
var ErrTransactionNotEditable = NewAppError("TRANSACTION_NOT_EDITABLE", http.StatusForbidden, "当前用户不能编辑该账单")
```

handler 不直接拼错误响应。

## 7. 日志策略

日志中应包含：

- request_id。
- user_id。
- path。
- method。
- error_code。
- internal_error。

生产环境不应把数据库路径、Session、密码 hash、密钥等敏感信息返回给前端。

## 8. 验收标准

- 所有 API 失败响应格式一致。
- 前端 api/client.ts 能统一识别错误码。
- 未登录自动跳转登录页。
- 权限错误不会暴露资源是否真实存在的敏感信息。
- 备份、导出失败时有明确错误码。
- 表单校验错误能定位到字段。

## 9. Task50 v1.3 目标错误码

以下错误码已由 Task50P.4 冻结，当前 router 完成 Task50 实现前不得视为已上线：

| 错误码 | HTTP 状态 | 说明 |
|---|---:|---|
| LEDGER_REQUIRED | 400 | 账本内业务请求未显式指定账本 |
| LEDGER_CONTEXT_MISMATCH | 400 | path 账本与 `X-Ledger-Id` 不一致 |
| LEDGER_ACCESS_DENIED | 403 | 当前用户不是账本成员或角色不允许操作 |
| INSTANCE_ADMIN_REQUIRED | 403 | 当前用户没有整库运维能力 |
| LEDGER_OBJECT_NOT_FOUND | 404 | 对象不属于当前可访问账本或不存在 |
| LEDGER_INVALID_STATE | 409 | 当前账本生命周期不允许该操作 |
| LEDGER_ARCHIVED | 409 | 对 archived 账本执行被禁止的写操作 |
| LEDGER_VERSION_CONFLICT | 409 | `If-Match` 与当前账本 version 不同 |
| LEDGER_MEMBER_LIMIT_REACHED | 409 | 账本已达到 Task50 两人上限 |
| LEDGER_OWNER_TRANSFER_REQUIRED | 409 | Owner 离开前必须先移交所有权 |
| LEDGER_OWNER_INVARIANT_VIOLATION | 409 | 操作会产生零 Owner 或多 Owner |
| LEDGER_READY_IMPORT_EXISTS | 409 | 未过期 ready 导入批次阻止归档 |

前端必须按错误码提供刷新、移交、完成/放弃导入等可执行动作，不能解析 SQLite 文案。
