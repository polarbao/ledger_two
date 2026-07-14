# LedgerTwo Fresh Light UI/UX 实施规格（Codex 阅读版）

状态：设计目标已冻结，等待分阶段实现  
生成日期：2026-07-13  
适用阶段：v1.2 收口后的 UI 专项、v1.3+ 全局主题演进  
视觉蓝本：`docs/ui/lynntest(1).html`  
线上工作稿：`Ledger Two｜双人记账 Web UI Redesign`  
Figma：`https://www.figma.com/design/Xsw1qqEkPraqVJCIGkl41Y`

> 本文件是 Fresh Light 新版设计的本地化事实源。它供 Codex、Cursor、Copilot 和人工开发者查阅，不替代 PRD、API 契约、金额规则、权限规则或测试规范。

## 1. 执行前阅读顺序

1. `AGENTS.md`
2. `docs/00_DOCUMENT_INDEX.md`
3. `docs/prd/22-prd-v1.1-trust-and-daily-use.md`
4. `docs/prd/25-prd-v1.1-module-specs.md`
5. `docs/prd/29-prd-v1.2-module-business-service-breakdown.md`
6. `docs/tech/18-short-mid-architecture-slices.md`
7. `docs/tech/19-short-mid-implementation-readiness.md`
8. `docs/ui/14-v1.1-v1.2-module-flows.md`
9. `docs/ui/15-ledgertwo-ux-optimization-program.md`
10. `docs/ui/figma/ledger-two-design-system-brief.md`
11. `docs/ui/figma/v1.1-v1.2-ui-draft-spec.md`
12. 本文件和当前任务涉及的 React/API/测试文件

不得依据早期 Demo 文档删除已经完成的多账本、RBAC、附件、模板、周期规则、导入、审计、离线草稿或 NAS 部署能力。

## 2. 业务边界

必须保持：

- API 和数据库中的金额使用整数分；UI 显示元。
- 删除账单使用 soft delete。
- 共同支出生成 split 记录。
- 结算生成 settlement 记录，不修改历史共同支出抵消余额。
- `private` 不得向对方泄露；`partner_readable` 对方可见但不可编辑。
- 金额修改、删除、结算、导入、导出、备份和恢复等高风险动作写审计日志。
- 分摊、结算净额、权限和导入去重的权威结论来自服务端。
- TanStack Query 的 query key 必须包含 ledger id，避免跨账本缓存污染。

当前生产表单正式支持：

- 类型：`expense`、`income`、`shared_expense`。
- 分摊：`equal`、`payer_only`。
- 可见性：`private`、`partner_readable`。

未有后端契约前，不在生产 UI 中加入比例分摊、自定义金额分摊或多人份额。

## 3. 外部产品可迁移经验

只吸收通用交互，不复制品牌视觉。

### Splitwise

- 首页突出余额和“谁欠谁”。
- 共同账单先确定付款人，再确定分摊。
- 结算是独立操作和独立记录。
- 常用默认值减少重复输入。

### YNAB

- 跨端、同步、离线状态可见。
- 报表回答明确问题，而不是堆图。
- 共同使用需要清楚的成员与权限语义。

### Actual Budget

- 桌面交易列表是可搜索、筛选和整理的工作台。
- 周期账单用于提醒；LedgerTwo 仍坚持确认后入账。
- 元数据优先归档，不物理删除历史引用。
- 备份恢复必须是独立可信路径。

不引入银行同步、OCR、复杂预算、净资产或原生 App。

## 4. Fresh Light 视觉语言

从 `lynntest(1).html` 吸收：

- 页面底色 `#F5F9FC`。
- 白色表面。
- 主色 `#10B981`。
- 深青主文字 `#0B3B3B`。
- 月度摘要、分段控件、移动交易卡片和轻量状态 Chip。
- 克制阴影和清晰边界。

不直接照搬：

- 手机模型式大圆角外壳。
- Emoji 作为正式图标。
- 纯移动单列结构套用桌面。
- 营销式 Hero、强发光、卡片套卡片。

建议 CSS 语义变量：

