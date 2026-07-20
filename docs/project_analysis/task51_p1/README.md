# Task51P.1 非约束性场景与证据工作区

状态：进行中，尚无有效真实小组证据<br>
启动日期：2026-07-16<br>
正式 Gate：Task50 与 Task53 WSL2 技术门禁已满足；P.1 内部准备完整，当前只等待真实证据形成 `continue/narrow/defer`

## Purpose

本目录用于证明或否定 LedgerTwo 是否需要 3+ 成员账本。它不是 Task51 PRD、版本承诺或开发准入。

优先验证：

1. 3-6 人长期家庭或合租。
2. 3-8 人旅行、装修等阶段性项目。
3. 不实现通知、邀请链接和支付追踪时，核心多人账本是否仍有价值。

单次聚会 AA、公开群组和组织报销不作为默认主线。

## Files

| 文件 | 用途 |
|---|---|
| `evidence-register.md` | 只登记匿名证据状态和评审结论 |
| `evidence-record-template.md` | 访谈、同意和工作流回放模板 |
| `prototype-replay-fixtures.md` | 3/5 人假设原型 Fixture，不算真实需求证据 |
| `interview-and-replay-runbook.md` | 招募、同意、访谈、匿名化与工作流回放执行手册 |
| `p1-decision-review-template.md` | Gate 计数、质量审查和 continue/narrow/defer 决策模板 |

进入执行阶段的完整性评审见 `../2026-07-20-task51-p1-entry-review.md`。评审确认无需继续创建平行模板；下一步是按 runbook 取得真实、匿名、经同意的证据。

## Guardrails

1. 禁止记录真实姓名、用户名、联系方式、商户、订单号、账单备注、附件或可识别金额明细。
2. 每条真实证据必须注明来源类型、授权方式和匿名化处理。
3. 假设 Fixture、竞品观察和产品经理判断不能计入真实小组数量。
4. P1 只允许 `continue`、`narrow`、`defer`；材料不足默认 `defer`。
5. P1 完成前不创建 Task51 PRD、ADR、OpenAPI 或 migration 草案。

## Current state

```text
valid_group_records=0
complete_workflow_replays=0
three_person_prototype_fixture=ready
five_person_prototype_fixture=ready
research_runbook=ready
decision_review_template=ready
decision=pending
```

当前允许执行访谈和原型回放，但不得提前创建 P.2 PRD、ADR、OpenAPI、migration 或业务代码。NAS Task53 staging 尾项与本研究工作区相互独立。
