# UI 交互设计模块目录（Post-Task30 Foundation）

本目录用于描述 LedgerTwo 当前 UI 和进入 v1.1 前必须补齐的基础 UI 框架。

## 文件列表

```text
01-layout-navigation.md        整体布局、导航、响应式结构
02-dashboard.md                首页 Dashboard
03-transactions.md             流水列表、筛选、详情抽屉
04-transaction-form.md         记一笔、新增/编辑账单表单
05-settlement.md               结算中心
06-analytics.md                统计分析
07-settings.md                 设置、分类、标签、账户、数据管理
08-mobile-pwa.md               移动端与 PWA 交互
12-foundation-framework-ui.md   v1.1 前基础 UI 框架
13-settings-management-redesign.md 设置页信息架构重组
```

## 设计原则

1. 移动端优先，桌面端增强。
2. 当前账本、当前月份、当前结算状态始终可见。
3. 所有高风险操作必须二次确认。
4. 所有页面必须覆盖 loading / empty / error / forbidden / offline 状态。
5. 前端权限控制只做体验辅助，最终权限由后端决定。
6. 账本切换不应造成全页面硬刷新。
7. 设置页要能承载成员、分类、标签、账户、数据安全、系统诊断等长期模块。
