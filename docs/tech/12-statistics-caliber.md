# 技术：统计口径与报表计算规则

## 1. 文档目标

本文件定义 LedgerTwo 的统计口径，避免“付款金额”“承担金额”“消费金额”“结算金额”混用。

统计模块必须保证：

1. 后端作为统计计算唯一可信来源。
2. 前端只展示后端返回结果。
3. settlement 不进入消费统计。
4. deleted 账单不进入统计。
5. 支付金额和承担金额必须分开展示。

## 2. 核心概念

### 2.1 实际支付金额 paid_amount

用户作为付款人实际付出去的钱。

例：A 支付晚餐 200 元，A 的 paid_amount 增加 200 元。

### 2.2 实际承担金额 share_amount

用户按照分摊规则应承担的钱。

例：A 支付晚餐 200 元，A/B 平均分摊，则 A share_amount=100 元，B share_amount=100 元。

### 2.3 净垫付金额 net_amount

用于结算计算：

```text
raw_net = paid_amount - share_amount
settlement_net = received_settlement - paid_settlement
final_net = raw_net - settlement_net
```

### 2.4 消费金额 expense_amount

用于消费统计的金额，通常指支出类交易金额，不包含 settlement、transfer。

### 2.5 收入金额 income_amount

用于收入统计的金额，只统计 income 类型。

## 3. 账单类型统计规则

| 类型 | 是否进入支出统计 | 是否进入收入统计 | 是否影响结算 | 说明 |
|---|---|---|---|---|
| expense | 是 | 否 | 否，除非扩展为共同支出 | 普通个人支出 |
| income | 否 | 是 | 否 | 普通收入 |
| shared_expense | 是 | 否 | 是 | 共同支出，进入分摊和结算 |
| settlement | 否 | 否 | 是，抵扣净额 | 结算记录，不是消费 |
| transfer | 否 | 否 | 否 | 账户间转账，后续 |
| refund | 视实现为支出抵扣 | 否 | 视关联账单而定 | 后续 |
| adjustment | 否，除非显式指定 | 否 | 视用途而定 | 后续手工调整 |

## 4. deleted 状态

所有 `status=deleted` 或 `deleted_at IS NOT NULL` 的账单默认不进入：

- 流水列表，除非开启回收站。
- 首页统计。
- 分类统计。
- 标签统计。
- 成员统计。
- 结算计算。

如果后续做“回收站”，只能用于查看和恢复，不进入任何统计。

## 5. 可见性与统计

### 5.1 当前用户视角统计

用户只能看到自己有权限看到的账单：

- 自己的 private。
- 自己或对方的 partner_readable。
- shared。

### 5.2 账本整体统计

账本整体统计可以包含 shared 和双方可见账单，但 private 是否纳入需要明确口径。

Demo 建议：

- 首页“本月总支出”只统计当前用户可见账单。
- 共同支出统计只统计 shared_expense。
- 成员结算只统计 shared_expense 和 settlement。

后续如需家庭账本整体统计，应增加“统计范围”切换：

- 我可见的账单。
- 共同账单。
- 全账本账单，只有 owner 可见。

## 6. 首页 Dashboard 口径

Dashboard 建议返回：

```text
month_total_expense
month_total_income
my_paid_amount
partner_paid_amount
my_share_amount
partner_share_amount
current_settlement_amount
recent_transactions
category_top_list
```

规则：

- month_total_expense：当前用户可见支出 + shared_expense，不含 settlement。
- month_total_income：当前用户可见 income。
- my_paid_amount：当前用户作为 payer 的 expense/shared_expense 支付金额。
- partner_paid_amount：对方作为 payer 且当前用户可见的支付金额。
- my_share_amount：共同支出中当前用户应承担金额。
- partner_share_amount：共同支出中对方应承担金额。
- current_settlement_amount：按 settlement algorithm 计算。

## 7. 分类统计口径

分类统计只统计：

- expense。
- shared_expense。

不统计：

- income。
- settlement。
- transfer。
- deleted。

共同支出分类统计默认按账单总额统计，而不是按当前用户承担金额统计。

后续可以增加视角：

- 按总消费统计。
- 按我的承担统计。
- 按付款人统计。

## 8. 标签统计口径

标签统计与分类统计一致，只统计 expense 和 shared_expense。

一笔账单有多个标签时，有两种方案：

### 方案 A：每个标签都计入全额

优点：直观，标签用于搜索和偏好分析。

缺点：多个标签合计会超过总支出。

### 方案 B：按标签数量平分金额

优点：标签合计等于总额。

缺点：不符合用户直觉。

建议 Demo 使用方案 A，并在文档中说明：标签统计是标签命中金额，不用于总额校验。

## 9. 成员统计口径

成员统计必须区分：

| 字段 | 含义 |
|---|---|
| paid_amount | 实际支付金额 |
| share_amount | 实际承担金额 |
| raw_net | 支付 - 承担 |
| settlement_paid | 已支付结算 |
| settlement_received | 已收到结算 |
| final_net | 当前仍应收/应付 |

不要使用“记账人排行”作为成员支出排行，因为创建人不等于付款人，也不等于承担人。

## 10. 趋势统计口径

趋势统计支持：

- 按日。
- 按周。
- 按月。
- 按年。

默认按 `occurred_at` 聚合，不按 created_at。

删除账单不进入趋势。

## 11. refund 未来口径

退款功能后续实现时，推荐：

- refund 关联原始 transaction。
- 退款金额作为原账单支出抵扣。
- 分类跟随原账单，除非用户手动指定。
- 若原账单是 shared_expense，则退款也需要按原分摊规则抵扣 share_amount。

v0.2-v0.4 未实现 refund 时，统计中不要预留半成品逻辑。

## 12. transfer 未来口径

transfer 是账户间转账，不属于消费也不属于收入。

例如：银行卡转入支付宝，不应导致收入增加。

transfer 只影响账户余额，Demo 不做复杂余额时可暂不实现。

## 13. API 建议

### 13.1 月度汇总

```text
GET /api/reports/monthly-summary?month=2026-06
```

返回：

```json
{
  "month": "2026-06",
  "total_expense": 120000,
  "total_income": 300000,
  "shared_expense": 80000,
  "personal_expense": 40000,
  "settlement_amount": 6000
}
```

### 13.2 成员统计

```text
GET /api/reports/member-summary?month=2026-06
```

返回：

```json
{
  "members": [
    {
      "user_id": "u1",
      "display_name": "pola",
      "paid_amount": 20000,
      "share_amount": 14000,
      "raw_net": 6000,
      "settlement_paid": 0,
      "settlement_received": 6000,
      "final_net": 0
    }
  ]
}
```

## 14. 测试用例

必须覆盖：

1. settlement 不进入支出统计。
2. deleted 不进入统计。
3. shared_expense 进入支出统计。
4. income 只进入收入统计。
5. 成员统计区分 paid 和 share。
6. 标签多选时总额口径符合文档说明。
7. Dashboard 与报表 API 的月度总额一致。

## 15. 验收标准

- 所有统计 API 使用同一套口径。
- 前端不自行计算最终结算金额。
- 分类、标签、成员统计口径在 UI 上不误导用户。
- settlement 和 transfer 不被当成消费。
- 删除账单后统计立即更新。
