# 前端模块级实现规格 v0.3

## 1. 前端目标

前端使用 React + TypeScript + Vite 实现响应式 Web 应用。Demo 版本优先保证：

1. 手机浏览器可用。
2. 桌面 Web 信息密度足够。
3. 与后端 REST API 对接清晰。
4. 组件边界可让 AI 按页面逐步实现。
5. 后续可升级 PWA 或复用接口开发 React Native / Tauri。

## 2. 技术栈锁定

```text
React 18+
TypeScript
Vite
Tailwind CSS
TanStack Query
Zustand
React Hook Form
Zod
Recharts 或 ECharts，Demo 推荐 Recharts
```

### 2.1 UI 组件库选择

| 方案 | 优点 | 缺点 | Demo 建议 |
|---|---|---|---|
| shadcn/ui | 代码可控、现代、适合 Web | 初始配置略多 | 推荐 |
| Ant Design | 组件完整、表格强 | 移动端视觉偏重 | 可选 |
| 自定义组件 | 最轻量、可贴合 UI | 开发量大 | 可搭配 Tailwind |

Demo 推荐：Tailwind + shadcn/ui 思路，但不强依赖全量组件库。

## 3. 前端目录结构

```text
frontend/
  src/
    main.tsx
    App.tsx
    routes.tsx
    api/
      client.ts
      auth.api.ts
      init.api.ts
      dashboard.api.ts
      transactions.api.ts
      sharedExpenses.api.ts
      settlements.api.ts
      reports.api.ts
      export.api.ts
    types/
      auth.ts
      transaction.ts
      settlement.ts
      dashboard.ts
      report.ts
    stores/
      ui.store.ts
      auth.store.ts
    hooks/
      useCurrentMonth.ts
      useResponsive.ts
    components/
      layout/
        AppShell.tsx
        Sidebar.tsx
        MobileTabBar.tsx
        TopBar.tsx
      common/
        AmountText.tsx
        EmptyState.tsx
        LoadingBlock.tsx
        ConfirmDialog.tsx
        TagPill.tsx
      dashboard/
        SummaryCards.tsx
        SharedBalanceCard.tsx
        RecentTransactionList.tsx
        CategoryMiniChart.tsx
      transaction/
        TransactionTable.tsx
        TransactionCard.tsx
        TransactionDetailDrawer.tsx
        TransactionFormDrawer.tsx
        TransactionFilterSheet.tsx
      settlement/
        SettlementBalancePanel.tsx
        SettlementDetailTable.tsx
        SettlementHistoryList.tsx
        SettlementFormDialog.tsx
      analytics/
        CategoryChart.tsx
        TagChart.tsx
        MemberStatsPanel.tsx
      settings/
        CategorySettings.tsx
        TagSettings.tsx
        DataExportPanel.tsx
    pages/
      LoginPage.tsx
      InitPage.tsx
      DashboardPage.tsx
      TransactionsPage.tsx
      SettlementPage.tsx
      AnalyticsPage.tsx
      SettingsPage.tsx
    utils/
      money.ts
      date.ts
      errors.ts
```

## 4. 路由设计

```text
/init
/login
/
/transactions
/settlement
/analytics
/settings
```

### 4.1 启动逻辑

```text
App 启动
-> GET /api/init/status
-> 如果未初始化，跳转 /init
-> 如果已初始化，GET /api/auth/me
-> 未登录跳转 /login
-> 已登录进入 /
```

## 5. API Client

### 5.1 fetch 封装

`src/api/client.ts` 必须处理：

1. `credentials: 'include'`
2. JSON 序列化
3. 统一错误解析
4. 401 自动跳转 login
5. 业务错误 message 展示

示例：

```ts
export async function apiGet<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: 'include' })
  const body = await res.json()
  if (!body.success) throw new ApiError(body.error.code, body.error.message)
  return body.data as T
}
```

## 6. 页面规格

## 6.1 InitPage

字段：

1. 账本名称
2. 用户 A 显示名
3. 用户 A 用户名
4. 用户 A 密码
5. 用户 B 显示名
6. 用户 B 用户名
7. 用户 B 密码
8. 默认币种

保存后调用：

```http
POST /api/init/setup
```

成功后跳转 `/login`。

## 6.2 LoginPage

字段：

1. username
2. password

交互：

1. 登录按钮 loading。
2. 错误显示在表单上方。
3. 成功后跳转 `/`。

## 6.3 DashboardPage

桌面端布局：

