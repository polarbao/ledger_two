# Task50.3B 成员与 Owner 不变量验收记录

状态：已完成<br>
日期：2026-07-16<br>
目标 schema：21，仅独立开发与自动化环境<br>
下一任务：Task50.3C 实例运维与契约收口

## 1. Scope

本切片完成成员列表、添加、角色调整、移除、Owner 原子移交和 Editor/Viewer 主动离开，并保持最多两人、唯一 Owner、显式 ETag、历史账务不变和跨账本隔离。

未实现 Task50.4 active-ledger 状态机、Task50.5 完整管理页面、第三成员、邀请、通知或 Task53 分类标签代码。

## 2. API

1. `GET /api/ledgers/{id}/members` 返回 `ledger + members`、`joined_at` 和当前 ETag。
2. `POST /api/ledgers/{id}/members` 要求 active Owner、If-Match 和历史可见性确认，成功返回 201。
3. `PATCH /api/ledgers/{id}/members/{userId}` 是正式 editor/viewer 调角接口。
4. `PUT /api/ledgers/{id}/members/{userId}` 保留 deprecated 兼容入口，与 PATCH 共用 handler/service。
5. `DELETE /api/ledgers/{id}/members/{userId}` 只允许移除非 Owner。
6. `POST /api/ledgers/{id}/members/{userId}/transfer-owner` 原子完成旧 Owner -> Editor、新 Owner -> Owner。
7. `POST /api/ledgers/{id}/leave` 允许 Editor/Viewer 离开，响应携带提交后的 ETag；Owner 返回 `LEDGER_OWNER_TRANSFER_REQUIRED`。

## 3. Transaction and invariants

1. 每个 mutation 在同一事务内重新读取数据库角色和状态，不能信任请求上下文伪造的 Owner。
2. `If-Match` 只 claim 一次 ledger version；成员写入、Owner 移交和审计任一步失败均回滚 version。
3. migration 020 的两人 trigger 与唯一 Owner partial index 继续作为数据库兜底。
4. 角色通用接口拒绝 owner；Owner 移交只写一个 `ledger_owner_transfer` 审计事件。
5. 添加用户使用精确 username 且只接受 active 用户；历史可见性未确认、第三成员、旧版本和归档状态均稳定拒绝。
6. 删除成员关系不删除或改写交易、split、settlement、附件或审计。

## 4. Historical behavior

1. Dashboard 用户映射包含当前成员和当前账本对象实际引用的历史参与者，不返回无关全局用户。
2. Settlement 余额使用当前成员与 shared expense/split/settlement 引用的历史参与者；成员离开或被移除后，既有债务方向和整数分金额不丢失。
3. 已离开的用户因 membership 删除立即失去账本访问权，但历史 actor、payer、owner 和 split 仍可由有权成员解释。

## 5. Frontend contract

1. `ledger.api.ts` 所有成员 mutation 携带冻结格式 ETag，角色更新已切换到 PATCH。
2. 旧 LedgerSettings 适配新的 member snapshot DTO，并使用服务端返回 version 继续后续 mutation。
3. 添加成员增加明确历史可见性 checkbox；未勾选时不能提交。
4. Task50.5 的 Owner 移交、离开和完整生命周期页面仍未提前实现。

## 6. Verification

已执行：

```text
Backend: go test ./... -count=1
Backend: go vet ./...
Backend: go build ./cmd/server
Frontend: npm run lint
Frontend: npm test -- --run
Frontend: npm run build
OpenAPI: YAML parse、local $ref 与 7 个 member method 检查
```

结果：

1. Backend 全包测试通过。
2. Frontend 30 个测试文件、115 项测试通过。
3. Frontend production build 通过；仍有既有主 chunk 超过 500 kB 的非阻断警告。
4. OpenAPI 共 18 条路径，成员接口 7 个 method 与实现一致。

## 7. Environment

本切片未新增 migration，未连接真实账单数据库，未迁移或部署 WSL staging/NAS。schema 21 继续只用于自动化和独立 Task50 development；production 禁止 goose down。
