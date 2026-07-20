# 技术：API 契约、OpenAPI 与错误码规范

状态：供审核  
目标：稳定 Web/PWA/未来移动端共用的 API 合同，减少 Codex/Gemini 误写字段和错误处理不一致。

## 1. API 版本策略

当前实际接口路径为：

```text
/api/...
```

Foundation 阶段建议：

1. 保留 `/api` 作为 v1 compatibility。
2. 新增 `/api/v1` alias 或在文档中定义 v1 正式路径。
3. 不在基础框架阶段做破坏性迁移。
4. 新 API 文档统一按 `/api/v1` 描述，同时列出兼容路径。

## 2. 响应格式

### 2.1 成功响应

```json
{
  "success": true,
  "data": {}
}
```

### 2.2 错误响应

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

## 3. 错误码规范

| HTTP | Code | 场景 |
|---:|---|---|
| 400 | BAD_REQUEST | 请求结构错误 |
| 400 | VALIDATION_ERROR | 参数校验失败 |
| 401 | UNAUTHORIZED | 未登录或登录失效 |
| 403 | FORBIDDEN | 无权限 |
| 404 | NOT_FOUND | 资源不存在或不可见 |
| 409 | CONFLICT | 状态冲突、重复邀请、重复名称 |
| 500 | INTERNAL_ERROR | 内部错误 |
| 503 | SERVICE_UNAVAILABLE | 存储不可用、数据库不可用 |

业务错误码继续细化，但必须归入稳定枚举。

## 4. DTO 命名规范

1. 金额统一 `*_cents`。
2. 时间统一 ISO8601 字符串。
3. ID 统一 string UUID。
4. 布尔值以 `is_` / `has_` 开头。
5. 列表响应如果需要分页，统一：

```json
{
  "items": [],
  "page": 1,
  "page_size": 20,
  "total": 100
}
```

6. 不返回数据库内部字段名，除非已经作为 API 合同稳定。

## 5. OpenAPI 输出要求

建议新增：

```text
docs/api/openapi.yaml
docs/api/README.md
```

OpenAPI 必须覆盖：

- auth。
- init。
- ledgers。
- ledger members。
- transactions。
- shared expenses。
- settlements。
- categories。
- tags。
- accounts。
- templates。
- recurring rules。
- import/export。
- backup/restore。
- attachments。
- reports/dashboard。

## 6. 前端类型策略

短期：手写类型继续保留，但必须与 API 文档同步。

中期：从 OpenAPI 生成：

```text
frontend/src/api/generated/
```

前端不得直接依赖数据库字段或拼接不稳定接口。

## 7. Query Key 规范

TanStack Query key 必须能完整描述数据来源。建议：

```ts
export const queryKeys = {
  me: ['auth', 'me'] as const,
  ledgers: ['ledgers'] as const,
  dashboard: (ledgerId: string, month: string) => ['dashboard', ledgerId, month] as const,
  transactions: (ledgerId: string, filters: TransactionFilters) => ['transactions', ledgerId, filters] as const,
  categories: (ledgerId: string) => ['categories', ledgerId] as const,
  tags: (ledgerId: string) => ['tags', ledgerId] as const,
  accounts: (ledgerId: string) => ['accounts', ledgerId] as const,
};
```

## 8. 验收标准

1. API 错误结构一致。
2. OpenAPI 草案存在。
3. 前端类型和 API DTO 对齐。
4. 分页、金额、时间、错误码规则被写入 docs。
5. 新任务不得新增未登记 API。