```text
Sidebar + TopBar + Main Grid
```

移动端布局：

```text
TopBar + Card List + Bottom TabBar + Floating Add Button
```

组件：

1. `SummaryCards`
2. `SharedBalanceCard`
3. `RecentTransactionList`
4. `CategoryMiniChart`

API：

```http
GET /api/dashboard?month=YYYY-MM&scope=all_visible
```

## 6.4 TransactionsPage

桌面端使用表格，移动端使用卡片。

功能：

1. 月份筛选。
2. 成员筛选。
3. 分类筛选。
4. 标签筛选。
5. 搜索备注/标题。
6. 点击行打开详情抽屉。
7. 点击“记一笔”打开表单抽屉。

API：

```http
GET /api/transactions?month=YYYY-MM&page=1&page_size=20
```

## 6.5 TransactionFormDrawer

表单类型：

1. 普通支出
2. 收入
3. 共同支出
4. 结算

Demo 可分为两个表单实现：

1. `NormalTransactionForm`
2. `SharedExpenseForm`

### 普通支出字段

| 字段 | 必填 | 说明 |
|---|---:|---|
| amount | 是 | 元输入，提交前转分 |
| type | 是 | expense/income |
| category_id | 是 | 分类 |
| payer_user_id | 是 | 付款人 |
| occurred_at | 是 | 日期时间 |
| visibility | 是 | private/partner_readable |
| tag_names | 否 | 标签 |
| note | 否 | 备注 |

### 共同支出字段

| 字段 | 必填 | 说明 |
|---|---:|---|
| amount | 是 | 元输入 |
| payer_user_id | 是 | 付款人 |
| participants | 是 | 默认两人都选 |
| split_method | 是 | equal/payer_only |
| category_id | 是 | 分类 |
| occurred_at | 是 | 日期 |
| tag_names | 否 | 标签 |
| note | 否 | 备注 |

## 6.6 SettlementPage

组件：

1. `SettlementBalancePanel`
2. `SettlementDetailTable`
3. `SettlementHistoryList`
4. `SettlementFormDialog`

核心文案：

```text
lynn 应向 polar 支付 ¥186.50
```

点击“生成结算记录”：

1. 弹出确认框。
2. 默认带入 from_user、to_user、amount。
3. 提交 `POST /api/settlements`。
4. 成功后刷新 balance 和 history。

## 6.7 AnalyticsPage

Tab：

1. 分类
2. 标签
3. 成员

Demo 不强制趋势图。

## 6.8 SettingsPage

Demo 实现：

1. 分类列表。
2. 标签列表。
3. 数据导出。
4. 备份入口。

## 7. 金额处理

前端输入以元为单位，API 以分为单位。

`utils/money.ts`：

```ts
export function yuanToCents(value: string): number {
  const normalized = value.trim().replace(',', '')
  if (!/^\d+(\.\d{0,2})?$/.test(normalized)) throw new Error('金额格式错误')
  return Math.round(Number(normalized) * 100)
}

export function centsToYuan(amountCents: number): string {
  return (amountCents / 100).toFixed(2)
}

export function formatCny(amountCents: number): string {
  return `¥${centsToYuan(amountCents)}`
}
```

## 8. 响应式断点

```text
mobile: < 768px
tablet: 768px - 1023px
desktop: >= 1024px
```

规则：

1. 桌面显示 Sidebar。
2. 移动端显示 BottomTabBar。
3. 桌面流水用表格。
4. 移动端流水用卡片。
5. 桌面新增用右侧抽屉。
6. 移动端新增用底部 Sheet。

## 9. 状态管理

### 9.1 TanStack Query

用于：

1. 当前用户。
2. Dashboard。
3. 账单列表。
4. 结算余额。
5. 统计数据。

### 9.2 Zustand

只用于 UI 状态：

```ts
type UIState = {
  currentMonth: string
  addDrawerOpen: boolean
  detailDrawerTransactionId?: string
  filterOpen: boolean
}
```

## 10. 前端测试

Demo 最少实现：

1. `money.test.ts`
2. `date.test.ts`
3. `TransactionForm` 表单校验测试
4. `SettlementBalancePanel` 展示测试

## 11. 跨端预留

为了后续 React Native / Tauri：

1. API 类型放 `src/types`，不要写死在组件内。
2. 页面状态和 API 调用解耦。
3. 金额/日期工具纯函数化。
4. 不直接使用浏览器全局对象，封装 storage。
5. 认证初版 Cookie，后续可扩展 Token。
