# UI-FL-03 Dashboard 验收记录

日期：2026-07-13

任务：`docs/codex_tasks/13-Fresh-Light界面交互协同开发计划.md` / UI-FL-03

## 1. 验收结论

UI-FL-03 已完成 Dashboard 的 Fresh Light 信息层级、组件拆分和响应式迁移，可以作为 UI-FL-04 记账抽屉的稳定入口。Dashboard 继续复用 Task41/43 已有 API、聚合口径、周期提醒 mutation 和全局记账抽屉，不改变业务契约。

## 2. 实现范围

- `dashboardModel.ts`：月度摘要、结算行动、金额、周期频率、流水类型和付款人展示模型。
- `MonthlySummary.tsx`、`SettlementActionCard.tsx`：五项摘要和“谁转给谁”的结算行动；不展示 UUID。
- `RecurringReminderList.tsx`：待确认、确认记账和跳过本期，保留既有 mutation 与失效策略。
- `CategorySummary.tsx`、`MemberContributionSummary.tsx`、`RecentTransactionList.tsx`：分类、成员支付/承担和最近流水。
- `DashboardPage.tsx`、`DashboardPage.css`：稳定首屏顺序、桌面双列内容区、375px 两列摘要及最后一项跨列。

## 3. 视觉证据

| 文件 | 状态 | 尺寸 |
|---|---|---|
| `fresh-light-dashboard-1440.png` | owner、在线、待结算、1 笔周期提醒 | 1440 x 1000 |
| `fresh-light-dashboard-375.png` | owner、在线、移动首屏 | 375 x 812 |
| `metrics.json` | CDP 视口、横向滚动和首屏信息检查 | JSON |

截图通过真实 AppShell 与 Dashboard 组件、Fresh Light Token、确定性 Dashboard/账本/周期提醒响应和本地状态生成。临时预览入口已删除，因此证据可证明组件和指定视口的呈现结果，但不替代真实后端登录、提醒写入、网络变化和跨页 E2E，也不代表线上 Figma 主文件已同步。

## 4. 验证结果

```text
corepack pnpm test
14 test files passed, 51 tests passed

corepack pnpm lint
passed

corepack pnpm build
passed
```

Chrome 150 CDP 指标：1440px 视口 `innerWidth=scrollWidth=1440`；375px 视口 `innerWidth=scrollWidth=375`。两张截图已人工检查，标题、金额、按钮和状态无横向裁切；移动摘要的第五项跨两列，不保留空白网格。

生产构建仍报告主 JavaScript chunk 约 661 kB，超过 500 kB。UI-FL-03 未新增第三方依赖，该告警继续归属后续性能专项。

## 5. 业务保持说明

- 月度金额和 paid/share/balance 只读取既有 Dashboard 响应，金额仍使用整数分。
- 结算入口进入 `/settlement`；Dashboard 不计算或写入 settlement，不改历史共同账单。
- 周期确认和跳过继续调用既有 API；确认后失效 Dashboard、提醒和流水，跳过只失效提醒。
- 首页“记一笔”和移动 FAB 继续打开全局 `TransactionFormDrawer`，没有修改 UI-FL-04 的表单契约。
- owner/editor/viewer 仍由既有权限组件和 AppShell 角色控制。

## 6. 下一步

UI-FL-04 先冻结记账抽屉的字段分组、Footer、移动 Bottom Sheet 和提交状态契约；UI-FL-05 在该入口契约稳定后迁移流水工作台。Task49X 的 NAS staging schema 19 和 production 发布门禁继续独立收口，不混入 UI-FL 页面提交；支付宝保持 CSV-only。
