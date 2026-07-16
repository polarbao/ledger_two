# Task51P.1 3/5 人假设原型回放 Fixture

状态：可用于本地流程和信息密度评审<br>
证据属性：假设 Fixture，不计入真实需求证据

## 1. Common constraints

1. 全部 ID、名称和金额均为虚构。
2. 金额使用整数分，显式份额之和必须等于交易金额。
3. 本文件不冻结 equal/ratio/shares 算法，也不指定未来 migration。
4. private 数据始终不因多人能力自动扩大可见性。

## 2. Three-person co-living fixture

```text
ledger_id=task51_p1_co_living_3
members=P-A owner, P-B editor, P-C editor
duration=12_months_hypothesis
transaction_id=task51_p1_tx_3
amount_cents=10001
payer=P-A
explicit_shares=P-A:5000,P-B:3000,P-C:2001
sum_shares_cents=10001
```

回放任务：

1. 创建三人长期合租账本。
2. 记录共同支出并解释三份显式金额。
3. 修改 P-C 的承担金额后定位总额不守恒错误。
4. 生成余额解释，但不把建议转账当作已支付。
5. P-C 离开后保留历史付款、分摊和姓名解释，不再授权其访问。

## 3. Five-person travel fixture

```text
ledger_id=task51_p1_travel_5
members=P-A owner,P-B editor,P-C editor,P-D viewer,P-E editor
duration=7_days_hypothesis
transaction_id=task51_p1_tx_5
amount_cents=23457
payer=P-D
explicit_shares=P-A:5000,P-B:4800,P-C:4700,P-D:4600,P-E:4357
sum_shares_cents=23457
```

回放任务：

1. 在 375px 下选择五名参与者并查看每人的金额。
2. 修正其中一人金额，错误必须定位到具体成员且总额解释可见。
3. 展示付款人与参与人不同的情况。
4. 项目结束后归档，不自动创建结算或通知。
5. 验证复制结算文案能否在没有应用内通知的情况下完成线下转账。

## 4. Review output

每次原型回放只记录：

```text
viewport:
completion_time_range:
misclick_or_correction_count:
largest_information_density_issue:
rounding_or_total_explanation_issue:
notification_free_completion: yes | no | uncertain
```

不得把原型参与者的主观偏好直接登记为真实目标小组证据，除非另行取得同意并按证据模板匿名记录。
