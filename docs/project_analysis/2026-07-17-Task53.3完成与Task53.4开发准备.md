# Task53.3 完成与 Task53.4 开发准备评审

状态：Task53.3 本地实现与全量门禁完成；Task53.4 准备阶段关闭<br>
日期：2026-07-17<br>
部署状态：未更新 WSL/NAS，未读取或修改真实账单

## 1. Task53.3 delivered scope

1. preview 按 `off/suggest/graded` 接入确定性分类器，并在 duplicate 判定后只处理 eligible 行。
2. 行级 status、confidence、source、reason、matched rules 和 suggested 投影写入 schema 22；batch 返回服务端聚合 summary。
3. high auto、suggestion、fallback、conflict、manual/bulk 保护和 commit 快照语义已形成集成测试。
4. 新增 `POST /api/imports/{batchID}/reclassify`：默认 dry-run，只处理 ready/未过期批次，不覆盖 manual/bulk，不创建 transaction。
5. execute 在单事务内校验 batch 快照、并发人工调整和到期状态，更新分类快照并写脱敏审计；规则变化不会隐式影响 commit。
6. 前端只同步分类 DTO、summary、reclassify API 和 contract test，不开始 Task53U 页面。

## 2. Validation evidence

当前已通过：

1. Qt MinGW CGO 环境下 `go test ./... -count=1`、`go vet ./...` 和 server build。
2. `npm run lint`、38 个测试文件/148 项测试和 production build。
3. 正式 OpenAPI 与 Task53 draft 的 YAML、`$ref`、path parameter，以及 category-tag expected JSON 校验。
4. `git diff --check`。

production build 仅保留既有 JavaScript chunk size warning。未运行 Docker/WSL/NAS、浏览器和真实账单验收；这些边界不属于 Task53.3 原子提交，仍归 Task53U/Task53.5。

## 3. Task53.4 preparation audit

此前材料存在四个实际缺口：bulk 草案缺少 account/summary、learn 请求与“只读取已保存行”原则冲突、规则 lifecycle/stale/命中指标未冻结、兜底替代没有 schema 22 可执行映射。本轮已经关闭：

1. bulk-adjust 冻结 row 上限、两种 action、部分成功结构、bulk 状态、事务回滚、审计脱敏和不创建规则/transaction。
2. learn 冻结为只接收 `source_scope`，服务端读取已保存分类/标签；不学习账户/可见性，以 UUIDv5 tuple 保证幂等，不覆盖 manual rule。
3. rule DTO 冻结 origin/source/apply_mode/stale/reference/committed-hit；命中数从 committed import items 动态聚合，不新增 migration 023。
4. fallback replacement 复用现有 archive 路径，在事务内转移 `expense_other/income_other` system_key；历史账单和规则结果不改写。
5. 稳定错误码、Owner/ledger/ready 边界、审计 payload、故障注入和 TDD 顺序已写入 Tech 28 与 OpenAPI draft。

结论：Task53.4 不需要额外准备会话。Task53.3 原子提交后，可直接进入 `Task53.4A bulk-adjust failing tests`。

## 4. Task53.4 execution order

| Slice | Scope | Completion gate |
|---|---|---|
| 53.4A | bulk-adjust model/service/repository/handler/route | partial result、回滚、保护和审计测试 |
| 53.4B | learn + rule DTO/lifecycle/committed hits | 幂等、manual conflict、双事务、跨账本测试 |
| 53.4C | metadata reference count/stale + fallback transfer | system_key 转移故障注入、历史数据不变测试 |
| Sync | frontend types/API、正式 OpenAPI、Inventory、Fixture | 不实现 UI 页面，全量质量门通过 |

## 5. Remaining risks and boundaries

1. `IMPORT_CLASSIFICATION_MODE` 默认仍为 off；Task53.3 本地完成不等于 WSL/NAS 已启用自动分类。
2. Task53.4 必须继续使用 schema 22，不得为命中计数临时新增 migration 或把 preview 次数当真实复用指标。
3. bulk 和 learn 必须保持独立，任何批量动作都不得隐式生成长期规则。
4. Task53U 仍等待 53.4 DTO/错误码落盘和本地视觉审阅稿；当前不能开始页面实现。
5. Task51 仍由真实小组证据门禁控制，不因 Task53 进展自动进入代码阶段。
