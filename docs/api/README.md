# API 契约文档目录

状态：Foundation before v1.1 冻结资料  
适用阶段：Task34 API 契约与 OpenAPI

本目录记录 LedgerTwo 当前实际 API、未来稳定 API 契约和 OpenAPI 草案。

## 文件列表

```text
API_INVENTORY.md  当前 router 实际暴露接口清单、认证要求、账本要求和稳定性标记
openapi.yaml      OpenAPI 草案，覆盖当前核心 API 路径、通用响应和主要请求 DTO
```

## 使用规则

1. 新增、删除或修改 API 路径前，必须先更新 `API_INVENTORY.md`。
2. 新增稳定业务接口前，必须同步更新 `openapi.yaml`。
3. 金额字段统一使用整数分，命名为 `*_cents`。
4. 时间字段统一使用 ISO8601 字符串。
5. 失败响应统一为 `{ "success": false, "error": { "code", "message", "details" } }`。
6. Foundation 阶段实际路径仍为 `/api/...`，文档可同时标注未来 `/api/v1/...` 目标路径，但不得把未实现 alias 描述为已上线。

## 稳定性标记

| 标记 | 含义 |
|---|---|
| stable | v1.1 可以冻结，允许前端和未来移动端长期依赖 |
| transitional | 当前可用，但存在兼容 fallback、字段命名或错误码待治理 |
| deprecated | 历史兼容或存在风险，后续应迁移或关闭 |
| internal | 健康检查、静态文件等非业务客户端契约 |
