# 技术：分摊与结算算法

## 1. 目标

分摊与结算算法用于计算共同支出中每个成员实际支付、实际应承担和最终净额。

## 2. 基本概念

- paid_amount：成员实际支付金额。
- share_amount：成员应承担金额。
- settlement_paid：成员已支付结算金额。
- settlement_received：成员已收到结算金额。
- net_amount：成员净额。

## 3. 净额公式

```text
net_amount = paid_amount - share_amount - settlement_received? + settlement_paid?
```

为了避免歧义，推荐统一从“该成员当前还应收/应付”的角度计算：

```text
raw_net = paid_amount - share_amount
settlement_net = received_settlement - paid_settlement
final_net = raw_net - settlement_net
```

解释：

- final_net > 0：该成员仍应收款。
- final_net < 0：该成员仍应付款。
- final_net = 0：该成员已结清。

## 4. 双人计算示例

A 支付 20000 分，两人平摊：

```text
A paid=20000 share=10000 raw_net=10000
B paid=0 share=10000 raw_net=-10000
结果：B 应向 A 支付 10000 分
```

B 又支付 8000 分，两人平摊：

```text
A 新增 share=4000
B 新增 paid=8000 share=4000
累计：
A raw_net=10000-4000=6000
B raw_net=-10000+4000=-6000
结果：B 应向 A 支付 6000 分
```

B 向 A 结算 6000 分：

```text
A received_settlement=6000
B paid_settlement=6000
A final_net=6000-6000=0
B final_net=-6000+6000=0
双方结清
```

## 5. 分摊方式

Demo：

- equal：平均分摊。
- payer_only：仅付款人承担。

后续：

- amount：固定金额。
- ratio：比例。
- shares：份数。
- custom：自定义。

## 6. 分分钱处理

金额无法整除时，多出的分建议由付款人承担。

例：10001 分两人 equal：

```text
payer share = 5001
other share = 5000
```

## 7. 技术实现建议

- 所有计算放在后端 `BalanceCalculator`。
- 前端不得自行计算最终结算金额。
- settlement 记录只抵扣净额，不修改历史 shared_expense。
- 统计消费时排除 settlement。

## 8. 测试用例

- A 支付 20000，两人平摊，B 欠 A 10000。
- B 支付 8000，两人平摊，合并后 B 欠 A 6000。
- B 结算 6000 后双方结清。
- 删除共同支出后净额重新计算。
- 修改共同支出后净额重新计算。
