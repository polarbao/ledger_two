# 产品定位与版本路线

状态：当前事实源
更新时间：2026-07-02

## 1. 产品定位

LedgerTwo 是面向情侣、夫妻、合租伙伴、小家庭和小范围成员的私有化共享记账系统。

它不追求复杂会计、投资资产管理或银行同步，而是解决共同生活中的记录、分摊、统计、结算和数据安全问题。

核心问题：

1. 谁付款。
2. 谁参与消费。
3. 每个人应承担多少。
4. 当前谁应该转给谁。
5. 哪些账单可见，哪些保持私人。
6. 数据如何长期安全保存。

## 2. 当前版本状态

Task01-Task30 已完成，项目已经进入 v1.0/MVP 完成后的 Foundation before v1.1 阶段。

当前重点不是继续堆功能，而是：

- 文档事实源收口。
- 配置与部署安全。
- LedgerContext 与 RBAC。
- API 契约。
- 测试和质量门禁。
- 分类、标签、账户等长期基础能力。
- 移动端和高频记账体验。

## 3. 当前规划入口

后续产品与开发规划以以下文档为主：

```text
docs/prd/20-product-retrospective-and-positioning.md
docs/prd/21-roadmap-short-mid-long.md
docs/prd/22-prd-v1.1-trust-and-daily-use.md
docs/prd/23-feature-priority-and-deferral-decisions.md
docs/prd/24-short-mid-module-breakdown.md
docs/prd/25-prd-v1.1-module-specs.md
docs/prd/26-prd-v1.2-import-module-specs.md
docs/prd/29-prd-v1.2-module-business-service-breakdown.md
docs/prd/27-acceptance-case-matrix.md
docs/prd/28-transaction-caliber-supplement.md
docs/tech/18-short-mid-architecture-slices.md
docs/ui/14-v1.1-v1.2-module-flows.md
docs/codex_tasks/05-foundation-task-plan.md
docs/codex_tasks/08-product-roadmap-dev-plan.md
docs/codex_tasks/09-task41-49-detailed-plan.md
docs/codex_tasks/10-task33-40-detailed-plan.md
```

早期 Demo / v0.3 文档继续保留为历史约束和实现背景，但不再单独代表当前阶段。

## 4. 版本路线

| 阶段 | 目标 | 核心模块 |
|---|---|---|
| v1.0 / Task30 | MVP 完成 | 记账、共同支出、结算、统计、导入导出、备份恢复、PWA、NAS 发布文档 |
| Foundation before v1.1 | 可信基础框架 | 文档事实源、配置安全、LedgerContext、RBAC、API 契约、测试门禁、附件权限 |
| v1.1 | 可信赖与高频记账 | 快捷记账、复制一笔、模板、周期账单待确认、分类标签账户管理、移动端优化、复制结算文案 |
| v1.2 | 数据导入与省时间 | 微信/支付宝 CSV 导入、预览、去重、导入规则、批次追踪 |
| v1.3 | 轻家庭账本 | 多账本正式化、多成员角色、多人分摊、最小转账建议体验 |
| v1.5 | 长期复盘与安全健康 | 月度/年度报告、备份健康、恢复演练、长期未结算提醒 |

## 5. 当前优先级

P0：

- Task31-Task40 Foundation before v1.1。
- 配置、权限、账本上下文、附件访问控制。
- 文档事实源统一。

P1：

- 快捷记账。
- 复制一笔、模板、周期账单待确认。
- 分类/标签/账户管理。
- 移动端高频路径。
- 结算页可解释性。

P2：

- CSV 导入、预览、去重和规则。

P3：

- 多账本正式化、多成员角色和多人结算体验。

P4：

- 直接通知共同支付。
- OCR。
- 原生 App。
- 复杂预算。

## 6. 产品边界

近期不做：

- 银行自动同步。
- OCR 小票识别。
- 原生 App。
- 企业多租户。
- 投资资产管理。
- 复杂复式会计。
- 自动通知共同支付。

直接通知共同支付的当前决策：延后。短期只做“复制结算文案”，由用户自行粘贴到微信或其他沟通渠道。

## 7. 成功指标

短期：

- 普通支出 10 秒内完成。
- 共同支出 20 秒内完成。
- 核心结算回归稳定。
- 文档和任务入口不冲突。

中期：

- 用户能连续 30 天记账。
- CSV 导入可稳定去重。
- 移动端完成主要高频路径。

长期：

- 多账本/多成员小家庭场景稳定。
- 数据可备份、可恢复、可迁移。
- 报告和趋势具备复盘价值。
