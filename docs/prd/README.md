# PRD 模块目录

状态：当前产品事实源入口
最近更新：2026-07-16

本目录按照产品业务模块拆分 LedgerTwo 的 PRD。后续每个模块可以独立进入设计、开发、测试和迭代。

## 总览判断

当前 PRD 已有总览文档，不需要新增平行的产品总览：

| 层级 | 文件 | 作用 |
|---|---|---|
| 产品总览 | `00-product-roadmap.md` | 当前产品定位、版本状态、路线、优先级和边界 |
| 产品复盘 | `20-product-retrospective-and-positioning.md` | 从产品经理视角重整定位、用户场景和不做事项 |
| 版本路线 | `21-roadmap-short-mid-long.md` | Foundation、v1.1、v1.2、Fresh Light、v1.3+ 路线 |
| 优先级决策 | `23-feature-priority-and-deferral-decisions.md` | 必做、应做但不急、延后和不做功能 |
| 验收口径 | `27-acceptance-case-matrix.md`、`28-transaction-caliber-supplement.md` | 关键业务验收样例和交易/账户口径 |

后续产品规划应优先更新以上总览，而不是新增重复 PRD。只有新版本或新模块已经完成范围冻结，才新增独立 PRD。

## 当前产品阶段

截至 2026-07-17，Task50.1-Task50.6 已完成，项目具备本机 `v1.3.0-rc/schema 21` 独立 staging 候选。Task53P.1-P.6 分类标签准备包完整，下一实现任务推荐为 Task53.1；Task51P.1 有效真实小组证据仍为 0。

当前产品重点：

1. 保持 v1.2 业务范围冻结。
2. 完成 NAS staging schema 19 和 production 发布门禁。
3. 保持已关闭 Fresh Light 组件、主题、响应式和可访问性基线，不改变已冻结金额、权限、导入和结算规则。
4. 按已完成的 Task53 分级自动化准备基线从 Task53.1 进入实现；Task51P 在真实证据成立前不争用共享 schema/router。
5. 第三成员、多人分摊和通知继续延后。

当前不进入开发：

1. 银行自动同步。
2. OCR 小票识别。
3. 原生 App。
4. 企业多租户。
5. 复杂预算系统。
6. 自动通知共同支付。

## 模块列表

```text
00-product-roadmap.md        产品定位、版本路线、优先级
01-ledger-member.md          账本、成员、权限
02-transaction.md            普通账单、收入、退款、转账扩展
03-shared-split-settlement.md 共同支出、分摊、结算
04-category-tag-account.md   分类、标签、账户
05-analytics-report.md       统计分析与报表
06-import-export.md          导入、导出、规则自动化
07-attachment-receipt.md     附件、小票、OCR 预留
08-budget-reminder.md        预算、提醒、周期账单
09-cross-platform.md         跨端、PWA、移动端、同步策略
11-foundation-framework-before-v1.1.md Foundation before v1.1 基础框架 PRD
12-current-progress-gap-analysis.md    当前进度与缺口分析
20-product-retrospective-and-positioning.md 产品复盘与定位重整
21-roadmap-short-mid-long.md            短期、中期、长期产品路线图
22-prd-v1.1-trust-and-daily-use.md      v1.1 可信赖与高频记账版 PRD
23-feature-priority-and-deferral-decisions.md 功能优先级与延后决策
24-short-mid-module-breakdown.md        短中期模块拆解总表
25-prd-v1.1-module-specs.md             v1.1 模块级需求规格
26-prd-v1.2-import-module-specs.md      v1.2 导入与省时间模块规格
27-acceptance-case-matrix.md            v1.1-v1.2 验收样例矩阵
28-transaction-caliber-supplement.md    交易与账户口径补充
29-prd-v1.2-module-business-service-breakdown.md v1.2 导入模块业务与服务细分
30-prd-v1.2-xlsx-import-special.md    v1.2 微信 XLSX/支付宝 CSV 导入专项 PRD
31-prd-v1.3-multi-ledger.md           v1.3 Task50 多账本正式化冻结 PRD
32-v1.3-task50-acceptance-fixtures.md Task50 匿名 Fixture、跨账本隔离与验收矩阵
33-task51-scenario-evidence-and-scope-questions.md Task51 多人分摊场景证据模板、隐私边界与范围问题
34-prd-v1.3-category-tag-intelligence.md Task53 分类、标签、默认元数据与导入智能归类 PRD
```

## 使用方式

开发某个模块前，先阅读：

1. 本模块 PRD。
2. `docs/ui/` 中对应 UI 文档。
3. `docs/tech/` 中对应技术实现文档。
4. `docs/codex_tasks/` 中对应任务卡；任务卡只用于执行切片，不替代 PRD 事实源。

发生冲突时，优先级为：

1. 当前代码、迁移、测试和已验收发布记录。
2. `00-product-roadmap.md`、`20-30` 当前 PRD。
3. `docs/tech/` 和 `docs/ui/` 当前契约。
4. `docs/codex_tasks/` 任务卡。
5. 早期 `01-09` 模块 PRD 和根目录 Demo 文档。

## 当前推荐入口

Task30 后的产品规划建议优先阅读：

1. `00-product-roadmap.md`
2. `20-product-retrospective-and-positioning.md`
3. `21-roadmap-short-mid-long.md`
4. `22-prd-v1.1-trust-and-daily-use.md`
5. `23-feature-priority-and-deferral-decisions.md`
6. `24-short-mid-module-breakdown.md`
7. `25-prd-v1.1-module-specs.md`
8. `26-prd-v1.2-import-module-specs.md`
9. `29-prd-v1.2-module-business-service-breakdown.md`
10. `27-acceptance-case-matrix.md`
11. `28-transaction-caliber-supplement.md`
12. `30-prd-v1.2-xlsx-import-special.md`（Task49X 开发前必读）
13. `31-prd-v1.3-multi-ledger.md`（Task50 准备与开发前必读）
14. `32-v1.3-task50-acceptance-fixtures.md`（Task50 测试、Migration 与验收必读）
15. `33-task51-scenario-evidence-and-scope-questions.md`（Task51P.1 非约束性发现进行中；有效真实证据仍为 0，不代表开发准入）
16. `34-prd-v1.3-category-tag-intelligence.md`（当前 Task53 产品评审入口）
