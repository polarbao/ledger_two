# Task53.4B 完成与 Task53.5 准备度复核

日期：2026-07-20<br>
结论：Task53.4B 本地实现与契约联动完成；Task53.5 准备阶段仍完整，执行准入尚未满足<br>
部署边界：未启动 WSL/Docker，未修改真实账单，未部署 NAS

## 1. Task53.4B delivered

1. 新增 Owner-only `POST /api/imports/{batchID}/rows/{rowID}/learn`，只接受 ready、未过期、当前账本且已另行保存的 manual/bulk 行。
2. 服务端只读取最终分类和完整标签；不接受客户端分类值，不学习账户、可见性、金额、备注或其他账单原文。
3. 固定 namespace、ledger、source scope 和规范化商户生成 UUIDv5；重复学习更新同一规则，归档规则显式学习返回 `restored`。
4. current_source 与 all_sources 严格隔离；同范围 active manual `merchant_equals` 返回 `CLASSIFICATION_CONFLICT`，不覆盖手工规则。
5. 规则写入和 `import_rule_learn` 脱敏审计同事务；审计失败回滚规则，但此前独立完成的行保存保持不变。
6. rule DTO/CRUD 已返回 origin/source_type/apply_mode/confidence；manual create 固定 manual/high，learned identity 字段不可编辑。
7. classification mode off 的 Task49 兼容路径只使用 manual rule 并遵守 source_type，不会提前启用 learned auto。

## 2. Automated coverage

Task53.4B 定向测试覆盖：

- UUIDv5 与重复学习幂等；
- current/all source scope 和 manual conflict；
- archived learned restore 与可编辑设置保留；
- 未保存行、空商户、跨账本、归档 metadata；
- 分类类型错配、标签上限和并发行快照变化；
- 审计故障回滚、行保存/learn 双事务不变量；
- HTTP unknown-field 拒绝和前端 API payload。

2026-07-20 提交前本地验证结果：

- `go test ./... -count=1`：通过；
- `go vet ./...`：通过；
- `go build ./cmd/server`：通过；
- `npm run lint`：通过；
- `npm test -- --run`：38 个测试文件、149 个测试通过；
- `npm run build`：通过，仅保留既有的大于 500 kB chunk 提示；
- 正式与 v1.3 草案 OpenAPI：共 81 个 path、449 个内部引用，YAML 解析、引用解析和 path 参数一致性通过；
- category-tag 的 5 个 JSON 期望 fixture 可解析；4 个 Task53 脚本均通过 `sh -n`。

本文不把尚未执行的 WSL、38092、浏览器或 NAS 验收写成已完成。

## 3. Verification evidence

本地已执行并通过：

```text
backend: go test ./... -count=1
backend: go vet ./...
backend: go build ./cmd/server
frontend: npm run lint
frontend: npm test -- --run (38 files / 149 tests)
frontend: npm run build
contract: formal/draft OpenAPI YAML parse and local $ref check (missing=0)
fixture: category-tag expected JSON parse
repository: git diff --check
```

前端构建仅保留既有大于 500 kB chunk 警告，不是本切片新增失败。未执行 WSL、Docker、38092、浏览器或 NAS 验收。

## 4. Task53.5 preparation audit

现有专用资产与 Task53.4B 契约相容：

| Asset | 53.4B dependency | Preparation state |
|---|---|---|
| `verify-task53-staging.sh` | schema 22、固定镜像、38092、模式 health | complete / not run |
| `verify-task53-mode-cycle.sh` | off/suggest/graded 回退 | complete / not run |
| `check-task53-release-metrics.sh` | origin=learned、auto/high、committed/imported match | complete / not run |
| `rollback-task53-staging.sh` | schema 21 配对备份和固定 Task50.6 镜像 | complete / not run |
| RC acceptance template | explicit learn/manual conflict 功能矩阵 | complete / not run |

新增匿名 `learn-created.json` 后，Task53.5 不存在需要再开准备会话的 53.4B 缺口。执行仍被以下准入条件阻断：

1. Task53.4C metadata safeguard 未完成；
2. Task53U 页面、响应式、无障碍和视觉证据未完成；
3. 尚未生成固定 Task53 candidate image；
4. 本轮没有 WSL/38092 或 NAS 部署授权。

## 5. Next order

1. Task53.4C：rule reference count、stale、committed hit 和 fallback system_key 替代事务。
2. Task53U：基于最终 53.4 DTO 实现 Fresh Light/Dark、required Frame、响应式和无障碍。
3. Task53.5：全量质量门禁、固定镜像、隔离 schema 21 -> 22、模式循环、浏览器与配对回滚。
4. Task53.5 关闭后回到 Task51P.1 证据评审，不自动准入 Task51 代码。
