# LedgerTwo 本地 Figma 设计文件审阅入口

状态：已建立，等待设计文件同步

本目录用于接收从 Figma 或 VS Code Figma 扩展导出的本地设计资料，供 Codex 在不依赖线上账号授权的情况下进行设计审阅、实现比对和验收。

## 建议放入的文件

优先提供以下可直接审阅的格式：

1. `*.png`：关键 Frame 的 1x 或 2x 导出图。
2. `*.pdf`：多页面设计稿和流程稿。
3. `*.svg`：图标、Logo 和可编辑矢量资产。
4. `*.json`：Variables、Tokens、Frame Manifest 或组件元数据。
5. `*.html`、`*.css`：可运行原型与样式参考。
6. `*.fig`：Figma 本地副本，可用于归档、哈希核对和后续重新导入 Figma。

`.fig` 是 Figma 的本地工程格式，Codex 不能保证直接解析其完整节点、变量和 Auto Layout 结构。需要精确评审时，应同时提供 PNG/PDF 导出图，以及 Variables 或组件清单 JSON。

## 推荐命名

```text
local-review/
  LedgerTwo-v1.2-main.fig
  LedgerTwo-v1.2-variables.json
  00-foundations.pdf
  01-components.pdf
  03-import-entry-desktop.png
  03-import-preview-mobile-390.png
  03-import-xlsx-errors.pdf
```

文件名应包含版本、页面或 Frame 名称和视口，不要使用 `final-final` 等不可追踪命名。

## 审阅输出

收到本地文件后，Codex 应输出：

1. 文件清单、大小和 SHA-256，不修改原始文件。
2. 与 `ledger-two-frame-manifest.json` 的缺失/重复 Frame 对照。
3. 与前端实现的颜色、字号、间距、响应式和交互状态差异。
4. 可执行的 UI 修改任务、截图验收结果和 Node/组件映射替代记录。
5. 适合进入版本库的脱敏导出物；原始 `.fig` 默认不提交 Git。

## 安全边界

1. 不要放入真实账单、账号、邮箱、订单号或其他财务隐私。
2. 原始 `.fig`、压缩包和临时文件由本目录 `.gitignore` 默认排除。
3. 需要提交 PNG/PDF/SVG 时，先确认无敏感数据并控制文件体积。
4. 不因本地设计稿改变已冻结的金额、权限、导入和结算业务规则。
