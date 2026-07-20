# 05 前端设计与实现方案：LedgerTwo v0.2

## 1. 前端目标

1. 还原清新绿色 UI 风格。
2. 桌面端支持左侧导航 + 主内容 + 右侧抽屉。
3. 移动端支持底部导航 + 卡片列表 + 底部弹窗。
4. 表单录入足够快。
5. 图表和统计清晰区分“支付”和“承担”。
6. 预留 PWA 和跨端 API 复用。

## 2. 技术选型

### 2.1 推荐栈

```text
React + TypeScript + Vite
Tailwind CSS
Radix UI / shadcn/ui
TanStack Query
Zustand
React Hook Form
Zod
Recharts 或 ECharts
```

### 2.2 UI 库对比

| 方案 | 优点 | 缺点 | 推荐 |
|---|---|---|---|
| Tailwind + shadcn/ui | 可控、现代、适合自定义设计 | 初期要搭组件体系 | 推荐 |
| Ant Design | 表格、表单成熟，桌面端强 | 移动端风格偏重，视觉不够轻量 | 管理后台可选 |
| Ant Design Mobile | 移动端体验好 | 桌面端弱 | 移动优先可选 |
| MUI | 组件全 | 默认风格偏 Material | 可选 |

推荐：Tailwind + shadcn/ui，自定义绿色主题。

### 2.3 图表库对比

| 方案 | 优点 | 缺点 | 推荐 |
|---|---|---|---|
| Recharts | React 友好、简单 | 高级图表弱 | 推荐 MVP |
| ECharts | 能力强、图表丰富 | 封装稍重 | 统计增强推荐 |
| Chart.js | 简单 | React 集成一般 | 可选 |

推荐 MVP 使用 Recharts，后续复杂图表再切 ECharts。

## 3. 前端目录结构

```text
frontend/src/
  app/
    App.tsx
    router.tsx
    providers.tsx
  api/
    client.ts
    auth.api.ts
    dashboard.api.ts
    transactions.api.ts
    settlement.api.ts
    reports.api.ts
    settings.api.ts
  components/
    layout/
      AppShell.tsx
      SideNav.tsx
      TopBar.tsx
      MobileTabBar.tsx
    common/
      MoneyText.tsx
      MonthPicker.tsx
      SyncStatusBadge.tsx
      EmptyState.tsx
      ConfirmDialog.tsx
    transaction/
      TransactionCard.tsx
      TransactionTable.tsx
      TransactionDetailDrawer.tsx
      TransactionFormDrawer.tsx
      SplitEditor.tsx
      TagSelector.tsx
    dashboard/
      SummaryCards.tsx
      SettlementSummaryCard.tsx
      RecentTransactions.tsx
      CategoryBriefCard.tsx
    settlement/
      SettlementHero.tsx
      SettlementBreakdown.tsx
      SettlementHistory.tsx
    analytics/
      TrendChart.tsx
      CategoryChart.tsx
      MemberStats.tsx
      TagRanking.tsx
  pages/
    LoginPage.tsx
    InitWizardPage.tsx
    DashboardPage.tsx
    TransactionsPage.tsx
    SettlementPage.tsx
    AnalyticsPage.tsx
    SettingsPage.tsx
  hooks/
    useMediaQuery.ts
    useMonth.ts
    useAuth.ts
  stores/
    ui.store.ts
  types/
    user.ts
    transaction.ts
    settlement.ts
    report.ts
  utils/
    money.ts
    date.ts
    clsx.ts
```

## 4. 路由设计

```text
/login
/init
/app/dashboard
/app/transactions
/app/settlement
/app/analytics
/app/settings
```

移动端可以共享同一路由，只是布局不同。

## 5. 布局实现

### 5.1 响应式规则

| 宽度 | 布局 |
|---|---|
| < 768px | 移动端底部 Tab，表单底部弹窗 |
| 768px - 1199px | 平板布局，隐藏部分侧栏 |
| >= 1200px | 桌面端左侧导航 + 主内容 + 抽屉 |

### 5.2 AppShell

AppShell 负责：

- 登录态判断
- 左侧导航
- 顶部栏
- 移动底部导航
- 主内容容器
- 全局新增按钮

## 6. 页面组件设计

### 6.1 DashboardPage

数据来源：`GET /api/v1/dashboard?month=YYYY-MM`。

组件：

- SummaryCards
- SettlementSummaryCard
- RecentTransactions
- CategoryBriefCard

交互：

- 点击待结算卡片跳转结算中心
- 点击分类跳转流水页并带筛选
- 点击最近流水打开详情抽屉
- 点击 + 打开记账抽屉

### 6.2 TransactionsPage

桌面端：表格。
移动端：卡片。

状态：

```ts
type TransactionQuery = {
  month: string;
  userId?: string;
  categoryId?: string;
  tagId?: string;
  type?: string;
  splitMode?: string;
  keyword?: string;
  page: number;
  pageSize: number;
};
```

### 6.3 TransactionFormDrawer

核心表单字段：

```ts
type TransactionFormValues = {
  type: 'expense' | 'income' | 'shared_expense' | 'settlement';
  amount: string;
  categoryId?: string;
  payerUserId: string;
  participantUserIds: string[];
  splitMethod: 'equal' | 'payer_only' | 'ratio' | 'amount';
  splits?: Array<{ userId: string; amount?: string; ratio?: number }>;
  tagNames: string[];
  occurredAt: string;
  note?: string;
  visibility: 'private' | 'partner_readable' | 'shared';
};
```

### 6.4 SplitEditor

分摊编辑器必须做到：

- 平均分摊自动计算
- 仅付款人承担自动归零对方
- 按金额时校验总额
- 按比例时校验比例总和 100%
- 显示每个人应承担金额

## 7. API Client

### 7.1 client.ts

统一封装：

- baseURL
- credentials: include
- JSON 解析
- 错误码处理
- 未登录跳转

### 7.2 TanStack Query Key

```ts
['me']
['dashboard', month, scope]
['transactions', query]
['transaction', id]
['settlementBalance', month]
['reports', 'category', month]
```

## 8. 金额处理

所有 API 使用分，前端输入使用元。

```ts
export function centsToYuan(cents: number): string {
  return (cents / 100).toFixed(2);
}

export function yuanToCents(yuan: string): number {
  return Math.round(Number(yuan || '0') * 100);
}
```

## 9. 跨端预留

前端应将业务逻辑放入可复用层：

```text
src/api       可被 PWA/React Native 复用
src/types     可被跨端复用
src/utils     金额、日期等纯函数可复用
src/domain    可选，分摊计算前端预览逻辑
```

预留：

- PWA manifest
- service worker
- 响应式布局
- API token 登录方式
- 上传附件接口

## 10. 状态与异常处理

### 10.1 保存中

按钮显示“保存中...”，禁止重复提交。

### 10.2 保存失败

展示 toast：

```text
账单保存失败，请稍后重试
```

### 10.3 删除确认

```text
删除后会影响本月结算金额，是否继续？
```

### 10.4 离线状态

顶部 SyncStatusBadge 显示：

```text
无法连接到账本服务
```

## 11. 前端验收标准

1. 桌面端 Dashboard 布局符合线框。
2. 移动端首页符合卡片风格。
3. 记一笔可在 10 秒内完成普通支出。
4. 共同支出可显示双方承担金额。
5. 流水详情可展示实付和应承担。
6. 结算中心可清楚显示谁给谁多少钱。
7. 筛选条件可同步到 URL query 或本地状态。
8. 刷新页面后当前月份和筛选状态可恢复。
