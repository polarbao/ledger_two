# Task53.4C 完成与 Task53 后续准备度复核

日期：2026-07-20
结论：Task53.4C 本地实现完成；下一实现切片为 Task53U，随后执行 Task53.5；冻结任务树没有 Task53.6

## 1. Task53.4C 完成范围

1. metadata 列表新增 `rule_reference_count`，只统计当前账本 active import rule 对分类、账户和标签的去重引用。
2. import rule DTO 新增稳定排序的 `stale_reference_ids`、`is_stale`、`committed_hit_count` 和 `last_committed_hit_at`。
3. 命中指标只读取当前账本 committed batch 中 status=`imported` 的行；ready、skipped 和跨账本数据不计入。
4. stale rule 不进入分类候选；恢复在单一写事务内复核引用与 manual conflict、更新状态并写审计，避免 TOCTOU，失败时状态与审计全部回滚。
5. 归档 `expense_other` / `income_other` 必须提供同账本、active、同类型、非自身且无 system key 的替代分类。
6. system key 清除、转移、旧分类归档和脱敏审计处于同一事务；失败全部回滚。
7. 历史 transaction、import rule result、metadata profile version、金额和 import hash 不改写；不新增 schema 23。
8. archive handler 严格拒绝未知字段、顶层 `null` 与尾随 JSON；metadata API client、前端类型、正式/增量 OpenAPI、API Inventory 和匿名 expected 已同步。

## 2. 验证证据

| Layer | Command/check | Result |
|---|---|---|
| Backend targeted | `go test ./internal/metadata ./internal/importer -count=1` | pass |
| Backend full | `go test ./... -count=1` | pass |
| Backend static | `go vet ./...` | pass |
| Backend build | `go build ./cmd/server` | pass |
| Frontend tests | `npm test -- --run` | 39 files / 150 tests pass |
| Frontend lint | `npm run lint` | pass |
| Frontend build | `npm run build` | pass；仅保留既有主 chunk 大于 500 kB 提示 |
| Contracts | 两份 OpenAPI YAML 解析与本地 schema `$ref` 检查 | pass |
| Fixtures | `fallback-replaced.json`、`rule-stale.json` JSON 解析 | pass |
| Independent review | stale restore TOCTOU、严格 JSON 与 draft required 差异复审 | 首轮问题已修复，复审结论非阻断、可提交 |
| Repository | `git diff --check` | pass |

Windows 默认 `D:\Program Files Tools\w64devkit\bin\gcc.exe` 生成 COFF bigobj，当前 Go cgo 解析失败；验证改用已安装的 Qt MinGW 13.1 GCC。该问题属于本机编译器工具链，不是 Task53.4C 代码失败，后续本机 Go 验证应继续显式设置 `CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe`，或单独统一开发环境工具链。

## 3. 后续任务准备度

### Task53U

准备阶段完整，可以直接开发：

1. 最终后端 DTO、错误码、正式 OpenAPI 和前端 API/type 已冻结。
2. `task53-frame-manifest.json` 已登记 15 个 status=`required` Frame，`reuse-evidence.md` 已登记组件复用边界。
3. UI 流程、状态矩阵、375/390/430/1440 与 Fresh Light/Dark Glass 验收要求已存在。
4. 未完成项属于 Task53U 实现本身：页面、交互、组件测试、浏览器截图和 generated review artifact；不能把结构准备误写为视觉验收完成。

### Task53.5

准备资产完整，但执行门禁尚未满足：

1. 独立 38092 compose/env、schema 21 -> 22 守恒、模式循环、发布指标、配对回滚脚本和 RC 验收模板均已存在。
2. 必须等待 Task53U 完成、固定候选镜像生成和用户明确部署授权。
3. 当前没有启动 WSL 38092、没有执行 migration 022、没有修改真实账单，也没有部署 NAS。

### Task53.6 判断

当前 PRD、Tech 26/28、Task53 详细计划和发布清单均没有定义 Task53.6。UI 收口由 Task53U 负责，环境与发布收口由 Task53.5 负责；另建 Task53.6 会制造重复责任和并行事实源，因此不新增该任务。

冻结顺序为：

```text
Task53.4C complete -> Task53U -> Task53.5 -> Task51P.1 evidence review
```

Task53.5 关闭后只回到 Task51P.1 证据评审。当前有效真实目标小组/完整工作流证据仍为 0/0，Task51P.2-P.6 和 Task51 代码未准入。

## 4. 下一步

直接进入 Task53U，先完成分类摘要/筛选与行级解释，再完成批量、学习、规则管理、基础包和兜底替代 UI；每个 UI 切片同步组件测试与本地审阅证据。Task53U 全部关闭后，才执行 Task53.5 独立 staging 验收。
