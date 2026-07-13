# LedgerTwo 本地 Figma 设计文件审阅入口

状态：已建立并开始保存 Fresh Light 审阅记录  
最近更新：2026-07-13

本目录用于接收和保存从 Figma、VS Code Figma 扩展或设计讨论中产生的本地设计资料，使 Codex 在不依赖线上账号授权的情况下完成设计审阅、实现比对和验收。

## 1. 默认保存内容

优先提交可 diff、可搜索、可长期维护的文件：

1. `*.md`：设计决策、差异审阅、实现映射和验收记录。
2. `*.json`：Variables、Tokens、Frame Manifest、组件元数据和导出清单。
3. `*.png`：关键 Frame 的 1x/2x 无敏感数据导出图。
4. `*.pdf`：多页面设计稿、组件库和流程稿。
5. `*.svg`：图标、Logo 和可编辑矢量资产。
6. `*.html`、`*.css`：可运行原型和样式参考。
7. `*.fig`：仅用于本地归档、哈希核对和重新导入，默认不提交 Git。

`.fig` 无法保证被 Codex 精确解析完整节点、Variables 和 Auto Layout。需要精确评审时，必须同时提供 PNG/PDF、Variables JSON、Frame Manifest 或组件清单。

## 2. 当前 Fresh Light 本地事实源

```text
../ledger-two-fresh-light-implementation-spec-2026-07-13.md
2026-07-13-fresh-light-design-consistency-review.md
../ledger-two-design-system-brief.md
../v1.1-v1.2-ui-draft-spec.md
../ledger-two-frame-manifest.json
../ledger-two.design-tokens.json
../ledger-two.figma-variables.json
```

以上文件是当前新版设计的仓库内事实源。即使线上 Figma 暂时不可编辑或调用额度受限，Codex 仍可依据这些文档进行评审和任务拆分。

## 3. 推荐导出目录

```text
local-review/
  2026-07-13-fresh-light-design-consistency-review.md
  fresh-light-2026-07-13/
    00-foundations.pdf
    01-components.pdf
    dashboard-desktop-1440.png
    dashboard-mobile-390.png
    transactions-desktop-1440.png
    transactions-mobile-390.png
    transaction-sheet-mobile-390.png
    settlement-desktop-1440.png
    settlement-mobile-390.png
    settings-desktop-1440.png
    import-preview-desktop-1440.png
    import-preview-mobile-390.png
    frame-export-manifest.json
    sha256sums.txt
```

文件名必须包含日期或版本、页面/Frame 和视口。不要使用 `final-final`、`new2` 等不可追踪名称。

## 4. 审阅输出

收到本地文件后，Codex 应输出：

1. 文件清单、大小和 SHA-256，不修改原始文件。
2. 与 `ledger-two-frame-manifest.json` 的缺失、重复和额外 Frame 对照。
3. 与 Fresh Light Token、逐屏规格和当前前端实现的差异。
4. 颜色、字号、间距、响应式、交互状态和可访问性问题。
5. Frame 到 React 页面/组件的映射。
6. 可执行的 UI-FL 任务、测试命令和截图验收路径。
7. 适合进入版本库的脱敏导出物。

## 5. 两个 Figma 文件的定位

- v1.2 生产基线：`https://www.figma.com/design/Q4m7LRw75qrkFdw4O5xmU0`
- Fresh Light 工作稿：`https://www.figma.com/design/Xsw1qqEkPraqVJCIGkl41Y`

线上文件不是唯一事实源。只有完成 Frame/节点、截图或本地导出物验证后，才能在审阅记录中标记为“已同步”。

## 6. 安全边界

1. 不放入真实账单、账号、邮箱、订单号或其他财务隐私。
2. 原始 `.fig`、`.figma`、压缩包和临时文件由 `.gitignore` 默认排除。
3. 提交 PNG/PDF/SVG 前确认无敏感信息并控制体积。
4. 不因本地设计稿改变已冻结的金额、权限、导入、结算和备份规则。
5. 不在没有证据时宣称 Figma、代码或 NAS 已同步。
