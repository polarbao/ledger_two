# Task51P.1 决策评审模板

状态：待真实证据后填写<br>
允许结论：`continue`、`narrow`、`defer`

## 1. Gate 计数

| 检查项 | 当前 | 最低要求 | 通过 |
|---|---:|---:|---|
| 独立真实目标小组 | 0 | 3 | 否 |
| 完整工作流回放 | 0 | 2 | 否 |
| 成员更替或项目结束 | 0 | 1 | 否 |
| 无通知仍愿意使用 | 0 | 2 个独立来源 | 否 |
| 同意和匿名化完整 | 0 | 所有计数证据 | 否 |

当前结论固定为 `pending`。在计数满足前不得填写 `continue` 或 `narrow`。

## 2. 证据质量审查

- [ ] 每条计数证据对应唯一小组，没有重复来源。
- [ ] 真实证据与假设 Fixture、竞品观察分开。
- [ ] 没有姓名、账号、联系方式、商户、订单、附件或可识别金额。
- [ ] 场景频率和当前替代方案来自最近事实，而不是未来意愿。
- [ ] 至少包含一个反例或放弃原因，没有只保留正向反馈。
- [ ] “无通知仍有价值”是在明确剔除通知后得到的结论。

## 3. 决策记录

```text
review_date:
reviewers:
decision: continue | narrow | defer
validated_primary_scenario:
validated_member_range:
excluded_scenarios:
notification_dependency:
history_and_privacy_risk:
current_alternative_and_switching_cost:
evidence_strength:
reasoning:
next_review_trigger:
```

## 4. 结论约束

| 结论 | 必须满足 | 后续动作 |
|---|---|---|
| continue | 多个场景有强证据，无通知仍成立 | 才允许开始 Task51P.2 |
| narrow | 仅一个明确场景成立，边界可冻结 | P2 只能围绕该场景，不扩张范围 |
| defer | 证据不足、低频或依赖通知 | 停止 P2-P6，记录下一次复审触发条件 |

即使结论为 `continue/narrow`，也只解除 P2 文档准备，不授权代码、OpenAPI、migration 或线上环境变更。
