# UI-FL-08 导入工作台验收记录

日期：2026-07-15  
实现提交：`c3399df`  
验收环境：本机 WSL2 staging，`http://localhost:38088`

## 1. 验收结论

UI-FL-08 已通过代码、自动化、运行时和数据安全门禁，可以关闭并进入 UI-FL-09：

1. 微信支持 CSV/XLSX，支付宝支持 CSV，通用模板支持 CSV；页面没有错误宣称支付宝支持 XLSX。
2. Entry、Preview、桌面表格、移动卡片、Row Editor、Rule Manager 和 Commit Confirm 均已迁移到 Fresh Light。
3. Preview 明确不写入 `transactions`；new、duplicate、suspicious、invalid、adjusted/skipped 均有文字状态和独立筛选。
4. 导入入口、预览、行调整、规则管理和提交继续为 Owner-only；设置页入口与后端权限保持一致。
5. 行级可见范围仅开放 `private` 与 `partner_readable`。当前 importer 不生成 `transaction_splits`，因此不在 UI 中伪造 `shared` 共同账单。
6. 提交确认显示导入、跳过、疑似和错误数量，并明确整批位于同一事务，任一行失败会回滚整批。
7. 375px 与 1440px 均满足 `scrollWidth = innerWidth`；移动端使用卡片和 Bottom Sheet，不显示桌面宽表；导入页隐藏全局记账 FAB，避免与批次提交栏重叠。

## 2. 证据清单

| 视口 | 入口 | 预览 | 行编辑 | 提交确认 |
|---|---|---|---|---|
| 1440px | `entry-1440.png` | `preview-1440.png` | `row-editor-1440.png` | `commit-confirm-1440.png` |
| 375px | `entry-375.png` | `preview-375.png` | `row-editor-375.png` | `commit-confirm-375.png` |

运行时断言：

- 1440px：桌面表格 4 行、移动卡片 0 行、无横向溢出。
- 375px：移动卡片 4 行、桌面表格 0 行、无横向溢出。
- 两个视口均识别 parser 摘要、状态筛选、安全提示、规则管理和可提交状态。
- 两个视口的 Row Editor 均包含类型、分类、账户、标签和可见范围，且不包含无 splits 支撑的共同账单选项。
- 两个视口的 Commit Confirm 均包含原子回滚说明、数量摘要和明确确认动作。

## 3. 数据安全

部署前备份：

- 路径：`backups/predeploy/ui-fl-08-20260715-160445/ledger.db`
- SHA-256：`dde1d2c7afc46f8e5fb5c7814712e3f2c48e739d499dbe8c5b02eed216b4b549`
- 备份校验：`PRAGMA quick_check = ok`

运行验收只上传仓库内匿名 fixture `docs/fixtures/imports/generic-basic.csv`，没有点击最终提交。验收前后：

| 指标 | 验收前 | 验收后 | 结论 |
|---|---:|---:|---|
| users | 2 | 2 | 不变 |
| ledgers | 2 | 2 | 不变 |
| transactions | 40 | 40 | Preview 未写正式账单 |
| settlements | 2 | 2 | 不变 |
| import_batches | 30 | 34 | 仅新增 4 次匿名 fixture 预览批次 |

最终数据库 `PRAGMA quick_check = ok`。没有使用真实支付宝/微信账单，没有访问或更新 NAS。

## 4. 部署回读

- 镜像标签：`ledger-two:1.2.0-rc-ui-fl-08`
- 镜像 ID：`sha256:abadd9b29b75086e5614c21b0709d41c4c99b5e436fc1c4bc39e28845c61d762`
- 前端入口：`/assets/index-DqxQYZOr.js`
- Health：`staging / schema 19 / XLSX enabled / db ok / version 1.2.0-rc`

本次只更新本机 WSL2 staging 静态资源和本地候选镜像，没有重建后端、执行 migration 或部署 NAS。

## 5. 自动化门禁

```text
frontend npm run lint       PASS
frontend npm test -- --run  PASS (25 files / 91 tests)
frontend npm run build      PASS
backend go test ./...       PASS (CGO + Qt MinGW gcc)
```

Vite 仍提示主包约 672.85 kB 大于 500 kB；这是既有性能告警，不阻断本次页面迁移，继续由 UI-FL-10/后续性能专项评估分包。

## 6. 变更边界

本任务没有修改 parser、文件 hash、批次状态机、提交事务、规则优先级、API DTO、OpenAPI、migration、Go 后端或第三方依赖。Task50 仍停留在文档准备阶段，没有借 UI-FL-08 提前进入业务编码。
