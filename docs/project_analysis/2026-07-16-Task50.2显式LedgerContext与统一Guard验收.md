# Task50.2 显式 LedgerContext 与统一 Guard 验收记录

日期：2026-07-16
任务：Task50.2
结论：通过，可进入 Task50.3；未开放生命周期/成员 mutation API，未部署 WSL staging 或 NAS

## 1. 实施范围

1. 账本内路由统一要求显式 `X-Ledger-Id`，账本成员路径同时校验 Path/Header 一致性。
2. 集中实现 `LifecyclePolicy`、`RolePolicy` 和 `InstancePolicy`；archived 账本允许读取及 Owner/Editor 导出，禁止业务写入。
3. 实例管理员与账本 Owner 分离；实例诊断、整库备份和恢复不消费账本 header，并写入实例级审计。
4. 清理 dashboard、transaction、settlement、reports、metadata、shared expense 和 safety 等生产路径的首账本 fallback。
5. 交易、模板、导入批次及明细更新等对象访问同时约束对象 ID 和 `ledgerID`，跨账本访问不暴露对象存在性。
6. 前端 API client 将账本请求默认为 required scope；无 active ledger 时在 fetch 前返回 `LEDGER_REQUIRED`，全局请求必须显式声明 `ledgerScope: 'none'`。
7. `/auth/me` 不再返回隐式当前账本，只返回用户信息及 `instance_admin`，设置页实例运维能力不再从账本角色推导。
8. 账本导出不再携带全局 users/app_settings；dashboard/report 成员字典、交易标签和 split 均通过父对象与当前账本约束。
9. 归档账本读取周期提醒不会执行懒生成或推进规则日期；导入预览调整、提交和旧版导入会拒绝其他账本的分类、账户、标签和付款人引用。

## 2. 错误与权限契约

| 场景 | HTTP | 错误码/结果 |
|---|---:|---|
| 账本内请求缺少 header | 400 | `LEDGER_REQUIRED` |
| Path/Header 不一致 | 400 | `LEDGER_CONTEXT_MISMATCH` |
| 非成员或角色不足 | 403 | `LEDGER_ACCESS_DENIED` |
| archived 账本业务写入 | 409 | `LEDGER_ARCHIVED` |
| A 账本对象从 B 账本读取 | 404 | `LEDGER_OBJECT_NOT_FOUND` |
| 非实例管理员访问整库运维 | 403 | `INSTANCE_ADMIN_REQUIRED` |
| 全局身份/初始化/账本列表 | 2xx | 忽略账本 header，不建立账本上下文 |

生命周期 Guard 先于角色 Guard 执行，因此 archived 写入稳定返回 409，不因调用者角色不同泄漏不一致结果。Viewer 可读取允许的账本数据，但不能执行写入；Owner/Editor 可按冻结契约导出 archived 账本。

## 3. 自动化验收

覆盖范围：

- `T50-API-005~008`：缺 header、非成员、Path/Header mismatch、稳定错误码与全局路由。
- `T50-ISO-001~010` 所涉及模块的代表性边界：交易、结算、报表、元数据、模板、导入、附件/安全及对象级 A/B 隔离。
- middleware、router、service、repository 和 API client 的正常、拒绝和兼容回归。
- 前端无 active ledger 不发送、显式 path ledger、全局请求不携带 header、实例管理员 UI 权限来源。

Task50.2 的目标是建立统一 Guard、对象约束和高风险回归样本，不在本任务宣称已逐行执行 Fixture 32 的完整 A/B 组合矩阵；全模块、全角色、全可见性的穷举矩阵仍由 Task50.6 发布门禁执行。

执行结果：

```text
CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go test ./... -count=1
PASS：backend 全部 package

CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go vet ./...
PASS

CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go build -o %TEMP%\ledger-two-task50-server.exe ./cmd/server
PASS

npm run lint
PASS

npm test -- --run
PASS：28 files / 109 tests

npm run build
PASS（保留既有大 chunk 告警，不属于 Task50.2 阻断项）
```

生产代码 fallback 搜索仅保留显式 header 上下文读取 helper；前端 `AppInitGuard`/`AppShell` 的首个账本选择属于 Task50.4 冻结的 active-ledger 状态机改造，不会绕过本次 API client 的 fail-closed 边界。

## 4. 环境与数据边界

1. 自动化测试仅使用内存库或临时库；未连接真实账单和 production 数据。
2. WSL staging 继续保持 v1.2/schema 19；NAS 未执行 migration 020/021，也未更新镜像。
3. Task50.2 不增加 migration，不改变 schema 21，不开放 Task50.3 的生命周期或成员 mutation 路由。
4. 回退时只回退 Task50.2 应用提交并保留 schema 21；该回退镜像不得部署到已依赖显式上下文契约的 production。

## 5. 下一任务准入

Task50.3 所需 PRD 31、Tech 25、OpenAPI v1.3、Fixture 32 和 Task50 详细实施卡已具备；Task50.1 提供数据库不变量与 repository 原语，Task50.2 提供显式上下文和统一 Guard。下一步应按 50.3A 生命周期、50.3B 成员不变量、50.3C 实例运维串行实现，并在写代码前完成 API/错误码/事务边界的逐项映射复核。
