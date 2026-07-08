# v1.2 导入模块 Fixture

状态：建议采纳  
适用阶段：v1.2 Task47-Task49  
用途：parser、normalizer、dedupe、UI 预览和验收样例

## 1. 文件清单

| 文件 | 来源类型 | 覆盖目标 |
|---|---|---|
| `wechat-basic.csv` | `wechat` | 微信支出、收入、退款、转账跳过、缺金额 invalid |
| `alipay-basic.csv` | `alipay` | 支付宝支出、收入、退款、疑似重复字段 |
| `generic-basic.csv` | `generic` | 通用 CSV 标准字段映射 |
| `expected/*.preview.json` | parser 预期结果 | Task47 parser/preview 断言基线 |

## 2. 使用规则

1. Fixture 不得包含真实个人账单、订单号、手机号、地址或账号。
2. 金额在 CSV 中可保留来源格式，normalizer 输出必须是整数分。
3. 转账、理财、信用卡还款等不明确流水默认 `unknown` 或 `skipped`，不得自动写入正式账单。
4. Task47 至少用每个文件覆盖 parser 和 preview。
5. Task48 复制同一 fixture 再导入，用于 duplicate 与 rollback。
6. Task47 parser 测试必须以 `expected/*.preview.json` 为断言基线；如实现调整预期，需同步更新 fixture 评审文档。

## 3. 预期重点

| 用例 | 期望 |
|---|---|
| 微信午餐 | `expense`，`amount_cents=3580`，`new` |
| 微信退款 | `income` 或退款建议，不自动作为支出 |
| 微信转账 | `unknown/skipped` |
| 缺金额行 | `invalid`，错误包含行号 |
| 支付宝同商户近时间 | 可用于构造 `suspicious` |
| 通用 CSV | 字段映射可直接进入标准 DTO |
