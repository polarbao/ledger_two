# 微信真实账单导入就绪评估

日期：2026-07-12

> 2026-07-14 纠偏：本文保留 Task49X 实施前的历史判断。当前代码已支持微信 CSV/XLSX；支付宝与通用模板保持 CSV-only，且 `alipay + xlsx` 会在解析前拒绝。

## 1. 结论

历史结论：本报告记录代码实现前的只读预检。后续本机 preview 已完成，仍未执行正式账单 commit。

实施更新（2026-07-12）：微信 XLSX 已在本机 `Ubuntu-24.04` WSL staging 创建 ready 预览批次，识别 276 行（275 new、1 suspicious、0 invalid），`transactions` 前后均为 40。NAS 未访问。最新证据以 `docs/project_analysis/2026-07-12-local-wsl-xlsx-csv-preview-acceptance.md` 为准。

决策更新（2026-07-12）：产品已批准在 v1.2 插入 Task49X 原生 XLSX 导入专项。当前已部署的 `1.2.0-rc/schema 18` 能力没有因此自动改变；在 Task49X、schema 19 和 staging 验收完成前，本文“不上传、不提交”的运行结论继续有效。

待处理文件：

```text
E:\__Project_Data\微信支付账单流水文件(20260101-20260701)_20260711234033.xlsx
```

当前已部署的 v1.2 导入 API 和前端只支持微信、支付宝与通用 **CSV**。在 Task49X 决策前，PRD 和技术契约曾把 Excel `xlsx` 列为非目标；后端仍直接使用 Go `encoding/csv` 读取上传字节，没有 XLSX 解包或工作表解析能力。把该文件直接上传会把 ZIP/XLSX 二进制当成 CSV，不能形成可信预览，因此没有上传到 staging 或 production，也没有写入任何导入批次或正式账单。

## 2. 文件只读检查

```text
格式: XLSX / ZIP container
文件大小: 33253 bytes
SHA-256: c39bbf15070bdf69eea1f1b1ca883cc7ee231eef8718675d167f5f1aff4bc573
工作表: Sheet1
有效工作表行数: 292
表头行: 18
数据行: 276
表头列数: 11
```

结构检查确认微信导入需要的 11 个字段全部存在，包括交易时间、交易类型、交易对方、商品、收/支、金额、支付方式、当前状态、交易单号、商户单号和备注。该文件具备后续受控转换为 CSV 的基础，但“可转换”不等于“当前可直接同步”。

## 3. 不直接转换并提交的原因

1. 用户要求不能直接导入时先不进行同步，本轮遵守该边界。
2. XLSX 前 17 行包含导出说明，转换时必须准确选择第 18 行作为表头。
3. 需要冻结日期、金额、交易单号、退款、转账和收支映射，不能只做“另存为 CSV”后直接提交。
4. 真实账单共 276 行，必须先在 staging 预览并核对 new/duplicate/suspicious/invalid/skipped 数量。
5. production 提交属于真实账务写入，必须在预览结果和抽样核对后获得明确确认。

## 4. 已确认后续路径

不再要求用户把 XLSX 手工另存为 CSV。Task49X 将在现有导入管线前增加受控 XLSX reader：

```text
格式与 OOXML 安全校验
-> 工作表和第 18 行表头定位
-> 复用现有标准化、规则和去重
-> staging 上传预览
-> 核对总行数与状态统计
-> 抽检金额/日期/交易单号
-> 处理转账、退款、疑似重复和无效行
-> production 再次预览
-> 用户确认导入数量
-> 事务提交
-> 对账与备份
```

专项 PRD、技术方案和任务计划：

- `docs/prd/30-prd-v1.2-xlsx-import-special.md`
- `docs/tech/24-v1.2-xlsx-import-implementation-plan.md`
- `docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`

若专项实施失败，降级方案是关闭 XLSX 开关并继续保留 CSV 导入，不使用临时转换脚本绕过验收。

## 5. 当前环境状态

```text
production: 1.2.0-rc / schema 18 / channel production / db ok
staging:    1.2.0-rc / schema 18 / channel staging / db ok
```

环境和数据库隔离已完成；本机 Task49X 核心实现与真实 preview 已通过，剩余发布门禁为 NAS schema 19 staging、production 一致性备份和逐批导入确认。
