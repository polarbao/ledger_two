# 技术：前端模块设计

## 1. 前端目标

前端负责提供响应式 Web 和后续 PWA 体验。所有最终业务计算以后端为准，前端只负责输入、展示、缓存和交互状态管理。

## 2. 推荐技术栈

- React + TypeScript + Vite。
- Tailwind CSS。
- TanStack Query：服务端状态。
- Zustand：轻量 UI 状态。
- React Hook Form + Zod：表单和校验。
- Recharts / ECharts：统计图表，优先 Recharts。

## 3. 目录建议

```text
frontend/src/
  app/
  pages/
  components/
    layout/
    dashboard/
    transaction/
    settlement/
    analytics/
    settings/
    common/
  api/
  hooks/
  stores/
  types/
  utils/
```

## 4. 页面模块

- InitPage：初始化。
- LoginPage：登录。
- DashboardPage：首页。
- TransactionsPage：流水。
- SettlementPage：结算。
- AnalyticsPage：统计。
- SettingsPage：设置。

## 5. API Client 规则

- 请求统一从 `src/api/client.ts` 发起。
- Cookie 登录态必须使用 `credentials: 'include'`。
- 统一处理 `{ success, data, error }` 响应。
- API 金额单位为分，前端展示转换为元。
- 时间使用 ISO8601。

## 6. 状态管理

TanStack Query 管理：

- 当前用户。
- Dashboard 数据。
- 流水列表。
- 分类、标签、账户。
- 结算状态。

Zustand 管理：

- 当前月份。
- 当前账本视图。
- 抽屉开关。
- 筛选条件。
- 移动端导航状态。

## 7. 表单原则

- 所有金额输入以元展示，提交前转为分。
- 表单校验使用 Zod。
- 共同支出表单必须明确付款人、参与人、分摊方式。
- 保存成功后刷新相关 Query。

## 8. 跨端预留

- API DTO 独立定义，避免 UI 直接依赖后端数据库结构。
- 组件分为业务组件和纯 UI 组件。
- 移动端交互优先使用 Drawer / Sheet，后续可复用到 PWA。
