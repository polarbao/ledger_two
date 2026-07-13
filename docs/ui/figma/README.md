# LedgerTwo Figma 配套设计包

状态：持续维护  
创建日期：2026-07-08  
最近更新：2026-07-13  
适用阶段：v1.1 历史收口、v1.2 导入模块、v1.2 收口后的 Fresh Light 专项、v1.3+ 长期 UI/UX 演进

## 1. 目的

本目录用于把 LedgerTwo 的 PRD、技术约束、UI 文档、当前 React 前端和 `docs/ui/lynntest(1).html` 原型沉淀为可供 Figma 建模、设计评审、Codex 查阅和前端实现使用的结构化事实源。

本目录不是代码替代品，也不是一次性换皮方案。它负责：

1. 保留 v1.1/v1.2 已验收界面的阶段事实。
2. 给 v1.2 导入工作台提供逐屏规格和组件约束。
3. 将 Fresh Light 全应用目标设计本地化，避免只存在于聊天或线上账号。
4. 给 Figma 提供 Variables、Token、Page、Frame 和组件清单。
5. 给前端实现提供路由、组件映射、状态、断点、优先级和验收口径。

## 2. 设计事实源层级

从高到低：

1. PRD、技术契约、API、金额和权限规则。
2. 当前版本专项文档和验收记录。
3. 本目录的 Fresh Light 实施规格、逐屏规格、Token、Variables、组件库和 Frame Manifest。
4. 已验证的 Figma Frame、截图和本地导出物。
5. 探索稿、历史 Figma 文件和原型。

Figma 或截图不得覆盖已冻结的金额、分摊、结算、权限、导入和备份规则。

## 3. 文件清单

| 文件 | 用途 |
|---|---|
| `ledger-two-design-system-brief.md` | 设计方向、现有差异、阶段迁移策略 |
| `ledger-two-fresh-light-implementation-spec-2026-07-13.md` | Fresh Light 全应用目标设计、Codex 实施规格和禁区 |
| `ledger-two.design-tokens.json` | 面向前端和设计系统的 Token |
| `ledger-two.figma-variables.json` | 面向 Figma Variables 的变量集合 |
| `ledger-two-frame-manifest.json` | Figma 页面、Frame、状态和代码映射清单 |
| `component-library.md` | 组件库规格、状态和 React 映射 |
| `v1.1-v1.2-ui-draft-spec.md` | v1.1/v1.2 逐屏规格与 Fresh Light 目标稿补充 |
| `v1.2-task49-import-rule-manager-handoff.md` | Task49 规则管理、命中解释、多标签和移动端规格 |
| `v1.2-task49x-xlsx-import-handoff.md` | Task49X XLSX 选择、解析摘要、错误状态和移动端规格 |
| `handoff-checklist.md` | UI 设计与开发交接检查清单 |
| `local-review/` | Markdown 审阅、PNG/PDF/SVG/JSON 导出物和受控本地设计入口 |

本轮一致性审阅：

- `local-review/2026-07-13-fresh-light-design-consistency-review.md`

## 4. 当前一致性判断

当前目录与新版 Fresh Light 思路总体一致，但原有文件主要把浅色方向定位为“v1.2 导入试点 / v1.3+ 探索”。2026-07-13 起调整为：

- Fresh Light 是全应用目标设计基线。
- 代码迁移仍按 Token、基础组件和页面逐步实施，不一次性重写。
- Dark Glass 保留为已经验收的历史基线和可回滚模式。
- v1.2 导入专项文档继续保持冻结，不因全局视觉方向重写业务规则。

详细矩阵见 `local-review/2026-07-13-fresh-light-design-consistency-review.md`。

## 5. Figma 文件定位

### 5.1 v1.2 生产基线

- 文件：`LedgerTwo v1.2 UI System - polar`
- URL：`https://www.figma.com/design/Q4m7LRw75qrkFdw4O5xmU0`
- 既有同步和截图验收：`docs/project_analysis/v1.2-task49-figma-sync-2026-07-09/`

该文件继续作为 v1.2 导入、规则管理和既有 Dark Glass 实现的历史/生产设计基线。

### 5.2 Fresh Light 全应用工作稿

- 文件：`Ledger Two｜双人记账 Web UI Redesign`
- URL：`https://www.figma.com/design/Xsw1qqEkPraqVJCIGkl41Y`
- 本地事实源：`ledger-two-fresh-light-implementation-spec-2026-07-13.md`

该文件是新版 Dashboard、流水、记账、结算、分析、设置和导入工作台的目标工作稿。只有在 Frame、截图和节点完成验证后，才能标记为“已同步”。

### 5.3 历史参考

- 原 `zy j` 账户文件：`https://www.figma.com/design/wkU5RRZs5R7McjNUlEaFF2`
- 原始生成记录：`docs/project_analysis/v1.2-task49-figma-2026-07-09/`

## 6. 本地化保存

仓库默认保存可 diff、可搜索、可供 Codex 查阅的 Markdown 和 JSON。线上 Figma 不应成为唯一事实源。

`local-review/` 可保存：

- 设计审阅 Markdown。
- 无敏感数据的 PNG/PDF/SVG。
- Variables、Tokens、Frame Manifest 和导出元数据 JSON。
- HTML/CSS 原型。

原始 `.fig`、`.figma`、ZIP 和临时文件默认不提交 Git。需要精确评审 `.fig` 时，应同时提供截图和结构化清单。

## 7. 使用顺序

1. 阅读 `ledger-two-design-system-brief.md`，理解阶段边界。
2. 阅读 `ledger-two-fresh-light-implementation-spec-2026-07-13.md`，确认全应用目标和代码禁区。
3. 使用 `ledger-two.design-tokens.json` 和 `ledger-two.figma-variables.json` 建立变量。
4. 按 `ledger-two-frame-manifest.json` 建立页面和 Frame。
5. 使用 `component-library.md` 建立或映射组件。
6. 使用 `v1.1-v1.2-ui-draft-spec.md` 和专项 handoff 完成逐屏状态。
7. 开发前使用 `handoff-checklist.md`。
8. 完成后将截图、CDP 指标或未验证原因写入 `local-review/` 或 `docs/project_analysis/`。

## 8. 阶段结论

- v1.1 的 Dark Glass 验收事实不回写或抹除。
- v1.2 导入工作台保持现有业务和验收边界。
- Fresh Light 作为 v1.2 收口后的目标设计，通过 UI-FL 原子任务逐页迁移。
- 任何主题迁移都必须保留回滚方式，并在 375/390/430/1440 视口和真实业务路径中验收。
