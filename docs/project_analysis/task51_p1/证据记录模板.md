# Task51P.1 证据记录模板

每个真实小组使用一份独立副本，仅填写匿名摘要。

## 1. Consent and source

```text
evidence_id:
source_type: interview | workflow_replay | anonymized_existing_usage
consent_method:
recorded_by:
recorded_at:
anonymization_checked_by:
```

不得填写真实姓名、账号、联系方式、商户、订单、附件或原始账单。

## 2. Scenario facts

```text
group_type: household | co_living | travel | renovation | gathering
member_count_range:
duration_range:
shared_expenses_per_period_range:
settlement_frequency:
current_workaround:
top_failure_or_cost:
privacy_expectation:
join_leave_expectation:
notification_need:
would_use_without_notification: yes | no | uncertain
```

金额只允许记录区间或相对频率，不记录可识别的真实金额明细。

## 3. Interview prompts

1. 这个小组为什么需要共同记录，而不是各自记账后偶尔算一次？
2. 当前一笔共同支出从发生到结清要经过哪些步骤？
3. 最容易出现错误或争议的是付款人、参与人、金额、历史可见性还是结算？
4. 新成员加入时，应该看到加入前哪些历史？
5. 成员离开后，其他人是否仍需解释其历史付款和分摊？
6. 如果没有邀请链接和通知，只能按现有账号加入并复制结算文案，是否仍愿意使用？
7. 这是长期重复需求，还是一次性活动需求？

## 4. Workflow replay

按顺序记录完成情况、阻力和替代动作：

```text
create_group_ledger:
add_members:
record_shared_expense:
edit_participants_or_shares:
explain_rounding_or_total:
view_balances:
record_settlement:
member_leave_or_project_archive:
largest_breakdown:
```

## 5. Reviewer classification

```text
scenario_repeatability: high | medium | low
core_product_fit: high | medium | low
requires_multi_member_model: yes | no | uncertain
notification_is_blocking: yes | no | uncertain
privacy_risk: high | medium | low
recommended_p1_signal: continue | narrow | defer | insufficient
```
