# UI 交互设计模块目录

本目录按照页面和交互模块拆分 LedgerTwo 的 UI 文档。

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
```

## 设计原则

- 桌面端：左侧导航 + 顶部状态 + 主内容 + 右侧抽屉。
- 移动端：顶部账本信息 + 底部 Tab + 记账快捷入口。
- 核心信息始终突出：本月总支出、我支付、对方支付、当前待结算。
- 账单卡片必须展示：分类、金额、付款人、分摊方式、标签、日期。
