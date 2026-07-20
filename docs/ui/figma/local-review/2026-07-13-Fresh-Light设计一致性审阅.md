# 2026-07-13 Fresh Light 设计一致性与本地化审阅

状态：已审阅并本地化  
审阅范围：`docs/ui/figma/`、`docs/ui/Fresh-Light视觉参考原型.html`、最新 PRD/技术/UI 文档、关键 React 页面<br>
新设计本地事实源：`../2026-07-13-Fresh-Light界面实施规范.md`

## 1. 结论

当前 `docs/ui/figma/` **总体方向与新版 Fresh Light 设计一致，但并非完全一致**。

一致部分：

- 已经把 `Fresh-Light视觉参考原型.html` 定义为浅色财务方向的来源。
- 已经存在 Fresh Light 和 Dark Glass 双模式 Token。
- 已经要求状态使用颜色、文字和图标，不只依赖颜色。
- 已经定义移动卡片、桌面表格、记账抽屉、结算解释、导入工作台和高风险确认。
- 已经建立 `local-review/`，允许在无法依赖线上 Figma 权限时进行本地审阅。

主要缺口：

1. 既有文档主要把 Fresh Light 限定为 v1.2 导入工作台试点或 v1.3+ 探索，缺少“全应用目标设计 + 分阶段迁移”的明确定位。
2. Frame Manifest 只有 Light Dashboard Exploration，没有完整覆盖 Dashboard、流水、记账、结算、分析和设置的 Fresh Light 目标稿。
3. 缺少一份可直接交给 Codex 的页面级实施规格、代码映射、优先级和禁区。
4. README 只记录旧的 v1.2 主 Figma 文件，没有记录本轮新的 Redesign 工作文件和本地化规格。
5. `local-review/` 虽然已有入口，但尚未保存本轮 Fresh Light 审阅结论和实施事实源。

本次采用的处理方式：

- 保留现有 v1.1/v1.2 冻结事实，不把历史文档重写成“已完成全局主题迁移”。
- 将 Fresh Light 定位为 **v1.2 收口后的目标设计基线**，代码按页面分阶段迁移。
- 保留 Dark Glass 作为可回滚模式。
- 新增 Codex 实施规格和本地一致性审阅。
- 更新目录索引、设计方向、逐屏规格和 Frame Manifest。

## 2. 文件逐项判断

| 文件 | 一致性 | 是否更新 | 判断 |
|---|---|---:|---|
| `README.md` | 部分一致 | 是 | 结构正确，但缺少新版工作稿、本地事实源和双 Figma 文件定位 |
| `LedgerTwo设计系统方向说明.md` | 部分一致 | 是 | Fresh Light 语言一致，但阶段判断仍偏“仅导入试点/未来探索” |
| `ledger-two.design-tokens.json` | 一致 | 否 | Fresh Light 色彩、状态、断点、圆角和 Dark Glass 回滚模式均符合新版方向 |
| `ledger-two.figma-variables.json` | 基本一致 | 否 | 已有 Fresh Light/Dark Glass 模式；变量覆盖较少但不阻断本轮文档本地化 |
| `ledger-two-frame-manifest.json` | 部分一致 | 是 | 已覆盖 v1.1/v1.2，但缺少全应用 Fresh Light 目标 Frame 和代码映射页 |
| `组件库规范.md` | 一致 | 否 | 组件原则、高风险确认、移动卡片/桌面表格和 React 映射与新版一致 |
| `v1.1至v1.2与Fresh-Light界面设计稿规范.md` | 部分一致 | 是 | 原有版本稿合理，但需要追加 Fresh Light 全应用目标设计章节 |
| `设计开发交接清单.md` | 一致 | 否 | Token、状态、移动/桌面尺寸和验收证据要求仍适用 |
| `v1.2-task49-*.md` | 一致 | 否 | 属于导入专项冻结资料，不应因全局视觉目标而重写业务验收 |
| `local-review/README.md` | 部分一致 | 是 | 已有机制，但需要说明结构化 Markdown 也应作为默认可提交审阅物 |

