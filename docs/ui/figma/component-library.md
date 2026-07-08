# LedgerTwo 组件库规格

状态：建议采纳  
适用范围：Figma 组件库、React 组件重构、v1.2 导入工作台

## 1. 组件原则

1. 组件先服务任务效率，再服务装饰效果。
2. 移动端卡片和桌面端表格表达同一业务对象，字段口径保持一致。
3. 状态必须同时使用颜色、文字和图标，不能只依赖颜色。
4. 金额、权限、导入、备份、恢复等高风险场景必须有明确确认。
5. 不做卡片套卡片。页面区块用分组和边界，重复项才使用卡片。

## 2. 基础组件

### 2.1 Button

变体：

| 变体 | 用途 | 示例 |
|---|---|---|
| Primary | 主要动作 | 记一笔、确认导入、确认记账 |
| Secondary | 次要动作 | 清空筛选、查看详情 |
| Danger | 高风险动作 | 删除、恢复备份、确认覆盖 |
| Ghost Icon | 工具动作 | 复制、筛选、关闭、刷新 |

状态：default、hover、pressed、disabled、loading。

约束：

1. 移动端主按钮可全宽。
2. 图标按钮必须有 tooltip 或 title。
3. 危险按钮不得与主按钮视觉混淆。

### 2.2 Segmented Control

用途：

1. 流水页：全部、个人、共同、收入。
2. 导入页：全部、新行、重复、疑似、错误。
3. 统计页：分类、成员、趋势。

字段：

| 字段 | 说明 |
|---|---|
| label | 用户可读名称 |
| count | 可选计数 |
| state | active/default/disabled |

### 2.3 Status Chip

状态：

| 状态 | 文案 | 色彩 |
|---|---|---|
| new | 新增 | brand primary |
| duplicate | 重复 | muted |
| suspicious | 疑似 | warning |
| invalid | 错误 | danger |
| adjusted | 已调整 | info |
| archived | 已归档 | muted |

每个 chip 至少包含文字。导入错误状态建议附带 `AlertTriangle` 图标。

### 2.4 Amount Display

字段：

| 字段 | 说明 |
|---|---|
| amount_cents | API 与内部值，整数分 |
| display | UI 展示元 |
| tone | expense/income/settlement/neutral |

约束：

1. UI 永远展示元。
2. 表单输入可使用元，但提交前转整数分。
3. 金额列必须防止长标题挤压。

## 3. 业务组件

### 3.1 Transaction Card

用于移动端流水、Dashboard 最近流水。

必要字段：

1. 标题或分类名。
2. 金额。
3. 分类名。
4. 日期。
5. 付款人。
6. 个人/共同/可见性。
7. 标签。

状态：

1. normal。
2. archived metadata reference。
3. selected in batch mode。
4. deleted pending confirm。

### 3.2 Transaction Drawer

分区：

1. 高频：金额、类型、分类、付款人、账户。
2. 共同支出：参与人、分摊方式、分摊预览。
3. 低频：标签、时间、备注、附件、可见性。
4. 动作：保存、保存并继续、取消。

验收：

1. 保存并继续后金额、标题、备注、附件为空。
2. 分类、账户、付款人可按最近选择保留。
3. 共同支出默认 equal split。

### 3.3 Settlement Summary

必要字段：

1. 谁应转给谁。
2. 金额。
3. paid/share/raw_net/settlement/final_net 解释。
4. 复制结算文案。
5. 登记结算。

风险提示：

1. 复制文案不等于已支付。
2. 登记结算创建 settlement record，不修改历史共同支出。

### 3.4 Metadata Manager

对象：分类、标签、支付账户。

必要能力：

1. 搜索。
2. active/archived 筛选。
3. 排序。
4. 新增、编辑、归档、恢复。
5. 使用数量和历史引用提示。

### 3.5 Import Row Card

用于 v1.2 移动端导入预览。

字段：

1. 来源行号。
2. 商户/标题。
3. 金额。
4. 时间。
5. 状态 chip。
6. 推荐分类/账户/标签。
7. 错误或疑似原因。
8. 编辑、跳过、确认导入动作。

桌面端对应 `Import Row Table`，字段口径一致。

### 3.6 Commit Confirm Modal

内容：

1. 本次将导入数量。
2. 默认跳过数量。
3. 疑似未处理数量。
4. 错误数量。
5. 写入后不可通过导入批次直接撤销的提示。

阻断：

1. invalid 未修复时不能提交。
2. suspicious 未处理时不能提交。
3. 已提交批次不能重复提交。

## 4. 前端映射建议

| Figma 组件 | 当前/建议 React 位置 |
|---|---|
| AppShell | `frontend/src/components/layout/AppShell.tsx` |
| TransactionCard | `frontend/src/components/transaction/TransactionCard.tsx` |
| TransactionDrawer | `frontend/src/components/transaction/TransactionFormDrawer.tsx` |
| Empty/Error/Loading | `frontend/src/components/ui/*` |
| RestoreConfirm | `frontend/src/components/ui/RestoreBackupModal.tsx` |
| ImportWorkbench | 建议新增 `frontend/src/pages/ImportPage.tsx` 子组件 |
| ImportRowCard | 建议新增 `frontend/src/components/import/ImportRowCard.tsx` |
| ImportRowEditor | 建议新增 `frontend/src/components/import/ImportRowEditorDrawer.tsx` |

## 5. 建模优先级

1. v1.1 收口：TransactionCard、TransactionDrawer、SettlementSummary、MetadataManager。
2. v1.2 Task47：ImportEntry、ImportPreview、ImportRowCard、RowEditorDrawer。
3. v1.2 Task48：CommitConfirm、ImportResultSummary。
4. v1.2 Task49：ImportRuleManager、RuleHitExplanation。

