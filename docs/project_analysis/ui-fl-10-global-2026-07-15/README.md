# UI-FL-10 全局体验验收记录

状态：通过，允许关闭 Fresh Light UI/UX 专项<br>
验收日期：2026-07-15<br>
验收环境：本机 WSL2 staging，`http://localhost:38088`，schema 19

## 1. 验收结论

1. 无主题偏好的新会话默认进入 Fresh Light；显式切换 Dark Glass 后本地偏好为 `dark-glass`，切回后为 `fresh-light`。
2. 登录、Dashboard、流水、分析、结算、设置、导入和周期规则在 375/390/430/1440 共 28 个路由视口组合均满足 `documentScrollWidth <= innerWidth`，未发现超出视口的可见元素。
3. AppShell 的 skip link 指向 `#main-content`，路由切换后的焦点落在主内容；登录密码显隐按钮具有可读名称且可进入键盘焦点顺序。
4. 周期规则使用语义分段控件和可读删除按钮；离线草稿改用共享 BottomSheet 与 ConfirmDialog，不再调用原生 `confirm()`。
5. 首轮视觉检查发现登录与周期规则遗留深色输入框、登录标题对比度不足；修复语义控件兼容层并重建镜像后，第二轮截图复核通过。
6. 本轮没有调用账单、结算、导入提交或设置写接口；数据库前后均为 `quick_check=ok`，计数保持 `2|2|40|2|34`。

## 2. 流程审阅

| 步骤 | 页面/流程 | 健康度 | 结论 |
|---|---|---|---|
| 1 | 登录与主题入口 | 通过 | Fresh Light 首次默认、字段对比度、标签、自动填充与密码按钮焦点正常 |
| 2 | Dashboard | 通过 | 月度摘要、待结算行动和移动记账入口层级清楚，四视口无横向溢出 |
| 3 | 流水 | 通过 | 桌面表格和移动卡片保持既有筛选、导出与操作入口 |
| 4 | 分析 | 通过 | 趋势、统计口径与四类页签可读，未改变服务端统计定义 |
| 5 | 结算 | 通过 | 当前结论、金额拆解与独立结算记录说明保持清晰 |
| 6 | 设置 | 通过 | 六分区、Owner 权限和高风险操作继续使用既有业务边界 |
| 7 | 导入 | 通过 | 来源、文件能力、Preview 不写正式账单提示均可见 |
| 8 | 周期规则 | 通过 | Fresh Light 表单控件、SegmentedControl 与删除确认完成收口 |

截图只能证明当前可见状态和响应式结果，不能单独证明完整 WCAG 合规；焦点目标、可读名称和 reduced-motion 另由 DOM 指标与自动化契约覆盖。

## 3. 证据清单

- `runtime-metrics.json`：28 个路由视口指标、主题回退、焦点、网络和运行时错误记录。
- `01-login-375.png`、`02-login-1440.png`：登录页与新会话默认主题。
- `03-dashboard-375.png`、`10-dashboard-390.png`、`11-dashboard-430.png`、`12-dashboard-1440.png`：Dashboard 四视口。
- `04-transactions-375.png`、`13-transactions-1440.png`：流水双端。
- `05-analytics-375.png`、`14-analytics-1440.png`：分析双端。
- `06-settlement-375.png`、`15-settlement-1440.png`：结算双端。
- `07-settings-375.png`、`16-settings-1440.png`：设置双端。
- `08-import-375.png`、`17-import-1440.png`：导入双端。
- `09-recurring-375.png`、`18-recurring-1440.png`：周期规则双端。

运行时未出现 JavaScript exception；登录前用于判断会话状态的三次 `/api/auth/me` 401 为预期匿名响应，登录后没有 4xx/5xx。

## 4. 构建与数据门禁

```text
镜像：ledger-two:1.2.0-rc-ui-fl-10
镜像 manifest：sha256:f8290c8584810dd3d40969ffba2e0ff9d760119b8735f6ac9e8555e883288eae
前端资源：index-BQU7hPS1.js / index-Crd5go0O.css
预部署备份：backups/predeploy/ui-fl-10-20260715-174039/ledger.db
备份 SHA256：eef355ca5dc916aa744f0c6d9b83042f7793879441c69eab8447c3ab64daed85
数据库验收后：quick_check=ok；users=2；ledgers=2；active transactions=40；settlements=2；import_batches=34
```

验证命令：

```text
frontend: npm run lint
frontend: npm run test            # 28 files / 104 tests
frontend: npm run build
backend:  go test ./...
runtime:  node backups/build/verify-ui-fl-10-runtime.mjs
```

生产构建仍报告单个 JS bundle 约 676.26 kB、gzip 196.06 kB 的既有告警。该项登记为后续 P2 性能专项，不在 UI-FL-10 内跨模块拆包。

## 5. 边界

- 未修改 API、DTO、金额、权限、导入解析、结算算法、migration 或业务状态机。
- 未部署 NAS，未把 Task50 页面、路由或 migration 混入 v1.2 镜像。
- 线上 Figma 同步仍以各 handoff 中的 `online_figma_sync` 状态为准，本地截图不冒充 Figma 账户同步证明。
