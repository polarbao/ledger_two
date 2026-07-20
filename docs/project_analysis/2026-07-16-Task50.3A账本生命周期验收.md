# Task50.3A 账本生命周期验收记录

日期：2026-07-16
任务：Task50.3A
结论：通过，可进入 Task50.3B；Task50.3 整体尚未关闭，Task50.4 仍未放行
环境边界：自动化临时库与前端构建；未迁移 WSL staging、NAS 或真实数据库

## 1. 实施范围

1. 实现 `GET/POST /api/ledgers`、`GET/PATCH /api/ledgers/{id}`、归档预检、归档和恢复路由。
2. 创建、详情和所有生命周期 mutation 返回冻结格式 ETag；rename/archive/restore 强制解析对应账本的 `If-Match`。
3. 名称按 trim 后 1-60 个 Unicode 字符校验；允许同名和连续创建，不增加账号级任意账本上限。
4. 创建者成为 active/version 1 的唯一 Owner；create/rename/archive/restore 写账本审计并记录 `actor_role`。
5. 归档预检只读取 trusted settlement balance 与未过期 ready 批次，不改变账本 version、批次、结算或审计。
6. 归档事务以 version claim 作为首个写入；ready 阻断、未结清未确认或后续失败会回滚 version、过期批次收敛和审计。
7. 归档成功不会生成 settlement 或改写历史交易；归档审计保留新 status/version、未结清快照和 ready batch count。
8. 新增 `POST /api/imports/{batchID}/discard`：Owner 显式把 ready 批次置为 expired，保留预览行/hash，不创建 transaction，并写独立 `import_batch_discard` 审计。
9. settlement repository 增加 caller transaction 内的汇总读取，生命周期模块复用既有整数分余额公式，不复制一套计算口径。
10. 前端同步 Ledger lifecycle DTO、API、ETag helper 和 import discard DTO/API；未实现 Task50.4 状态机或 Task50.5 页面。

## 2. 冻结契约结果

| 场景 | 结果 |
|---|---|
| 列表默认值 | 默认 active，可显式 archived/all |
| 非成员读取详情 | 403 `LEDGER_ACCESS_DENIED` |
| 缺失/非法 If-Match | 400 `VALIDATION_ERROR` |
| 旧 version | 409 `LEDGER_VERSION_CONFLICT`，无持久化副作用 |
| archived 重命名 | 409 `LEDGER_ARCHIVED` |
| 未过期 ready 批次 | 409 `LEDGER_READY_IMPORT_EXISTS`，version/批次/审计不变 |
| 仅过期 ready 批次 | 归档成功时同事务收敛为 expired |
| 未结清但未确认 | 400 `VALIDATION_ERROR`，不创建 settlement |
| 显式 discard | status=expired；行/hash 保留；无 transaction |
| 恢复 | archived -> active，清空归档字段，version 只增加一次 |

## 3. 自动化验收

测试采用先失败、再实现、再全量回归的顺序，覆盖：

1. Unicode 名称、同名/数量、状态过滤、成员详情、ETag 严格解析和旧版本竞争。
2. Owner 防伪复核、归档/恢复状态、审计 actor role 与 archive after_json。
3. 预检只读、expired/future ready 混合回滚、显式 discard 的数据守恒。
4. 真实 settlement provider 在 caller transaction 中计算余额，归档不生成结算。
5. Router 的 201/200/400/403/409、Path/Header、If-Match、ETag 和 discard 路由。
6. 前端 lifecycle/import API 的 URL、method、ledger header、If-Match 和请求 DTO。

最终验证命令：

```powershell
Set-Location backend
$env:CGO_ENABLED='1'
$env:CC='C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe'
go test ./... -count=1
go vet ./...
go build -o "$env:TEMP\ledger-two-task50-3a.exe" ./cmd/server

Set-Location ../frontend
npm run lint
npm test -- --run
npm run build

Set-Location ..
git diff --check
```

执行结果：backend 全部 package 测试、vet、server build 通过；frontend 30 个测试文件/114 个测试、lint、production build 通过。Vite 仍报告既有大 chunk 告警，不属于本切片新增阻断。

## 4. 数据与回滚边界

1. 本切片不新增或修改 migration，schema 仍为 21。
2. 自动化用例只使用内存库或测试临时库；工作树不包含 `.db/.sqlite`、真实账单、上传、备份、密钥或环境文件。
3. WSL staging 继续保持 v1.2/schema 19，NAS 未执行 migration 020/021，也未更新镜像。
4. 回滚只回退 Task50.3A 应用提交并保留 schema 21；已经产生的归档状态必须通过恢复 API 或向前修复处理，禁止直接修改真实数据库。

## 5. 下一任务准入

Task50.3B 的 PRD、Tech、OpenAPI、Fixture、事务顺序、兼容 `PUT` 策略和测试清单均已具备。Task50.3A 已提供最终 Ledger DTO、ETag/version claim 和生命周期审计原语，因此下一步可直接测试先行实现成员与唯一 Owner 不变量；不得复用旧的非事务成员方法拼装正式 mutation。

Task50.3C 的 InstancePolicy、全局 admin 路由和交叉权限基础也已存在，但必须等待 Task50.3B 完成后串行进入。Task50.4 依赖 Task50.3 最终成员 DTO/错误码，当前仍保持条件准入。
