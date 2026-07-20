# UI-FL-02 AppShell 与全局导航验收记录

日期：2026-07-13

任务：`docs/codex_tasks/13-Fresh-Light界面交互协同开发计划.md` / UI-FL-02

## 1. 验收结论

UI-FL-02 已完成组件级实现与响应式验收，波次 A 的主题基础和 AppShell 可以供 UI-FL-03 至 UI-FL-09 复用。应用默认主题仍为 Dark Glass；Fresh Light 页面内容按后续任务逐页启用，不在本任务一次性切换。

本任务未修改路由地址、API、DTO、金额、权限规则、导入契约、migration 或业务状态机。

## 2. 实现范围

- `frontend/src/components/layout/appShellModel.ts`：五项一级导航、工具入口、嵌套路由激活、角色文案和记账能力判断。
- `frontend/src/components/layout/AppShell.tsx`：账本切换、角色同步、网络/草稿状态、桌面工具区、移动 FAB 和可访问导航。
- `frontend/src/components/layout/AppShell.css`：248px 桌面侧栏、1024px 切换、375/390 移动上下文、底栏和独立 FAB 动作区。
- `frontend/src/components/layout/appShellModel.test.ts`：导航顺序、嵌套路由、RBAC、语义标记和响应式契约。

## 3. 生成审阅文件

| 文件 | 状态 | 尺寸 |
|---|---|---|
| `fresh-light-desktop-1440.png` | Fresh Light、owner、在线 | 1440 x 1000 |
| `dark-glass-drafts-desktop-1440.png` | Dark Glass、2 条草稿 | 1440 x 1000 |
| `fresh-light-viewer-desktop-1440.png` | Fresh Light、viewer 只读、记账禁用 | 1440 x 1000 |
| `fresh-light-drafts-mobile-390.png` | Fresh Light、草稿入口、FAB、五项底栏 | 390 x 844 |
| `fresh-light-offline-mobile-375.png` | Fresh Light、离线提示、草稿入口 | 375 x 812 |
| `metrics.json` | 自动化、视口和证据边界 | JSON |

截图通过真实 `AppShell` 组件、MemoryRouter、QueryClient、确定性账本/健康检查响应和本地状态生成。临时预览入口已删除，因此这些文件能证明组件在指定状态下的实现结果，但不能替代真实后端登录、网络切换和账本数据 E2E，也不能证明线上 Figma 已同步。

## 4. 验证结果

```text
corepack pnpm test
11 test files passed, 41 tests passed

corepack pnpm lint
passed

corepack pnpm build
passed
```

CDP 指标：390px 视口 `innerWidth=390`、`scrollWidth=390`；375px 视口 `innerWidth=375`、`scrollWidth=375`。桌面、移动、离线、草稿和 viewer 截图已人工检查，无横向滚动、文字遮挡或导航/FAB 覆盖。

生产构建仍报告主 JavaScript chunk 约 659 kB，超过 500 kB。UI-FL-02 未新增第三方依赖；新增体积主要来自 AppShell 代码和样式，分包告警继续归属后续性能任务。

## 5. 行为保持说明

- 切换账本仍调用 `setActiveLedger` 并执行 QueryClient 全量失效。
- 账本列表返回角色变化时会同步当前角色，避免持久化角色过期后错误开放写操作。
- owner/editor 保持记账入口，viewer 显示“只读”并禁用桌面和移动记账动作。
- 在线/离线监听、草稿箱、TransactionFormDrawer、DraftListDrawer 和退出登录行为保留。
- “网络在线”只说明浏览器网络状态，不宣称服务端数据已经同步。

## 6. 下一步

UI-FL-03 迁移 Dashboard 时直接复用 AppShell、Button、StatusChip 和语义 Token，只调整页面内容区；不得重新定义一级导航、移动断点、账本切换或 FAB。真实本地部署的登录、账本切换、离线事件和草稿操作应在 UI-FL-10 统一 E2E 中再次验证。
