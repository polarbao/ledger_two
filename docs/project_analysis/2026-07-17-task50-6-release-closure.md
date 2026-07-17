# Task50.6 全模块与发布收口验收

状态：已完成  
验收日期：2026-07-17  
候选提交：`7984013`  
候选镜像：`ledger-two:1.3.0-rc-task50.6-7984013`  
机器可读证据：`evidence/task50-6/release-acceptance.json`

## 1. 结论

Task50.6 已完成 Fixture 32 要求的全模块隔离、历史可见性、schema 19 -> 21、独立 v1.3 staging、成对回滚、浏览器回归和正式契约收口。Task50.1-Task50.6 至此整体关闭。

本次只部署到本机 WSL2 的独立 staging：

```text
URL: http://127.0.0.1:38091
container: ledger-two-v13-staging
channel: staging
version: 1.3.0-rc
schema: 21
database: .runtime/v13-task50-6-staging/data/ledger.db
```

现有 `http://127.0.0.1:38088` 的 v1.2/schema 19 容器与数据目录保持不变。NAS staging/production 未连接、未复制、未迁移，也未执行 migration 020/021。

本机匿名验收账号为 `qa_owner` / `pass123`。它只存在于被 Git 和 Docker build context 同时排除的 `.runtime` 数据库，不是 NAS 或正式账号。

## 2. 全模块隔离

新增自动化按两个账本、同一用户多账本和成员替换三类上下文覆盖：

1. 交易、批量标签、shared expense 与 splits。
2. 分类、标签、账户和个人默认值。
3. 结算列表、余额、Dashboard 和四类报表。
4. 模板、周期规则和周期提醒。
5. 导入批次/行、规则、hash 与 transaction import refs。
6. 受保护附件与裸 `/uploads` 关闭。
7. CSV/JSON 导出与账本审计。
8. archived 写阻断、当前账本读取和跨账本对象 404。

成员替换后：

- former member 的 `private` 历史对新成员继续隐藏。
- `partner_readable`、`shared`、splits 和历史 settlement 继续可读。
- former member 的账务对象不被删除或改写，但成员关系删除后不再授权访问。
- 历史参与者展示资料只通过当前账本可见对象引用解析，不查询或导出全局用户。

## 3. JSON 数据包

`GET /api/export/full.json` 现为带 `manifest` 的 `ledger_two_ledger_export` v1 只读数据包，明确 `restorable=false`。数据段覆盖：

```text
ledger_members / users / categories / tags / accounts
transaction_defaults / transactions / transaction_tags / transaction_splits
settlements / transaction_templates / recurring_rules / recurring_reminders
import_batches / import_items / transaction_import_refs / import_rules / audit_logs
```

数据包按当前角色可见性过滤，不含其他账本、全局 `app_settings`、`instance_admins`、密码或物理附件内容。设置页已明确它不能替代 SQLite 物理备份或直接恢复。

## 4. Migration 与回滚

匿名 schema 19 副本在独立目录完成以下闭环：

1. `PRAGMA quick_check=ok` 后使用 SQLite `.backup` 生成一致性备份。
2. SHA-256 为 `b933514aeaeb814eab20291f04838cd6d3b9a1d7ea00b5e9edfff460b797c36c`。
3. 候选镜像自动执行 migration 020/021，health 为 `1.3.0-rc / schema 21 / staging / db ok`。
4. 用户、账本、成员、交易、结算、导入和附件引用数量守恒；交易与结算金额汇总守恒。
5. 停止候选，恢复 schema 19 备份并启动固定 v1.2 镜像，health 通过。
6. 恢复固定 v1.3 候选并重新迁移到 schema 21，health 和 quick_check 再次通过。
7. 以 schema 21 作为旧镜像回滚库时脚本在启动前退出 1，证明旧镜像不能绕过成对回滚门禁。

production 回滚仍禁止 `goose down`，只允许恢复升级前完整备份并成对回退镜像。

## 5. 浏览器验收

最终候选在 375、390、430、1440 四个视口和 Fresh Light、Dark Glass 两个主题执行 8 个组合、48 个页面检查：

```text
/login
/
/transactions
/import
/settlement
/settings
/settings/ledgers
```

结果为横向溢出 0、page error 0、严重 console error 0；390px 下主题切换往返通过。设置页“只读数据包/不可直接恢复”、导入格式、结算历史语义和账本管理均在固定镜像中可见。

截图与完整 JSON 保留在忽略目录：

```text
.runtime/v13-task50-6-staging/evidence/browser-20260717T034831/
```

仓库仅提交摘要、选定截图 SHA-256 和可重跑脚本，不提交匿名数据库、截图全集、浏览器依赖、环境文件或 JWT secret。

## 6. 非阻断项

1. 前端主 chunk 约 723 kB，Vite 继续给出大于 500 kB 提示；不影响本次正确性，后续作为性能专项拆包。
2. Figma 线上账号和本地 handoff 的最终归属仍未自动验证；Task50 使用仓库本地事实源完成，不影响运行验收。
3. NAS v1.2 production 发布线与本机 v1.3 staging 保持独立；没有维护窗口确认前不部署 v1.3 到 NAS。

## 7. 最终质量门禁

```text
backend: go test ./... -count=1              pass
backend: go vet ./...                        pass
backend: go build ./cmd/server               pass
frontend: npm run lint                       pass
frontend: npm run test                       38 files / 147 tests pass
frontend: npm run build                      pass
OpenAPI: Redocly structural errors           0
API Inventory <-> OpenAPI                    83 / 83
Docker image health                          pass
browser combinations / route checks          8 / 48
```

## 8. 后续准入

Task50 完成后不直接进入 Task51 代码。基于当前证据：

1. Task53 分类/标签与分级自动化的 P1-P6、PRD、Tech、OpenAPI draft、migration review、Fixture 和 UI/Figma handoff 已完整，可把 `Task53.1` 作为下一开发任务。
2. Task51P.1 的方法、匿名模板和假设 Fixture 已具备，但有效真实 3+ 成员证据仍为 0；继续收集证据，不冻结 P2-P6，不解除 schema 21 的最多两人约束。
3. Task52 共同支付通知继续延后，不因 Task50 关闭而自动进入工程准备。
