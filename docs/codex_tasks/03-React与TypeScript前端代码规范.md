# 前端 React + TypeScript 代码风格规范

状态：供审核  
适用范围：`frontend/`

## 1. 基础原则

1. 使用 React + TypeScript + Vite。
2. 服务端状态使用 TanStack Query。
3. UI 状态使用 Zustand。
4. 表单使用 React Hook Form + Zod。
5. 组件保持小而清晰。
6. Hooks 只能在组件或自定义 Hook 顶层调用。
7. API 金额是分，UI 展示是元。
8. 前端权限只是体验控制，安全由后端兜底。

## 2. 推荐目录

```text
frontend/src/
  app/
    providers/
  api/
    client.ts
    queryKeys.ts
  features/
    ledger/
    membership/
    category/
    tag/
    account/
    transaction/
    settlement/
    safety/
    settings/
  components/
    ui/
    layout/
  stores/
  types/
  utils/
```

## 3. TypeScript 规范

1. API DTO 类型放在 `types/` 或 `api/generated/`。
2. 不使用 `any`，确需使用时必须注释原因。
3. 表单输入和 API payload 分开定义。
4. enum 可以使用 string union，避免过度复杂。
5. 金额类型命名必须表达单位，例如 `amountCents` / `amount_cents`。

## 4. React 组件规范

1. 组件名 PascalCase。
2. 自定义 Hook 以 `use` 开头。
3. 复杂页面拆为容器组件 + 展示组件。
4. 不在 JSX 中写大量内联业务逻辑。
5. 纯 UI 组件不直接调用 API。
6. 页面组件负责组合 hooks 和 mutation。

## 5. TanStack Query 规范

所有 query key 必须稳定且包含 ledgerId：

```ts
queryKeys.transactions(ledgerId, filters)
queryKeys.dashboard(ledgerId, month)
queryKeys.categories(ledgerId)
```

Mutation 成功后只 invalidate 相关 ledger 的 query。

禁止：

```ts
queryKey: ['transactions'] // 缺 ledgerId，会导致多账本缓存串数据
```

## 6. Zustand 规范

Zustand 只存：

- 当前 UI 状态。
- active ledger id 和 role。
- drawer/sheet/modal 开关。
- 当前月份。
- 离线草稿。

不存：

- 完整交易列表。
- 统计报表。
- 分类/标签/账户服务端数据。

## 7. 表单规范

1. 使用 Zod schema。
2. 金额输入为字符串，提交时转 cents。
3. 日期输入转换为 ISO8601 或明确 date。
4. 表单错误就近展示。
5. 高风险操作使用确认弹窗。
6. 移动端表单底部按钮固定，避免误触。

## 8. UI 状态规范

每个页面必须覆盖：

- loading。
- empty。
- error。
- forbidden。
- offline。

## 9. Accessibility

1. 按钮有明确文本或 aria-label。
2. 表单 label 与 input 关联。
3. 错误提示可读。
4. 弹窗打开后焦点在弹窗内。
5. 颜色不作为唯一状态表达。

## 10. 验证命令

```bash
cd frontend
pnpm lint
pnpm test
pnpm build
```
