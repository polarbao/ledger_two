# PRD：账单系统模块

## 1. 模块目标

账单系统是 LedgerTwo 的核心数据入口，负责记录个人支出、收入、共同支出、结算流水以及后续扩展的转账、退款、周期账单等。

## 2. 当前 Demo 范围

支持：

- expense：普通支出。
- income：普通收入。
- shared_expense：共同支出。
- settlement：结算记录。

暂不支持：

- transfer：账户转账。
- refund：退款。
- recurring：周期账单。
- attachment：附件。

## 3. 账单核心字段

- title：标题。
- amount：金额，单位为分。
- type：账单类型。
- category_id：分类。
- account_id：账户。
- payer_user_id：付款人。
- owner_user_id：归属人。
- created_by_user_id：创建人。
- occurred_at：发生时间。
- visibility：可见性。
- note：备注。
- status：normal/deleted。

## 4. 账单可见性

| 可见性 | 说明 |
|---|---|
| private | 仅自己可见 |
| partner_readable | 对方可查看但不可编辑 |
| shared | 共同账本，双方可见 |

## 5. 交互要求

- 支持新增、编辑、删除、详情查看。
- 删除必须二次确认。
- 删除后影响结算金额，需要明确提示。
- 支持按月份、成员、分类、标签、金额、关键词筛选。
- 支持复制一笔，v0.3 实现。

## 6. 验收标准

- 普通支出和收入可正常创建。
- 金额为 0 或负数时创建失败。
- private 账单不会被对方看到。
- 删除后列表和统计中不再出现。
- 修改金额和删除账单写入审计日志。
