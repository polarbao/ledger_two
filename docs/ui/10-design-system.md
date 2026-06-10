# UI：设计系统与组件规范

## 1. 文档目标

定义 LedgerTwo 的视觉风格、颜色、字体、间距、圆角、组件状态和基础组件规范。

目标：

- 保证页面风格一致。
- 降低前端组件实现分歧。
- 让 AI 生成 UI 时有统一约束。
- 为后续 PWA 和移动端复用组件打基础。

## 2. 产品视觉关键词

LedgerTwo 的视觉风格应体现：

- 清晰。
- 温和。
- 可信。
- 轻量。
- 生活化。
- 适合长期记录真实账务。

不要做成复杂金融后台，也不要过度游戏化。

## 3. 色彩规范

### 3.1 主色

```text
Primary: #35C489
Primary Dark: #174D3A
Primary Light: #EAF8F2
```

使用场景：主按钮、当前导航状态、正向金额、已结清状态、当前待结算卡片辅助色。

### 3.2 背景色

```text
Page Background: #F6F8F7
Card Background: #FFFFFF
Subtle Background: #F3F5F4
```

### 3.3 文字色

```text
Text Primary: #17211D
Text Secondary: #5E6B65
Text Muted: #8A9A94
Text Disabled: #B6C1BC
```

### 3.4 状态色

```text
Success: #22C55E
Warning: #F59E0B
Danger: #EF4444
Info: #3B82F6
```

删除和危险操作使用 Danger；备份失败、服务异常使用 Warning 或 Danger；已结清、保存成功使用 Success。

## 4. 字体规范

推荐系统字体栈：

```css
font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
```

| 用途 | 字号 | 字重 |
|---|---:|---:|
| 页面标题 | 20px | 600 |
| 模块标题 | 16px | 600 |
| 正文 | 14px | 400 |
| 辅助说明 | 12px | 400 |
| 核心金额 | 28px | 700 |
| 卡片金额 | 22px | 700 |

## 5. 间距与圆角

使用 4px 基准：

```text
4 / 8 / 12 / 16 / 20 / 24 / 32
```

页面规则：

- 桌面端页面 padding：24px。
- 移动端页面 padding：16px。
- 卡片内部 padding：16px 或 20px。
- 表单字段间距：12px。

圆角：

```text
大卡片：20px
普通卡片：16px
按钮：12px
输入框：12px
标签：999px
弹窗/抽屉：20px
```

## 6. 阴影规范

默认少用强阴影。

卡片阴影：

```css
box-shadow: 0 8px 24px rgba(23, 33, 29, 0.06);
```

弹窗阴影：

```css
box-shadow: 0 16px 40px rgba(23, 33, 29, 0.16);
```

## 7. 基础组件

### 7.1 Button

类型：primary、secondary、ghost、danger、link。

状态：default、hover、active、disabled、loading。

规则：

- 页面主操作只能有一个 primary。
- 删除使用 danger，不要只用文字链接。
- loading 状态禁止重复提交。

### 7.2 Card

用于 Dashboard、统计、结算状态等模块。

Card 包含 title、value、description、action、status。

核心金额卡片需要突出金额，不要堆叠过多说明。

### 7.3 Input

状态：default、focused、error、disabled。

金额输入框：用户输入元，提交前转为分。

### 7.4 Select

用于分类、账户、成员、分摊方式。

移动端建议使用底部选择器或全屏选择页，避免小屏弹层拥挤。

### 7.5 Tag

用于账单标签、筛选条件、状态展示。

类型：normal、selected、removable、status。

### 7.6 Table

桌面端流水页使用表格。

列：日期、分类、标题、金额、付款人、分摊方式、标签、操作。

表格行点击打开详情抽屉。

### 7.7 TransactionCard

移动端流水使用卡片。

展示字段：分类图标、标题、金额、付款人、分摊方式、标签、日期。

### 7.8 Drawer / Sheet

用于新增账单、编辑账单、账单详情、筛选条件。

桌面端右侧抽屉，移动端底部 Sheet 或独立页面。

### 7.9 Modal

用于删除账单、生成结算、导出数据、恢复备份、批量导入等高风险确认。

必须包含标题、风险说明、取消按钮、确认按钮。

危险确认按钮使用 danger。

## 8. 金额展示规范

API 金额单位为分，UI 展示为元。

格式：

```text
¥48.00
¥1,280.50
```

规则：

- 收入可以使用 Success。
- 待结算金额需要突出。
- 不要在 UI 中直接展示“分”，除非调试或导出。

## 9. 状态标签

建议状态：

```text
已结清
待结算
个人账单
共同支出
仅自己可见
对方可见
已归档
已删除
```

## 10. 响应式规范

桌面端：左侧导航固定、表格优先、详情使用右侧抽屉。

移动端：底部 Tab、卡片列表、筛选使用底部 Sheet、表单可使用全屏或底部 Sheet，避免横向滚动。

## 11. 可访问性

- 按钮必须有可读文本或 aria-label。
- 表单错误需要与字段关联。
- 危险操作不能只靠颜色区分。
- 键盘可以操作主要表单。
- 对比度保持清晰。

## 12. AI 实现约束

```text
请遵守 docs/ui/10-design-system.md。
不要随意新增颜色、字号、圆角和组件风格。
新增组件时优先复用已有 Button、Card、Input、Modal、Drawer。
```

## 13. 验收标准

- 页面主色、圆角、字号统一。
- Button、Card、Input、Tag、Modal、Drawer 组件可复用。
- 移动端和桌面端视觉风格一致。
- 危险操作使用统一 danger 风格。
- 金额展示格式统一。
