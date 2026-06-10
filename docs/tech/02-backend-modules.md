# 技术：后端模块设计

## 1. 后端分层

- handler：HTTP 请求解析和响应。
- service：业务规则。
- repository：数据库访问。
- dto：请求与响应结构。
- middleware：认证、日志、错误处理。

## 2. 模块目录

```text
internal/auth
internal/user
internal/transaction
internal/split
internal/settlement
internal/report
internal/category
internal/tag
internal/account
internal/export
internal/backup
internal/audit
```

## 3. 关键规则

- 金额使用 int64 cents。
- service 层负责业务规则。
- repository 不写业务判断。
- handler 不直接操作数据库。
- 错误统一转换为 API error code。

## 4. v0.2 重点

- 审计日志完整化。
- 备份导出。
- 权限判断复查。
- 结算服务测试。