```css
:root,
[data-theme="fresh-light"] {
  color-scheme: light;
  --lt-bg-page: #f5f9fc;
  --lt-bg-surface: #ffffff;
  --lt-bg-surface-muted: #f0faf5;
  --lt-bg-control: #f3f9f7;
  --lt-border-subtle: #e7f0ed;
  --lt-border-strong: #cfe3dc;
  --lt-text-primary: #0b3b3b;
  --lt-text-secondary: #4f6f7a;
  --lt-text-muted: #6b9080;
  --lt-brand: #10b981;
  --lt-brand-hover: #0ea371;
  --lt-brand-strong: #0e5e4f;
  --lt-info: #3a9bdc;
  --lt-warning: #f6b83e;
  --lt-danger: #ef4444;
  --lt-support-purple: #af7ac5;
  --lt-shadow-sm: 0 2px 8px rgba(0, 80, 100, 0.06);
  --lt-shadow-md: 0 8px 24px rgba(0, 80, 100, 0.08);
  --lt-focus-ring: 0 0 0 3px rgba(16, 185, 129, 0.16);
  --lt-radius-control: 8px;
  --lt-radius-card: 12px;
  --lt-radius-panel: 16px;
  --lt-radius-drawer: 20px;
  --lt-radius-pill: 999px;
}
```

迁移规则：

1. 先用 token 映射现有全局语义变量，减少 JSX 变更。
2. 保留 Dark Glass 作为可回滚模式，不立即删除。
   过渡期在登录、初始化和 AppShell 提供显式主题按钮并持久化用户选择；只有 UI-FL-10 完成全局验收后才评审默认值翻转。
3. 新组件禁止硬编码紫绿渐变或大面积半透明玻璃色。
4. 紫色降级为辅助强调。
5. 金额使用等宽数字或 `font-variant-numeric: tabular-nums`。

## 5. 信息架构

一级导航：

```text
首页        /
流水        /transactions
结算        /settlement
分析        /analytics
设置        /settings
```

二级工具：

```text
账单导入        /import
周期规则        /recurring-rules
模板管理        设置页或记账抽屉子入口
草稿箱          顶部状态或记账入口
系统诊断        设置 > 数据与安全
```

移动端底部保持：首页 / 流水 / 分析 / 结算 / 设置。`+ 记一笔` 作为悬浮或吸附主按钮，不作为第六个等权 Tab。

## 6. 页面实现规格

### 6.1 AppShell

对应：`frontend/src/components/layout/AppShell.tsx`

桌面：固定侧栏、账本选择、月份、同步/草稿状态、五个一级导航；导入和周期规则收纳为工具入口。  
移动：顶部账本与状态、底部五项导航、导航上方记账 FAB。  
不得破坏 active ledger、query 失效、离线监听、草稿箱、全局表单和 RBAC。

### 6.2 Dashboard

对应：`frontend/src/pages/DashboardPage.tsx`

首屏顺序：

1. 页面标题和“记一笔”。
2. 本月总支出、我已支付、对方已支付、待结算。
3. `谁应转给谁 ¥X` 结算行动卡。
4. 周期账单待确认。
5. 最近流水。
6. 分类摘要。

要求：

- “本月总收入”降为次级摘要。
- 欢迎 Banner 收敛为普通标题，不做营销 Hero。
- 跨月未结必须写明范围并链接结算页。
- 最近流水展示分类、标题、付款人、共同/个人、日期和金额。
- 删除 `Max 10`、`Top N` 等开发型文案。
- 分类摘要可钻取到流水筛选。
- 周期账单“跳过本期”必须确认；优先通过预填表单确认记账。

### 6.3 Transactions 工作台

对应：`frontend/src/pages/TransactionsPage.tsx`

桌面：月份、关键词、快速分段、更多筛选、活跃筛选 Chip、导出、表格和详情抽屉。  
表格列：日期、类型、分类、标题、付款人、共同/个人与分摊、标签、金额、操作。  
移动：搜索、分段、交易卡片和筛选 Bottom Sheet，不展示宽表格。

行操作：查看、编辑、复制一笔、存为模板、软删除确认。

实施边界（2026-07-14）：查看、复制一笔、存为模板和软删除确认已在 UI-FL-05 核心工作台落地；UI-FL-05E 已补齐真正的原账单编辑态。编辑时锁定类型，只允许创建者和可写角色操作，普通附件可调整，共同支出快捷编辑限 `equal/payer_only` 且参与人必须与当前成员一致；离线、自定义分摊、历史成员不一致和 settlement 明确禁用，不得退化成“复制一笔”。详细契约见 `docs/project_analysis/2026-07-14-ui-fl-05e-edit-contract.md`。

