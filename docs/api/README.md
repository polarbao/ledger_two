# API 契约文档目录

状态：Foundation before v1.1 冻结资料  
适用阶段：Task34 API 契约与 OpenAPI

本目录记录 LedgerTwo 当前实际 API、未来稳定 API 契约和 OpenAPI 草案。

## 文件列表

```text
API_INVENTORY.md  当前 router 实际暴露接口清单、认证要求、账本要求和稳定性标记
openapi.yaml      OpenAPI 草案，覆盖当前核心 API 路径、通用响应和主要请求 DTO
openapi-v1.2-import-draft.yaml v1.2 导入模块已实现补充契约；文件名保留 draft 以避免破坏既有引用
openapi-v1.3-ledger-draft.yaml Task50 多账本生命周期、成员、实例运维与 Task53.1 新账本 profile 补充契约；已实现部分以 inventory 为准
openapi-v1.3-category-tag-draft.yaml Task53 分类、标签、默认元数据与导入分级自动化契约；Task53.1-Task53.4B 已实现，Task53.4C stale/命中指标/兜底替代仍是草案
API_CONVENTIONS.md 错误码、分页、筛选、排序、金额、时间和 Ledger Context 规范
```

## 使用规则

1. 新增、删除或修改 API 路径前，必须先更新 `API_INVENTORY.md`。
2. 新增稳定业务接口前，必须同步更新 `openapi.yaml`。
3. 新增错误码、筛选字段、排序字段或分页行为前，必须同步更新 `API_CONVENTIONS.md`。
4. 金额字段统一使用整数分，命名为 `*_cents`。
5. 时间字段统一使用 ISO8601 字符串。
6. 失败响应统一为 `{ "success": false, "error": { "code", "message", "details" } }`。
7. Foundation 阶段实际路径仍为 `/api/...`，文档可同时标注未来 `/api/v1/...` 目标路径，但不得把未实现 alias 描述为已上线。
8. v1.2 导入接口以 `openapi-v1.2-import-draft.yaml` 为已实现补充契约，冻结检查需与 `API_INVENTORY.md` 和 router 同步核对。
9. v1.3 多账本接口以 `openapi-v1.3-ledger-draft.yaml` 为冻结契约；当前实现状态必须与 `API_INVENTORY.md`、router 和验收测试交叉核对。
10. Task53.1 默认 profile 三个路径已经实现并纳入 inventory；Task53.2 只有内部分类基础，没有新增业务路径；其余 Task53 导入归类路径在对应任务完成前仍不得用于客户端生产调用。

## 稳定性标记

| 标记 | 含义 |
|---|---|
| stable | v1.1 可以冻结，允许前端和未来移动端长期依赖 |
| transitional | 当前可用，但存在兼容 fallback、字段命名或错误码待治理 |
| deprecated | 历史兼容或存在风险，后续应迁移或关闭 |
| internal | 健康检查、静态文件等非业务客户端契约 |
