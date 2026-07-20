# UI：v1.1 前基础 UI 框架

状态：供审核  
目标：在 v1.1 具体业务开发前，补齐当前 Web/PWA UI 的基础结构，使其能长期承载多账本、多角色、分类标签账户管理和数据安全能力。

## 1. 当前问题

1. 账本切换依赖页面 reload。
2. SettingsPage 承载过多功能。
3. 分类、标签、账户管理入口不足。
4. Role-based UI 控制不统一。
5. 邀请/通知等未来能力没有统一入口，但本阶段不实现业务。
6. offline/draft 能力已有雏形，但需要与 active ledger 绑定。

## 2. 目标信息架构

```text
AppShell
  ├── TopBar
  │   ├── ActiveLedgerSwitcher
  │   ├── MonthPicker
  │   ├── OfflineBanner
  │   └── UserMenu
  ├── Sidebar / MobileTabBar
  ├── PageOutlet
  ├── TransactionDrawer
  └── DraftDrawer
```

## 3. ActiveLedgerSwitcher

### 3.1 展示

- 显示当前账本名称。
- 显示当前用户在账本中的角色。
- 支持切换账本。
- 如果用户无账本，展示空状态并引导创建账本。

### 3.2 行为

1. 切换账本时更新 LedgerProvider。
2. 不使用 `window.location.reload()`。
3. 自动 invalidate 当前账本相关 query。
4. 如果当前页面对新角色不可访问，跳转到 Dashboard 或展示 forbidden。

## 4. PermissionGate

前端提供统一组件：

```tsx
<PermissionGate allow={["owner", "editor"]} fallback={<ForbiddenHint />}>
  <button>新增账单</button>
</PermissionGate>
```

用途：

- 隐藏或禁用按钮。
- 展示权限说明。
- 防止 viewer 误操作。

注意：前端 PermissionGate 不是安全边界，后端必须再次校验。

## 5. 页面状态

所有主页面必须支持：

| 状态 | UI 要求 |
|---|---|
| loading | Skeleton 或 LoadingSpinner |
| empty | 解释为什么为空，并给出下一步入口 |
| error | 展示错误原因和重试按钮 |
| forbidden | 说明当前角色无权限 |
| offline | 明确说明不会保存正式数据 |
| stale | 标记当前数据可能不是最新 |

## 6. 分类/标签/账户管理 UI

Foundation 阶段至少设计以下二级页面：

```text
/settings/categories
/settings/tags
/settings/accounts
```

### 6.1 列表项展示

- 名称。
- 类型。
- 颜色/图标，可选。
- 使用次数。
- 是否归档。
- 排序把手。
- 编辑按钮。

### 6.2 操作

- 新增。
- 编辑名称/图标/颜色。
- 拖拽或上下移动排序。
- 归档。
- 恢复。

### 6.3 确认

- 归档前提示：历史账单保留，但新增账单不再默认显示。
- 已被使用的项不显示“删除”，只显示“归档”。

## 7. 设置页分组

设置页建议拆为：

1. 账号与登录。
2. 账本与成员。
3. 分类、标签、支付账户。
4. 周期账单与模板。
5. 导入导出。
6. 备份恢复。
7. 系统诊断。

移动端使用二级列表，桌面端可使用卡片网格。

## 8. 验收标准

1. 375px 宽度下无横向滚动。
2. active ledger 切换无硬刷新。
3. 所有 query 数据按 ledgerId 正确刷新。
4. viewer 看不到写入型主按钮，直接访问写入页面也显示 forbidden。
5. 分类/标签/账户管理页面具备 empty/error/loading 状态。
6. 高风险操作二次确认。