## 3. 与新版设计思想的关系

### 3.1 保留的方向

- 浅蓝灰背景、白色表面、绿色主色、深青文字。
- 轻边框、克制阴影、胶囊分段控件和状态 Chip。
- 移动端负责快速录入、筛选和结算。
- 桌面端负责整理、导入和复盘。
- Dashboard 第一屏回答“花了多少、谁垫付、谁应付、是否待确认”。
- 流水页是可处理工作台，不只是静态列表。
- 结算必须解释 paid/share/raw_net/settlement/final_net。
- 导入必须 preview 先于 commit。

### 3.2 新增的方向

- Fresh Light 不再只是一张 Dashboard 探索稿，而是全应用目标系统。
- 迁移方式是 token 先行、基础组件统一、页面分阶段替换，不一次性重写。
- 一级导航保持首页、流水、结算、分析、设置；导入和周期规则收纳为二级工具。
- 记账表单把金额和高频字段放前，模板、标签、备注、附件等低频信息折叠。
- 分析页成员视角改为支付、承担和净垫付，避免“记账人排行”误导。
- Figma Frame 必须明确映射路由和 React 文件。

## 4. Figma 事实源判断

当前仓库记录的 v1.2 主文件：

- `LedgerTwo v1.2 UI System - polar`
- `https://www.figma.com/design/Q4m7LRw75qrkFdw4O5xmU0`

本轮 Fresh Light 新版工作文件：

- `Ledger Two｜双人记账 Web UI Redesign`
- `https://www.figma.com/design/Xsw1qqEkPraqVJCIGkl41Y`

两者应并存：

- `Q4m7...`：已经与 v1.2 导入和既有 Dark Glass 实现关联的生产基线。
- `Xsw1...`：Fresh Light 全应用目标工作稿。
- 仓库中的 Markdown/JSON：不依赖账号权限的本地化事实源。

由于本轮最新 Figma 写入曾受 Starter 调用额度限制，仓库文档不得把 `Xsw1...` 描述为已完成逐屏同步。设计是否完成以 Frame 清单、截图和 Figma 节点验证为准。

## 5. 本地化保存策略

本轮已把新版设计保存为仓库可审阅文本，而不是只保留在聊天或线上 Figma：

```text
docs/ui/figma/2026-07-13-Fresh-Light界面实施规范.md
docs/ui/figma/local-review/2026-07-13-Fresh-Light设计一致性审阅.md
```

后续真实导出文件建议放入：

```text
docs/ui/figma/local-review/fresh-light-2026-07-13/
  00-foundations.pdf
  dashboard-desktop-1440.png
  dashboard-mobile-390.png
  transactions-desktop-1440.png
  transaction-sheet-mobile-390.png
  settlement-mobile-390.png
  import-preview-mobile-390.png
  frame-export-manifest.json
  sha256sums.txt
```

原始 `.fig`、`.figma`、ZIP 和临时文件仍由 `.gitignore` 排除。需要提交 PNG/PDF/SVG 时必须确认无真实账单、账号、邮箱、订单号或其他财务隐私。

## 6. 审阅基线

Codex 或人工开发者评审新版设计时，应同时核对：

1. `2026-07-13-Fresh-Light界面实施规范.md`。
2. `ledger-two-frame-manifest.json`。
3. `ledger-two.design-tokens.json`。
4. `组件库规范.md`。
5. `设计开发交接清单.md`。
6. 对应 React 页面和 API 契约。
7. 375/390/430/1440 截图或 CDP 指标。

## 7. 不因本次设计改变的事实

- 不修改已应用 migration。
- 不在前端重新计算权威分摊或结算。
- 不删除多账本、权限、附件、模板、周期、导入、审计或离线草稿。
- 不把视觉工作稿当作代码已实现证据。
- 不把 Figma 链接当作唯一事实源。
