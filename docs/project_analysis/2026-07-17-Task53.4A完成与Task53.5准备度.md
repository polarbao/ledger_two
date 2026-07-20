# Task53.4A 完成与 Task53.5 准备度复核

日期：2026-07-17<br>
结论：Task53.4A 已完成本地实现；Task53.5 准备阶段已补齐，执行准入尚未满足<br>
部署边界：未启动 WSL/Docker、未修改真实账单、未部署 NAS

## 1. Task53.4A delivered

1. 新增 Owner-only `POST /api/imports/{batchID}/rows/bulk-adjust`，只接受 ready、未过期、当前账本批次。
2. `accept_suggestions` 只消费行上已持久化的建议；`apply_values` 要求分类、显式可空账户和完整标签集。
3. 请求冻结 1-500 个唯一行 ID、最多 8 个唯一标签；invalid/duplicate/skipped 不修改，conflict 和行级错误独立返回。
4. 支出/收入分类错配、stale metadata、缺失行按行返回；顶层 payload、权限、batch 和 apply metadata 错误在写事务前失败。
5. 成功行写 selected 值、adjusted/bulk/high 和固定原因，原 suggested、suggestion reason 与 matched rule 快照继续保留。
6. repository 在单事务内复核 batch updated_at、行快照和 active metadata；SQL、并发或审计失败不留下半行更新。
7. 一次有效请求只写一条 ID/count-only `import_bulk_adjust` 审计，不创建 transaction 或 learned rule。
8. 前端只同步 DTO/API client，未提前实现 Task53U；正式 OpenAPI、草案状态和 API Inventory 已同步。

## 2. Verification evidence

本提交已完成：

```text
backend: go test ./... -count=1
backend: go vet ./...
backend: go build ./cmd/server
frontend: npm run lint
frontend: npm test -- --run (38 files / 148 tests)
frontend: npm run build
contract: formal/draft OpenAPI and compose YAML parse; formal $ref missing=0
deploy scripts: w64devkit sh -n (4/4)
repository: git diff --check
```

Task53.4A 测试覆盖：

- payload、显式 null account、重复行 ID、8 标签上限；
- persisted suggestion、manual overwrite、分类类型错配；
- invalid/duplicate/skipped/conflict/stale/missing 的部分成功结果；
- 请求顺序、summary、单审计、隐私字段、无 transaction/rule；
- audit 故障注入后的整次回滚。

当前 Windows shell 无 Docker/sqlite3 CLI，系统 `bash` 还受旧 WSL VHD 路径故障影响；本轮使用 w64devkit `sh -n` 完成四个脚本的静态语法检查，并完成 YAML/OpenAPI 结构解析，但不能诚实声称 compose runtime、38092 或浏览器已验证。

## 3. Task53.5 preparation audit

复核前的实际缺口是：Task53 专用 compose/env 已存在，但通用 `verify-staging.sh` 和 `rollback-staging.sh` 仍硬编码 Task50 tag、schema 21 和 38091/旧回滚链，不能直接用于 Task53。

现已补齐：

| Asset | Prepared behavior | Runtime state |
|---|---|---|
| `verify-task53-staging.sh` | schema 21/22、固定 Task53 tag、38092、health mode、守恒/import hash、备份 | not run |
| `verify-task53-mode-cycle.sh` | off -> suggest -> graded -> suggest -> off 与账务不变量 | not run |
| `check-task53-release-metrics.sh` | committed learned match 修正率与最小样本门禁 | not run |
| `rollback-task53-staging.sh` | schema 21 backup + fixed Task50.6 image paired rollback | not run |
| Task53 acceptance template | 自动化、功能、浏览器、指标、回滚和最终决策 | not run |

结论：Task53.5 的“准备阶段”完整，不需要再开一个规划会话；但“执行阶段”仍被 Task53.4B、Task53.4C、Task53U、候选镜像和用户部署授权共同阻断。

## 4. Next order

1. Task53.4B：explicit learn、UUIDv5 幂等、manual conflict、archived learned restore 和双事务证明。
2. Task53.4C：rule lifecycle/stale/reference count/committed hit 与 fallback system_key 替代事务。
3. Task53U：基于最终 DTO 完成 Fresh Light/Dark、required Frame、响应式与无障碍实现。
4. Task53.5：全量质量门禁、固定镜像、独立 38092、schema 21 -> 22、模式循环、浏览器和配对回滚。
5. Task53.5 关闭后只回到 Task51P.1 证据评审；当前 0/0 不授权 Task51 代码。
