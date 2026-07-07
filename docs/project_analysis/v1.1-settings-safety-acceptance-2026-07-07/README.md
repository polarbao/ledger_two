# v1.1 设置路径与数据安全验收记录

日期：2026-07-07  
环境：本机 WSL2 Docker，`http://localhost:38088`  
健康检查：`version=1.1.0-rc`、`schema_version=12`、`db=ok`  
证据文件：`settings-safety-metrics.json`

## 1. 验收范围

本轮覆盖 v1.1 冻结前仍缺少深度验收的设置路径和数据安全路径：

1. 分类、标签、账户新增、归档、恢复和 editor 拒绝创建。
2. 模板创建、归档、恢复。
3. 周期账单 pending 提醒、跳过本期、确认入账和空提醒列表。
4. 手动备份创建、下载、系统诊断和非 owner 拒绝访问。
5. 附件上传、private 账单关联、受控读取、非授权成员拒绝读取和裸 `/uploads` 拒绝访问。

## 2. 发现并修复的问题

### 2.1 空周期提醒返回 `null`

问题：

- `GET /api/recurring-reminders/` 在无 pending 提醒时返回 `data: null`。
- 前端类型期望数组，冻结前应稳定返回 `[]`。

处理：

- `backend/internal/transaction/service.go` 将响应切片初始化为非 nil。
- `backend/internal/http/handler/recurring_test.go` 增加确认后空列表不为 null 的回归断言。

### 2.2 editor 可执行手动备份

问题：

- `editor` 调用 `POST /api/admin/backup` 返回 200。
- v1.1 PRD 中备份恢复应仅 owner 可用。

处理：

- `backend/internal/safety/service.go` 对 `ManualBackup`、`RestoreBackup` 增加 owner 校验。
- `backend/internal/safety/handler.go` 对备份列表和下载增加 owner 校验。
- `backend/internal/http/router/rbac_acceptance_test.go` 增加备份创建、列表、下载 owner-only 验收测试。

## 3. 验收结论

通过：

- owner 登录成功，editor 登录成功。
- 分类、标签、账户归档后 `is_archived=true`，恢复后 `is_archived=false`。
- editor 创建分类、标签、账户均返回 403。
- 模板归档恢复闭环通过。
- 周期账单跳过本期不生成账单，确认后生成真实账单。
- 空周期提醒列表返回数组，数量为 0。
- owner 可创建并下载手动备份，诊断数据库和四类存储目录状态均为 `ok`。
- editor 访问诊断、创建备份、查看备份列表、下载备份均返回 403。
- private 附件 owner 可读取，editor 读取返回 404，裸 `/uploads` 返回 404。

仍未覆盖：

- NAS 地址下同等浏览器内复核。
- 恢复备份的人工停机覆盖动作未执行，仅验证了备份创建与下载。

## 4. 验证命令

已运行：

```bash
wsl -u root sh -lc "cd /mnt/e/__Code/__Prj/ledge_two/ledger_two && docker compose up -d --build"
wsl -u root sh -lc "cd /mnt/e/__Code/__Prj/ledge_two/ledger_two && docker build --target backend-builder -t ledger-two-backend-test -f deploy/docker/Dockerfile . && docker run --rm ledger-two-backend-test go test ./internal/http/handler -run TestRecurringBilling -count=1 && docker run --rm ledger-two-backend-test go test ./internal/http/router -run 'TestRBACAcceptance(BackupEndpointsOwnerOnly|DiagnosticsOwnerOnlyAndSanitized|PrivateAttachmentCannotBypassVisibility)' -count=1"
python API 验收脚本，输出 `settings-safety-metrics.json`
```

未通过/未采用：

```bash
go test ./internal/http/handler -run TestRecurringBilling -count=1
```

Windows 原生 Go 测试失败于 `github.com/mattn/go-sqlite3` cgo 对象解析：`cgo: cannot parse gcc output ... as ELF, Mach-O, PE, XCOFF object`。已改用项目 Dockerfile 的 `backend-builder` 环境完成相关测试。
