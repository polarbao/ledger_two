# 产品定位与版本路线

状态：当前事实源
更新时间：2026-07-16

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

Task01-Task50 已完成，Foundation、v1.1、v1.2 导入与 v1.3 多账本均完成实现和本机验收。`1.3.0-rc/schema 21` 已在独立 WSL staging 运行，NAS 仍保持独立发布线。Task53.1 已完成 schema 22 与默认分类/标签基础，下一任务为 Task53.2 纯分类器；schema 22 尚未部署。

当前重点不是继续堆功能，而是：

- 保持 v1.2 范围冻结。
- 完成发布说明和升级说明。
- 复核 API 契约、migration 和质量门禁。
- 在部署窗口同步本机候选版本到 NAS。
- Task53 的默认分类/标签、规则优先级、显式学习、解释和既有规则兼容已经冻结为准备基线。
- 从 Task50.5 继续多账本正式化；Task51 后续只允许开展不冻结范围的 P1 场景与证据准备。

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
docs/prd/34-prd-v1.3-category-tag-intelligence.md
docs/tech/26-v1.3-category-tag-intelligence-contract.md
docs/tech/27-v1.3-category-tag-migration-review.md
docs/api/openapi-v1.3-category-tag-draft.yaml
docs/ui/17-v1.3-category-tag-intelligence-flows.md
docs/codex_tasks/18-task53-category-tag-predevelopment-plan.md
```

早期 Demo / v0.3 文档继续保留为历史约束和实现背景，但不再单独代表当前阶段。

## 4. 版本路线

| 阶段 | 目标 | 核心模块 |
|---|---|---|
| v1.0 / Task30 | MVP 完成 | 记账、共同支出、结算、统计、导入导出、备份恢复、PWA、NAS 发布文档 |
| Foundation before v1.1 | 可信基础框架 | 文档事实源、配置安全、LedgerContext、RBAC、API 契约、测试门禁、附件权限 |
| v1.1 | 可信赖与高频记账 | 快捷记账、复制一笔、模板、周期账单待确认、分类标签账户管理、移动端优化、复制结算文案 |
| v1.2 | 数据导入与省时间 | 微信/支付宝 CSV 导入、预览、去重、导入规则、批次追踪 |
| v1.3 | 轻家庭账本与元数据智能化 | 分类标签默认包、导入分级自动化、多账本正式化、成员角色 |
| v1.4+ | 多人场景评审 | 多人分摊、最小转账建议体验，是否实施由 Task51P 证据决定 |
| v1.5 | 长期复盘与安全健康 | 月度/年度报告、备份健康、恢复演练、长期未结算提醒 |

## 5. 当前优先级

P0：

- v1.2 发布候选冻结、阻断缺陷修复和部署验收。
- Task50.5 Fresh Light 账本管理、归档/恢复与成员流程。

P1：

- 前端包体与加载性能专项。
- 完整 Figma 设计系统持续补齐。

P2：

- Task50.5-Task50.6 多账本正式化已完成。

P3：

- Task53 分类标签智能化从 Task53.1 开始；Task51P 继续证据优先。

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
