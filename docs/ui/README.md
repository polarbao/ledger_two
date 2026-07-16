# UI 交互设计模块目录

状态：当前 UI/UX 事实源入口
最近更新：2026-07-16

本目录按照页面和交互模块拆分 LedgerTwo 的 UI 文档。

## 总览判断

当前 UI/UX 已有总览与专项文档，不需要新增平行的 UI 总览：

| 层级 | 文件 | 作用 |
|---|---|---|
| UI 目录入口 | `README.md` | 页面文档、当前推荐入口和 Figma 目录边界 |
| 当前模块流程 | `14-v1.1-v1.2-module-flows.md` | 快捷记账、模板、周期账单、设置、结算、导入和移动端流程 |
| 长期体验专项 | `15-ledgertwo-ux-optimization-program.md` | 外部产品观察、体验原则、页面级专项和设计系统约束 |
| Figma 配套入口 | `figma/README.md` | Figma 文件定位、Token、Variables、Frame、handoff 和本地审阅规则 |
| Fresh Light 实施规格 | `figma/ledger-two-fresh-light-implementation-spec-2026-07-13.md` | Fresh Light 全应用目标、页面实现规格、组件拆分和禁区 |

后续 UI 规划应优先更新 `14`、`15`、`figma/README.md` 和 Fresh Light 实施规格。除非引入全新版本设计系统，否则不要新增并列的 UI 总览。

## 文件列表

```text
01-layout-navigation.md       整体布局、导航、响应式结构
02-dashboard.md               首页 Dashboard
03-transactions.md            流水列表、筛选、详情抽屉
04-transaction-form.md        记一笔、新增/编辑账单表单
05-settlement.md              结算中心
06-analytics.md               统计分析
07-settings.md                设置、分类、标签、账户、数据管理
08-mobile-pwa.md              移动端与 PWA 交互
09-empty-error-loading-states.md 空状态、错误态、加载态
10-design-system.md           设计系统
11-data-safety-confirmations.md 数据安全与确认交互
12-foundation-framework-ui.md Foundation before v1.1 UI
13-settings-management-redesign.md 设置管理重构
14-v1.1-v1.2-module-flows.md  v1.1-v1.2 模块流程细化
15-ledgertwo-ux-optimization-program.md LedgerTwo 长期 UI/UX 专项
16-v1.3-multi-ledger-flows.md Task50 多账本选择、生命周期、成员与无账本流程
17-v1.3-category-tag-intelligence-flows.md Task53 自动分类、标签、学习与默认元数据流程
figma/                         Figma 配套变量、组件、页面设计稿与交接清单
```

## 设计原则

- 桌面端：左侧导航 + 顶部状态 + 主内容 + 右侧抽屉。
- 移动端：顶部账本信息 + 底部 Tab + 记账快捷入口。
- 核心信息始终突出：本月总支出、我支付、对方支付、当前待结算。
- 账单卡片必须展示：分类、金额、付款人、分摊方式、标签、日期。

## Figma 与 Fresh Light 边界

`docs/ui/figma/` 是当前 UI/UX 规范的一部分，不能当作普通历史附件清理或搬迁。处理该目录时遵守：

1. `figma/README.md` 是唯一入口。
2. Token、Variables、Frame Manifest、component library 和 handoff 是设计输入，可约束实现。
3. `local-review/` 下的 HTML/SVG/PNG/PDF 是审阅证据和生成预览，不等于线上 Figma 已完成同步。
4. Figma、截图和本地预览不得覆盖已冻结的 PRD、金额、权限、分摊、结算、导入、备份和 migration 规则。
5. Fresh Light 已完成分阶段迁移并成为无偏好新会话默认体验；Dark Glass 仍是已验收历史基线和显式回滚参考，不能为了换肤删除业务能力。

UI-FL 任务以 `docs/codex_tasks/13-fresh-light-ui-interaction-plan.md` 为执行入口，并与当前业务任务双向登记。

当前状态：UI-FL-01 至 UI-FL-10 已完成，Fresh Light UI/UX 专项关闭。Task50.3B 起暂停，当前优先评审 Task53 自动分类/标签交互；本地 Figma handoff、required Frame Manifest 和组件状态矩阵已生成，视觉审阅稿与线上同步仍未验证。Task53 与暂停的账本创建 UI 存在文件所有权冲突，恢复 Task50 前必须复核。

## 当前推荐入口

短中期产品开发建议优先阅读：

1. `12-foundation-framework-ui.md`
2. `13-settings-management-redesign.md`
3. `14-v1.1-v1.2-module-flows.md`
4. `15-ledgertwo-ux-optimization-program.md`
5. `figma/README.md`
6. `16-v1.3-multi-ledger-flows.md`（Task50 UI 开发前必读）
7. `17-v1.3-category-tag-intelligence-flows.md`（当前 Task53 UI 评审入口）
8. `figma/task53-v1.3-category-tag/README.md`（Task53U 本地设计要求与同步状态）
