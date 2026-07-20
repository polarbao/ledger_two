# UI-FL-01 双主题基础验收记录

日期：2026-07-13

任务：`docs/codex_tasks/13-Fresh-Light界面交互协同开发计划.md` / UI-FL-01

## 1. 验收结论

UI-FL-01 已完成基础实现和组件级验收，可以进入 UI-FL-02 AppShell 与全局导航迁移。当前结论只覆盖主题语义层和共享基础组件，不代表 Fresh Light 已成为全应用默认主题，也不代表后续页面迁移已经完成。

本任务未修改 API、DTO、金额、权限、导入规则、migration 或业务状态机。Dark Glass 继续作为默认解析和页面回退基线。

## 2. 实现范围

- `frontend/src/theme/theme.ts`：主题解析与 DOM 应用。
- `frontend/src/styles/tokens.css`：Fresh Light/Dark Glass 语义 Token 和旧变量兼容映射。
- `frontend/src/styles/ui-primitives.css`：基础组件、焦点、弹层和响应式样式。
- `frontend/src/components/ui/`：Button、StatusChip、SegmentedControl、StatePanel、ConfirmDialog、BottomSheet，并统一既有 Empty/Error/Loading/Skeleton 状态。

## 3. 生成审阅文件

| 文件 | 用途 | 证据边界 |
|---|---|---|
| `fresh-light-desktop.png` | Fresh Light 基础组件矩阵 | 1440 x 1400 实现截图 |
| `dark-glass-desktop.png` | Dark Glass 回退组件矩阵 | 1440 x 1400 实现截图 |
| `fresh-light-dialog.png` | 危险确认框、遮罩和动作层级 | 1440 x 900 实现截图 |
| `fresh-light-bottom-sheet-mobile.png` | 移动 Bottom Sheet 与 390px 排版 | 390 x 844 CDP 截图 |
| `metrics.json` | 自动化、视口和已知限制 | 机器可读验收摘要 |

这些 PNG 是从临时组件矩阵页面生成的代码实现审阅证据。临时页面未保留在生产代码中；截图不是 Figma 原始文件，也不能证明线上 Figma 节点、Variables 或组件已同步。

## 4. 验证结果

```text
corepack pnpm test -- src/theme/theme.test.ts src/components/ui/uiFoundation.test.ts
10 test files passed, 36 tests passed

corepack pnpm lint
passed

corepack pnpm build
passed
```

390px CDP 指标：`innerWidth=390`、`scrollWidth=390`、`innerHeight=844`。组件矩阵无横向滚动，Bottom Sheet 的关闭按钮、三段筛选和固定确认动作均在视口内。

生产构建仍报告约 655 kB 主 JavaScript chunk 超过 500 kB。该告警在 UI-FL-01 前已存在，本任务未引入新依赖，后续应作为性能/代码分包任务处理，不阻塞基础组件验收。

## 5. 下一步

启动 UI-FL-02 前，继续保持页面默认 Dark Glass；先冻结 AppShell 的导航、账本切换、离线/草稿/RBAC 状态和 375/390/1440 验收矩阵，再组合本任务组件。线上 Figma 若后续完成同步，应另补文件、节点和账号验证证据。
