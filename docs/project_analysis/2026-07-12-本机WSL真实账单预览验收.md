# 本机 WSL CSV/XLSX 真实账单预览验收

日期：2026-07-12
范围：仅本机 WSL staging，不访问 NAS，不提交正式账单

## 1. 结论

`E:\__Project_Data` 中的支付宝 CSV 和微信 XLSX 均已被当前代码正确识别，并已通过本机 `POST /api/imports/preview` 创建 ready 预览批次。

本次只执行 preview，没有调用 `/api/imports/{batch_id}/commit`。本机 `transactions` 在操作前后均为 40，两个批次的 `imported_rows` 均为 0。NAS staging/production 未访问、未部署、未迁移，也没有接收原始文件。

## 2. 文件与只读结构

| 来源 | 文件 | 大小 | SHA-256 | 结构结果 |
|---|---|---:|---|---|
| 支付宝 | `支付宝交易明细(20260101-20260701).csv` | 245023 B | `7ec2d6a64068dc3229322de90ca621e81fc65c771670e30545dcf5e4570714fb` | GB18030、逗号分隔、说明行后表头、1232 条数据 |
| 微信 | `微信支付账单流水文件(20260101-20260701)_20260711234033.xlsx` | 33253 B | `c39bbf15070bdf69eea1f1b1ca883cc7ee231eef8718675d167f5f1aff4bc573` | 标准 OOXML、Sheet1、表头第 18 行、276 条数据 |

微信文件未发现 VBA、公式或隐藏数据行。数据区外的说明区存在合并单元格，不影响第 18 行之后的流水读取。支付宝实际交付样本是 CSV，不是 XLSX；它已验证当前支付宝官方字段和 GB18030 读取。2026-07-14 确认支付宝当前仅导出 CSV，因此不再等待支付宝 XLSX 冻结样本。

## 3. 本机环境与迁移

原 `Ubuntu` WSL 注册项指向缺失的 `E:\WSL\Ubuntu\ext4.vhdx`，本轮没有注销、覆盖或删除该发行版。为避免破坏旧环境，另行安装并使用 `Ubuntu-24.04`。

本机部署状态：

```text
URL:                http://localhost:38088
container:          ledger-two
image:              ledger-two:1.2.0-rc
deployment_channel: staging
schema_version:     19
database:           ok / PRAGMA quick_check = ok
```

迁移前数据库副本：

```text
backups/predeploy/ledger-local-pre-xlsx-20260712-112402.db
SHA-256: b6d74d436d1fe90abc946f126f3d60187c7c89f53fdeb38f3776264f2afab640
```

该副本保留 schema 18 数据。当前本机数据库经应用启动自动执行 migration 019 升级为 schema 19，原有正式账单仍保留。

## 4. Preview 结果

| 来源 | batch_id | 格式/表格 | total | new | duplicate | suspicious | invalid | skipped | imported | 状态 |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|
| 支付宝 | `2d2c8b6f-d33d-4e0e-8579-5e33c78ca3bb` | CSV，表头记录 22 | 1232 | 1226 | 0 | 6 | 0 | 89 | 0 | ready |
| 微信 | `6e89cbb6-bb13-491b-931c-6d3689ee2172` | XLSX，Sheet1，表头 18 | 276 | 275 | 0 | 1 | 0 | 56 | 0 | ready |

说明：支付宝 CSV 的 `header_row_number=22` 使用 Go CSV reader 的记录编号口径，空白物理行不会形成 CSV record；原始文件中的可见表头位于说明段之后。`skipped` 是预览行上的业务选择统计，仍包含在 total 状态守恒中，不表示解析漏行。

## 5. 代码与验证边界

实现提交：

1. `3057f23`：CSV/XLSX 表格读取、格式安全、来源适配、migration 019 与后端测试。
2. `e299c87`：前端 CSV/XLSX 文件选择、解析摘要、错误文案与测试。

已验证：

1. 后端全量 `go test ./... -count=1`。
2. 前端 7 个测试文件、24 个测试用例。
3. 前端 production build；仅保留既有主 chunk 大小警告。
4. 本机 health、schema 19、数据库 quick_check、两个 ready 批次和 transactions=40。

未在本报告中关闭的门禁：

1. 未执行任何真实批次 commit，也未做 commit 后对账。
2. 支付宝当前仅支持 CSV，前后端必须拒绝支付宝 XLSX。
3. 未完成账户主 Figma 同步及 375px/390px 浏览器视觉截图验收。
4. 未执行 NAS schema 19 部署；NAS 继续保持既有 schema 18 状态。

## 6. 后续顺序

1. 在浏览器打开 `http://localhost:38088`，使用现有本机 QA 账号复核两个预览批次，不在未确认前提交。
2. 金额总计、收支方向、退款、转账、首尾记录和订单号分层抽样已完成，见 `v1.2-task49x-ui-acceptance-2026-07-12/`。
3. Task49X.4 已按支付宝真实 CSV 完成冻结；未来官方格式变化时另开需求。
4. 375px/390px 本机视觉验收已完成；Figma 写入因账号只有 View 权限待处理，再决定 NAS staging schema 19 发布窗口。
5. production commit 必须重新上传、重新预览，并由用户明确确认各状态数量。

## 7. 2026-07-13 本机复验

在 Git 历史同步完成后，使用 `E:\__Project_Data` 中同一组本地文件重新执行了本机 WSL staging preview。文件 SHA-256 与本报告第 2 节一致，说明本次复验输入没有发生变化。

复验环境：

```text
URL:                 http://localhost:38088
version:             1.2.0-rc
schema_version:      19
deployment_channel: staging
import_xlsx_enabled: true
database:            ok
```

复验结果：

| 来源 | 新 batch_id | 格式/表格 | total | new | suspicious | invalid | skipped | imported | 状态 |
|---|---|---|---:|---:|---:|---:|---:|---:|---|
| 支付宝 | `99fc1199-ed4a-4424-9954-74235c56ced8` | CSV，表头记录 22 | 1232 | 1226 | 6 | 0 | 89 | 0 | ready |
| 微信 | `60798923-a133-40ce-9895-ec0c8666953f` | XLSX，Sheet1，表头 18 | 276 | 275 | 1 | 0 | 56 | 0 | ready |

preview 前后正式账单均为 40 条，没有调用任何 batch commit API。复验再次确认微信真实 XLSX 与支付宝真实 CSV 的当前解析路径稳定；支付宝 XLSX 不在当前范围内。本记录不代表 NAS staging 或 production 已支持 schema 19/微信 XLSX。
