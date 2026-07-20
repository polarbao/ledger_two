# Task50.1 Migration 与数据基础验收记录

日期：2026-07-15
任务：Task50.1
结论：通过，可进入 Task50.2；未授权部署 WSL staging 或 NAS

## 1. 实施范围

1. 新增 migration 020：账本 `status`、归档信息、`version`、审计角色、查询索引、唯一 Owner 索引和最多两名成员 trigger。
2. 新增 migration 021：`instance_admins`、`instance_audit_logs`、实例审计索引和确定性首位实例管理员回填。
3. `db.Init` 在迁移前执行 schema、`quick_check`、Owner/成员、角色和外键预检；升级后核对核心表和分账本业务指标守恒。
4. Ledger model/repository 增加 lifecycle、状态列表、按 ID 读取、成员/Owner 计数、实例管理员查询和 version 条件竞争原语。
5. 全新实例初始化事务同步登记首位实例管理员；没有提前开放 Task50.3 API，也没有修改前端。

## 2. 自动化验收

覆盖范围：

- 全新库 001 -> 021 与 schema 19 -> 21。
- schema 18、未版本化非空库、第三名成员、零 Owner、双 Owner、非法角色和外键异常的迁移拒绝。
- `quick_check`、确定性实例管理员回填、新字段、索引、trigger 和初始化事务。
- 核心表行数、分账本交易/结算数量与金额、成员、导入批次、审计、导入引用和附件引用守恒。
- lifecycle repository、归档元数据、成员计数、实例管理员和乐观并发 version 原语。
- 既有 RBAC 验收 Fixture 调整为双人成员模型，未放宽生产约束。

执行结果：

```text
CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go test ./... -count=1
PASS：backend 全部 package

CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go vet ./...
PASS

CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go build ./cmd/server
PASS
```

Windows 默认 w64devkit gcc 无法被当前 Go CGO 链正确识别，因此验收明确固定仓库既有可用的 Qt MinGW gcc；该问题不影响本次代码结论，但后续 Windows 本地测试应继续显式设置 `CC`。

## 3. 环境隔离

只读请求 `http://localhost:38088/api/healthz` 返回：

```text
deployment_channel=staging
version=1.2.0-rc
schema_version=19
db=ok
```

本任务未连接或迁移 WSL staging 数据库，未访问 NAS production，未导入真实账单，未生成或提交 development 数据库。migration 020/021 仅在自动化临时库中执行。

## 4. 数据与回滚结论

1. schema 19 数据不会被无条件升级：只有通过完整预检后才会先备份、再执行 020/021。
2. schema 版本不是 19 的既有数据库会被明确拒绝；空库和已达 schema 21 的数据库按冻结契约处理。
3. production 禁止 `goose down`。后续候选环境回滚必须恢复升级前完整备份，并让镜像与数据库版本成对回退。
4. Task50.2 只能建立显式 LedgerContext 与统一 Guard，不得提前混入生命周期 API、前端管理页或 NAS 部署。

## 5. 下一任务准入

Task50.2 所需 PRD、Tech 25、OpenAPI v1.3、Fixture 32 和详细任务卡已冻结，Task50.1 repository 原语已经提供。Task50.2 可以开始，但必须先补缺 header、非成员、Path/Header mismatch、archived 写入和角色矩阵失败测试，并在独立提交中关闭所有生产首账本 fallback。
