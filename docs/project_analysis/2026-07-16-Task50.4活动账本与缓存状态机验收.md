# Task50.4 Active Ledger 与缓存状态机验收

状态：代码与自动化验收完成；下一实现任务为 Task50.5  
日期：2026-07-16  
范围：前端活动账本偏好、启动校验、切换取消、迟到响应隔离、无活跃账本 Shell、离线草稿与本地偏好隔离

## 1. 验收结论

Task50.4 已按 PRD 31、Tech 25、UI 16 和 Fixture 32 的 `T50-WEB-001~004` 完成实现。当前活动账本不再依赖列表第一项；无可访问 active 账本时不会挂载 Dashboard、流水、分析、结算、导入、元数据或周期规则页面。

本任务没有新增 migration、后端业务表或 API 路由，没有部署 WSL staging/NAS，也没有修改 schema 21 数据。

## 2. 已实现能力

1. Zustand 只持久化 `activeLedgerId` 与按账本最近使用时间；旧版持久化的 role/status 快照会在 v2 migration 中移除。
2. 启动时使用最新 active 列表重新校验偏好；有效偏好保持，无效偏好按最近使用时间选择可访问 active 账本，没有 active 时进入明确的 no-active 状态。
3. 身份失败与账本列表失败分开处理；账本列表失败不再把已登录用户误判为退出登录。
4. 切换账本前先取消并失效旧账本 Query；全部账本内 Query Key 保留 ledger ID，页面子树按 `activeLedgerId` 重新挂载。
5. TanStack Query 的 `AbortSignal` 已传递到 Dashboard、流水、分析、结算、元数据、周期规则、导入、账本成员和表单依赖等读取请求。
6. `LedgerSwitcher` 复用同一 Desktop/Mobile model，只列 active 账本，展示角色、当前项、归档数量和管理入口。
7. `NoActiveLedgerShell` 只使用全局账本列表/创建 API，支持创建并进入新账本，以及读取已归档账本摘要。
8. 活动账本失效时显示一次原因提示；已有 active 列表的后台刷新失败保留当前页面并提供重试，不用瞬时网络错误清空上下文。
9. 离线草稿、最近分类/账户/标签/付款人等本地表单偏好按 ledger ID 隔离；旧版草稿迁移到浏览器最后持久化账本，避免 A 账本草稿显示在 B。
10. Fresh Light 与 Dark Glass 继续共用 Token、AppShell 和业务逻辑，没有建立第二套页面或组件库。

## 3. 自动化证据

新增或扩展的自动化覆盖：

1. 有效偏好保持、失效偏好按最近使用回退、无 active 不选择 archived。
2. role/status 不持久化及旧存储 migration。
3. 切换顺序为 `abort old -> commit next`。
4. A 迟到响应不覆盖 B Query Cache。
5. no-active Shell 不导入或挂载业务 API/Outlet。
6. 设置页创建第二账本复用安全切换路径。
7. 离线草稿与本地表单偏好按账本隔离。
8. 业务读取 Query 传递 AbortSignal。

已运行：

```text
frontend npm test -- --run
35 test files passed, 133 tests passed

frontend npm run lint
0 errors, 0 warnings

frontend npm run build
TypeScript and Vite production build passed
```

构建仍报告既有单 bundle 大于 500 kB 的非阻断 warning；该性能债不属于 Task50.4。

## 4. 边界与后续

1. Task50.5 现在可复用稳定的 active/no-active 状态机，开发 `/settings/ledgers`、账本详情、归档/恢复和成员/Owner 管理页面。
2. archived viewing context、完整只读历史 Banner 和 28 required Frame 的浏览器双主题验收仍归 Task50.5。
3. 全模块 A/B Fixture、schema 19 -> 21、副本升级、独立 v1.3 staging、浏览器截图和回滚演练仍归 Task50.6。
4. Task51P.1 继续只维护非约束性匿名证据；有效真实小组证据仍为 0，P2-P6 不因 Task50.4 完成而提前开放。
