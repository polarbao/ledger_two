# Task50.3C 实例运维与契约收口验收记录

状态：已完成<br>
日期：2026-07-16<br>
目标 schema：21，仅独立 development/自动化环境<br>
下一任务：Task50.4 active-ledger、Query Cache 与 AppShell 状态机

## 1. Scope

本切片关闭 Task50.3 的实例运维子任务，复核并收口整库诊断、备份、备份列表/下载和恢复准备的独立实例管理员权限、实例审计、DTO、备份 key 安全和 OpenAPI/API Inventory。

未新增实例管理员授予页面、热恢复数据库、单账本物理备份、Task50.4 状态机、Task50.5 管理页面或 Task51 多人代码。

## 2. Permission boundary

1. `/api/admin/*` 继续位于全局认证路由，不解析或消费 `X-Ledger-Id`。
2. 每次请求由 middleware 与 safety service 分别从 `instance_admins` 读取权限，账本 Owner、Editor、Viewer 均不会自动获得整库运维能力。
3. 实例管理员即使不是目标账本成员，也可执行实例运维；但访问任一账本业务 API 仍返回 `LEDGER_ACCESS_DENIED`。
4. 实例操作不创建伪造 `ledger_id`，也不向 `audit_logs` 写入实例事件。

## 3. Instance audit

以下成功操作统一写入 `instance_audit_logs`：

| API | action |
|---|---|
| `GET /api/admin/diagnostics` | `system_diagnostics` |
| `POST /api/admin/backup` | `manual_database_backup` |
| `GET /api/admin/backups` | `list_database_backups` |
| `GET /api/admin/backups/{key}` | `download_database_backup` |
| `POST /api/admin/restore` | `prepare_database_restore` |

手动备份和恢复前置备份在审计失败时删除本次生成文件，不返回成功响应。

## 4. DTO and file safety

1. 手动备份返回完整 `BackupInfo`：`filename`、`size_bytes`、`created_at`。
2. 恢复准备返回 `filename`、`instructions`、`requires_downtime=true`，不会在 HTTP 请求中替换运行数据库。
3. 备份列表返回受管理目录内的安全相对 key，例如 `manual/backup_*.db`；前端按路径段编码下载 URL。
4. 服务端拒绝空 key、`.`、`..`、反斜杠、非法字符、非 `.db` 文件、目录、符号链接和受管理目录外路径。
5. 同前缀兄弟目录穿越测试已覆盖，恢复准备不会为非法目标创建前置备份或审计。

## 5. Contract closure

1. OpenAPI 已同步实例路由、完整诊断 DTO、BackupInfo、恢复准备 DTO、安全相对 key 和稳定错误码。
2. API Inventory 已完成实际 router 双向复核：生产账本内路由均标记为 `required`，实例路由统一为 `none`。
3. Frontend safety API 使用最终 DTO，备份下载 URL 保留嵌套相对 key，不再把整个 key 编码为单一路由段。
4. Task50.3A、3B、3C 均有独立验收记录，Task50.3 整体关闭并放行 Task50.4。

## 6. Verification

执行范围：

```text
Backend: Task50.3C router RED/GREEN 定向测试
Backend: safety handler 回归
Backend: go test ./... -count=1
Backend: go vet ./...
Backend: go build ./cmd/server
Frontend: safety API RED/GREEN 定向测试
Frontend: npm run lint
Frontend: npm test -- --run
Frontend: npm run build
OpenAPI: YAML parse、local $ref、Task50 路由/方法与实例 DTO 检查
Repository: git diff --check、文档引用和敏感文件审计
```

结果：

1. Backend 全包测试、`go vet` 和 server build 通过。
2. Frontend lint、31 个测试文件共 117 项测试和 production build 通过。
3. OpenAPI 共 18 条路径、160 个本地引用和 19 个 Task50/实例 method 检查通过；BackupInfo、恢复准备和诊断 DTO 与实现一致。
4. API Inventory 不再包含生产 `optional` 账本路由；工作树 diff 和敏感文件审计通过。
5. Frontend build 仍存在既有主 chunk 超过 500 kB 的非阻断警告，留待独立性能专项。

## 7. Environment

本切片未新增或修改 migration，未连接真实账单数据库，未迁移或部署 WSL staging/NAS。schema 21 继续限定在自动化与独立 Task50 development；production 禁止 goose down。
