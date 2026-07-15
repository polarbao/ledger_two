# Task50 v1.3 多账本 Figma Handoff

状态：本地设计要求已冻结；线上 Figma 未验证同步  
冻结日期：2026-07-15  
关联任务：Task50P.5、未来 Task50.4/Task50.5

## 1. 目录角色

本目录保存 Task50 多账本的结构化 Figma 要求和代码交接，不保存真实 `.fig`、账户凭据或线上同步承诺。

| 文件 | 分类 | 作用 |
|---|---|---|
| `README.md` | 设计要求 | 范围、组件、状态、同步和审阅规则 |
| `task50-frame-manifest.json` | 设计要求 | Page/Frame/视口/状态/代码映射清单 |
| `component-state-matrix.md` | 设计要求 | 组件变体、权限、状态和文案边界 |

未来生成的 PNG/SVG/PDF/HTML 必须放到 `../local-review/task50-v1.3-<date>/`，并标注为生成审阅证据，不能与本目录要求混为“已同步 Figma”。

## 2. 事实源优先级

1. `../../../prd/31-prd-v1.3-multi-ledger.md`
2. `../../../tech/25-v1.3-multi-ledger-implementation-contract.md`
3. `../../../api/openapi-v1.3-ledger-draft.yaml`
4. `../../../prd/32-v1.3-task50-acceptance-fixtures.md`
5. `../../16-v1.3-multi-ledger-flows.md`
6. Fresh Light Token、组件库与本 Frame Manifest
7. 线上 Figma、截图和本地生成预览

设计文件不得反向改变两人上限、权限、归档只读、金额、分摊、导入、导出或整库运维规则。

## 3. Figma Page 建议

在 Fresh Light 工作稿新增 `07 v1.3 Multi-ledger` Page，不改写 v1.2 生产基线 Page。Page 内按以下 Section 排列：

1. `00 Context & Components`
2. `01 Ledger Switcher`
3. `02 Ledger Management`
4. `03 Lifecycle Confirmations`
5. `04 Members & Ownership`
6. `05 Archived & No Active`
7. `06 Error Permission Responsive`
8. `07 Code Mapping`

Frame 名称、尺寸和状态必须与 `task50-frame-manifest.json` 一致，避免同义别名。

## 4. 组件复用

必须从现有 Fresh Light 组件映射：Button、IconButton、StatusChip、SegmentedControl、StatePanel、Dialog、BottomSheet、AppShell、SettingsSection、ResponsiveDataList。

Task50 可增加业务组件：

1. `LedgerSwitcher`
2. `LedgerListItem`
3. `LedgerLifecycleSummary`
4. `LedgerMemberRow`
5. `ArchivedLedgerBanner`
6. `NoActiveLedgerShell`

这些是业务组合，不得重新定义颜色、圆角、按钮或 Dialog 基础样式。

## 5. 线上同步状态

```text
online_figma_sync=not_verified
figma_account=unknown_for_this_handoff
figma_file_url=not_bound
verified_node_ids=[]
```

本地 Fresh Light 工作稿 URL 只作为候选目标，不代表本目录内容已写入该账户。完成线上同步时必须补充：

1. 实际登录账号/团队的非敏感标识。
2. 可编辑文件 URL。
3. 每个 Frame 或 Section 的 node ID。
4. 同步时间和只读复核结果。
5. 与 manifest 的差异；不能只写“已同步”。

## 6. 开发准入

Task50 前端实现只有在以下条件同时满足后开始：

1. UI-FL-10 已完成并提交。
2. Task50P.6 已关闭。
3. manifest 的 required Frame 均有本地审阅稿或明确沿用已验收组件。
4. 代码映射、API、错误码与验收 ID 无未决冲突。
5. 线上 Figma 未同步时仍可基于本地事实源开发，但必须继续标注 `not_verified`。
