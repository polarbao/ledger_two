# v1.2 导入模块 Fixture

状态：建议采纳  
适用阶段：v1.2 Task47-Task49、Task49X
用途：parser、normalizer、dedupe、UI 预览和验收样例

## 1. 文件清单

| 文件 | 来源类型 | 覆盖目标 |
|---|---|---|
| `wechat-basic.csv` | `wechat` | 微信支出、收入、退款、转账跳过、缺金额 invalid |
| `alipay-basic.csv` | `alipay` | 支付宝支出、收入、退款、疑似重复字段 |
| `generic-basic.csv` | `generic` | 通用 CSV 标准字段映射 |
| `expected/*.preview.json` | parser 预期结果 | Task47 parser/preview 断言基线 |

Task49X 开发时新增：

| 文件 | 来源类型 | 覆盖目标 |
|---|---|---|
| `wechat-basic.xlsx` | `wechat` | 与 wechat-basic.csv 标准化结果和 hash 等价 |
| `xlsx/wechat-header-row-18.xlsx` | `wechat` | 说明行、物理表头 18 和数据行号 |
| `xlsx/multiple-matching-sheets.xlsx` | `wechat` | 多候选工作表拒绝 |
| `xlsx/formula-required-cell.xlsx` | `wechat` | 必需字段公式拒绝 |

## 2. 使用规则

1. Fixture 不得包含真实个人账单、订单号、手机号、地址或账号。
2. 金额在 CSV 中可保留来源格式，normalizer 输出必须是整数分。
3. 转账、理财、信用卡还款等不明确流水默认 `unknown` 或 `skipped`，不得自动写入正式账单。
4. Task47 至少用每个文件覆盖 parser 和 preview。
5. Task48 复制同一 fixture 再导入，用于 duplicate 与 rollback。
6. Task47 parser 测试必须以 `expected/*.preview.json` 为断言基线；如实现调整预期，需同步更新 fixture 评审文档。
7. XLSX fixture 必须由匿名数据生成，不得从真实账单直接脱敏后提交。
8. 二进制 fixture 新增或变化时记录 SHA-256，并由测试核对工作表、表头行和数据行数。
9. 微信 XLSX 与对应 CSV 的 occurred_at、amount_cents、direction、merchant、title、external_order_id 和 import_hash 必须一致。
10. 超大文件优先在测试中动态生成，避免把大二进制文件提交到仓库。
11. 支付宝当前仅支持 CSV；`alipay + xlsx` 必须由前后端拒绝，不创建 preview batch。

## 3. 预期重点

| 用例 | 期望 |
|---|---|
| 微信午餐 | `expense`，`amount_cents=3580`，`new` |
| 微信退款 | `income` 或退款建议，不自动作为支出 |
| 微信转账 | `unknown/skipped` |
| 缺金额行 | `invalid`，错误包含行号 |
| 支付宝同商户近时间 | 可用于构造 `suspicious` |
| 通用 CSV | 字段映射可直接进入标准 DTO |
