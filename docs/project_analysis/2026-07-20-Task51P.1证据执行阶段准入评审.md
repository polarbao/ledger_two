# Task51P.1 进入证据执行阶段评审

日期：2026-07-20<br>
结论：P.1 内部准备资产完整，可以执行真实证据收集；真实证据 Gate 为 0/0，P.2-P.6 和代码继续阻断

## 1. Entry facts

1. Task50 技术基线已关闭，Task53 WSL2 发布验收为 `pass_with_suggest_only`。
2. Task53 与 Task51 的共享代码并行风险已解除；NAS staging 尾项不影响 P.1 研究执行。
3. Task51 仍是“是否值得从双人扩展到 3+ 人”的产品决策，不是已承诺版本。
4. 直接通知共同支付、支付状态和催款继续排除；P.1 必须验证没有通知时是否仍有核心价值。

## 2. Preparation audit

| Asset | Status | Purpose |
|---|---|---|
| 场景与范围问题 | ready | 长期家庭/合租、旅行/装修与偶发聚会对照 |
| 匿名证据模板 | ready | 同意、去标识、事实行为和替代方案 |
| 招募/访谈/回放手册 | ready | 3-8 人配额、35-45 分钟脚本和停止条件 |
| 3 人/5 人 Fixture | ready | 原型信息密度和工作流回放，不计真实证据 |
| 证据登记 | ready/empty | 独立来源、回放和无通知价值计数 |
| 决策模板 | ready/empty | `continue/narrow/defer` 质量评审 |

没有发现需要新增 PRD、ADR、OpenAPI、migration 或 Figma 冻结稿的准备缺口。现在继续增加内部模板只会制造文档重复，不能代替真实证据。

## 3. External evidence gate

```text
valid_group_records=0/3
complete_workflow_replays=0/2
member_change_or_project_end=0/1
notification_free_value_sources=0/2
decision=pending
```

有效记录必须来自相互独立的真实目标小组，并按工作区模板完成同意和匿名化。竞品观察、产品判断、开发者自述、重复受访者和 3/5 人 Fixture 均不计数。

## 4. Execution order

1. 招募至少 1 组长期家庭/合租、1 组旅行/装修阶段项目和 1 组对照场景。
2. 先做事实访谈，再做 3 人或 5 人原型回放，避免用界面诱导需求。
3. 至少完成 2 次端到端回放，其中 1 次覆盖成员更替或项目结束。
4. 至少从 2 个独立来源确认“不提供通知仍愿意使用”，否则优先 `narrow/defer`。
5. 达到计数后执行证据质量审查；未达到时保持 pending，不进入 P.2。

## 5. Decision boundary

P.1 得出 `continue/narrow` 只授权 Task51P.2 编写和评审多人 PRD/权限历史矩阵，不授权代码、正式 OpenAPI、migration、WSL/NAS 部署或解除当前最多两名成员约束。证据不足默认 `defer`，继续双人记账主线。