显示规则：

- 不显示 UUID。
- 未分类显示“未分类”。
- 无法解析但已设置时显示“已设分类”。
- 已归档项显示历史名称和“已归档”。
- 可见性和共同/个人必须有文字，不只依赖颜色。

### 6.4 TransactionFormDrawer

对应：`frontend/src/components/transaction/TransactionFormDrawer.tsx`

目标：普通支出 10 秒，共同支出 20 秒。

默认展开：金额、类型、分类、账户、日期。  
共同支出展开：付款人、分摊方式、参与人和承担预览。  
低频折叠：标题、标签、可见性、备注、附件、模板管理。

要求：

- 打开后聚焦金额。
- 最近分类展示 3-5 个快捷项。
- 共同支出默认两人参与、均分。
- 金额、付款人或分摊变化时更新预览；最终结果以服务端为准。
- 模板入口紧凑化，模板管理从主流程分离。
- 复制来源用紧凑提示。
- 移动端使用接近全屏 Bottom Sheet，底部动作固定。
- 有脏字段时关闭需确认。

底部动作：取消 / 保存并继续 / 保存账单。移动端主按钮全宽。

“保存并继续”必须沿用 `buildContinueTransactionFormValues`：清空金额、标题、备注和附件，保留既定分类、账户、付款人和可见性，并重新聚焦金额。

### 6.5 Settlement

对应：`frontend/src/pages/SettlementPage.tsx`

结构：

1. 全部未结 / 仅本月。
2. `谁应转给谁 ¥X`。
3. 复制结算文案、查看影响账单、登记结算。
4. 实际支付、实际承担、共同支出净额、已登记结算、最终未结。
5. 历史结算。

登记确认必须说明：生成 settlement 记录，不修改历史共同支出，并显示双方和金额。按钮写“生成结算记录”，不写“确定”。复制文案不改变状态，复制失败提供可手工选择文本。

### 6.6 Analytics

对应：`frontend/src/pages/AnalyticsPage.tsx`

分为趋势、分类、成员、标签：

- 趋势回答支出如何变化。
- 分类回答钱花到哪里并支持钻取。
- 成员展示支付、承担和净垫付，不使用“记账人排行”。
- 标签展示高频场景和金额。

settlement 不进入消费统计。图表必须有可读数据和空状态。

### 6.7 Settings

对应：`frontend/src/pages/SettingsPage.tsx` 及元数据页面。

结构：账号与登录、账本与成员、分类/标签/支付账户、模板与周期账单、导入导出、备份恢复、系统诊断。

元数据支持搜索、活跃/归档筛选、排序、使用次数、归档和恢复。危险操作单独分区并明确影响；owner-only 前端隐藏或禁用，后端继续拒绝越权。

### 6.8 Import Workbench

对应：`frontend/src/pages/ImportPage.tsx` 和 `features/imports`。

入口明确来源和格式：微信支持 CSV/XLSX，支付宝和通用模板仅 CSV；明确 preview 不写正式账单。

桌面 Preview：批次摘要、状态分段、表格、行编辑抽屉、固定提交栏。  
移动 Preview：卡片列表，展示商户、金额、时间、状态、推荐分类和完整错误原因。

状态：

- new：绿色。
- duplicate：灰色。
- suspicious：黄色。
- invalid：红色。
- adjusted：蓝色。

状态必须有文字。未处理 suspicious 或存在 invalid 时不能提交；duplicate 默认跳过；已提交批次不能重复提交。规则只做推荐，用户手工修改后不得覆盖，引用归档元数据时必须提示替换。

## 7. 建议组件拆分

通用：

```text
StatusChip.tsx
SegmentedControl.tsx
SummaryMetric.tsx
FilterBar.tsx
ActiveFilterChips.tsx
ConfirmDialog.tsx
BottomSheet.tsx
ResponsiveDataList.tsx
```

Dashboard：

```text
MonthlySummary.tsx
SettlementActionCard.tsx
RecurringReminderList.tsx
RecentTransactionList.tsx
CategorySummary.tsx
```

