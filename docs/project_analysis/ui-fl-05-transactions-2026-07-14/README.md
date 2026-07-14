# UI-FL-05 流水工作台阶段验收

日期：2026-07-14

范围：UI-FL-05 核心工作台、共享响应式列表契约、本机视觉验收和 WSL staging 静态资源更新

## 1. 结论

流水页已从 Dark Glass 页面级大文件迁移为 Fresh Light 工作台：桌面使用可扫描表格，移动端只使用交易卡片；搜索、类型分段、更多筛选、活跃筛选 Chip、详情、复制、模板、软删除、批量标签和 Owner CSV 导出保留既有业务入口。

本阶段没有新增 API、DTO、migration 或第三方依赖。金额继续以整数分从 API 传递，页面只负责元显示；软删除、批量标签和 query invalidation 保持原调用范围。

## 2. 实现范围

- `ResponsiveDataList`：冻结桌面/移动双渲染区域契约，供后续导入和分析工作台复用。
- `ActiveFilterChips`：筛选条件可见、可单项移除、可全部清除。
- `transactionsPageModel`：集中维护账单类型、金额符号、可见性、分摊和筛选文字，不显示 UUID。
- `TransactionTable`：桌面日期、类型、分类/标题、付款人、范围/分摊、标签、金额和图标操作。
- `TransactionCard`：移动端明确显示个人/共同、可见范围、分摊、付款人、日期和金额，长标题不挤压金额。
- `TransactionDetailDrawer`：桌面右抽屉、移动 Bottom Sheet，展示承担明细、标签、备注和附件。
- `TransactionsPage`：URL 筛选、分页、权限、删除、批量标签、导出和缓存失效继续由页面编排。

## 3. 视觉证据

| 文件 | 视口/状态 |
|---|---|
| `fresh-light-transactions-1440.png` | 1440 x 1000 桌面工具栏与表格 |
| `fresh-light-transactions-390.png` | 390 x 844 移动卡片与分段筛选 |
| `fresh-light-transactions-filter-390.png` | 390 x 844 筛选 Bottom Sheet |
| `fresh-light-transactions-detail-390.png` | 390 x 844 账单详情 Sheet |
| `metrics.json` | CDP 响应式宽度和可见区域指标 |

CDP 结果：

```text
mobile innerWidth/scrollWidth: 390/390
mobile card clientWidth/scrollWidth: 340/340
filter sheet clientWidth/scrollWidth: 388/388
detail drawer clientWidth/scrollWidth: 390/390
desktop innerWidth/scrollWidth: 1440/1440
desktop table clientWidth/scrollWidth: 1390/1390
mobile table visible: false
desktop mobile list visible: false
```

截图使用真实 React 组件、Fresh Light Token 和确定性脱敏数据生成；临时预览入口已删除。它验证布局和组件状态，不替代真实账号的软删除、批量标签、导出和权限 E2E。

## 4. 自动化与部署

```text
corepack pnpm test   18 files / 66 tests passed
corepack pnpm lint   passed
corepack pnpm build  passed
```

production build 生成 `assets/index-CxwEA_qT.js` 与 `assets/index-BvriRqNp.css`，主 JavaScript chunk 约 660 kB，仍保留既有大于 500 kB 告警。

构建静态资源已复制到本机 WSL staging 容器。`http://localhost:38088/api/healthz` 回读为 `1.2.0-rc / staging / schema 19 / db ok / import_xlsx_enabled=true`。本次没有重建后端镜像、执行 migration、修改 SQLite 或访问 NAS。

## 5. 剩余边界

设计目标中的“编辑账单”尚未伪装成可用入口。后端虽已有 PATCH 路由，但当前 `TransactionFormDrawer` 只冻结了新增、复制、模板和草稿编排；编辑态还需明确共同支出参与人、附件保留/删除、归档元数据替换、脏字段关闭和审计错误回显。该工作归入 UI-FL-05E，完成后再把 UI-FL-05 标记为全部完成。

真实账号回归至少覆盖：URL 筛选与刷新、历史归档名称、viewer 行动作、partner-readable 只读、软删除后统计变化、批量标签原子失败、CSV 导出权限和刷新后 PWA 静态资源。