Transactions：

```text
TransactionToolbar.tsx
TransactionTable.tsx
TransactionCard.tsx
TransactionDetailDrawer.tsx
TransactionFilterSheet.tsx
```

表单保留 `TransactionFormDrawer.tsx` 作为 orchestration，可拆出金额、类型、高频字段、共同支出、预览、低频字段、模板选择和 Footer。拆分不得改变 mutation、copy、模板、草稿和 query invalidation 行为。

## 8. 实施顺序

```text
UI-FL-01  Fresh Light tokens + 基础组件
UI-FL-02  AppShell 导航、状态和移动 FAB
UI-FL-03  Dashboard 首屏与周期提醒
UI-FL-04  TransactionFormDrawer 高频/低频分层
UI-FL-05  Transactions 桌面工作台 + 移动卡片
UI-FL-06  Settlement 解释和确认流程
UI-FL-07  Settings 与元数据管理
UI-FL-08  Import Entry / Preview / Row Editor
UI-FL-09  Analytics 钻取与成员口径
UI-FL-10  375/390/1440 视觉、可访问性和真实业务验收
```

每个任务单独提交，并记录修改文件、测试命令、截图和真实业务操作或无法验证原因。

## 9. 状态和可访问性

关键页面覆盖默认、加载、空、错误、离线、权限不足和高风险确认。

- 状态不得只依赖颜色。
- 图标按钮必须有 `aria-label` 或可见文字。
- 正文对比达到 WCAG AA。
- 焦点使用绿色 focus ring。
- 抽屉打开后管理焦点，关闭后返回触发器。
- 触控目标至少 44×44。
- 375px 不得横向滚动。
- 长标题不得挤压金额。
- 动作按钮使用明确动词。

## 10. 验收

- 普通支出 10 秒内完成。
- 共同支出 20 秒内完成。
- Dashboard 5 秒内找到“记一笔”和待结算金额。
- 保存并继续连续录入 3 笔，无脏字段继承。
- 375/390/430px 无横向滚动。
- 共同支出保存前展示承担预览。
- 结算登记不修改历史账单。
- 归档元数据历史名称可见。
- private 账单不泄露。
- import preview 不写 transactions。
- suspicious/invalid 未处理不能 commit。

前端基础命令：

```bash
cd frontend
npm run lint
npm run test
npm run build
```

影响后端、API、金额、权限、导入、备份或 migration 时，必须执行对应测试和真实业务验收，不能只跑前端构建。

## 11. Figma Frame 名称

```text
00 Foundations / Fresh Light Tokens
00 Foundations / Typography Spacing Radius
01 Components / Buttons Chips Segments
01 Components / Transaction Cards and Table Rows
01 Components / Drawers Sheets Confirm Dialogs
02 Fresh Light Daily Use / Dashboard Desktop 1440
02 Fresh Light Daily Use / Dashboard Mobile 390
02 Fresh Light Daily Use / Transactions Desktop 1440
02 Fresh Light Daily Use / Transactions Mobile 390
02 Fresh Light Daily Use / Transaction Drawer Desktop
02 Fresh Light Daily Use / Transaction Sheet Mobile
02 Fresh Light Daily Use / Settlement Desktop 1440
02 Fresh Light Daily Use / Settlement Mobile 390
02 Fresh Light Daily Use / Analytics Desktop 1440
02 Fresh Light Daily Use / Settings Desktop 1440
03 Import Workbench / Entry Desktop 1440
03 Import Workbench / Preview Desktop 1440
03 Import Workbench / Preview Mobile 390
03 Import Workbench / Row Editor Drawer
03 Import Workbench / Commit Confirm Modal
04 States / Empty Loading Error Offline Permission
05 Code Mapping / React Component Map
```

每个 Frame 描述必须注明版本、路由、React 文件、状态、业务约束和验收宽度。

## 12. 禁区

- 不为新视觉删除已有功能。
- 不在前端重新实现权威分摊或结算计算。
- 不修改已应用 migration。
- 不引入非目标业务。
- 不展示 UUID。
- 不无确认执行结算、恢复或导入提交。
- 不一次性重写全部页面。
- 不只换颜色而忽略交互层级和业务解释。
- 未完成真实写入或验证时，不宣称 Figma、代码或 NAS 已同步。
