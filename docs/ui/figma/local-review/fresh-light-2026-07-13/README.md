# LedgerTwo Fresh Light 全套本地预览

状态：基于当前 Figma 版本构建的生成审阅快照<br>
生成日期：2026-07-13<br>
来源分支：`docs/fresh-light-ui-localization-20260713`<br>
Figma 参考：`Ledger Two｜双人记账 Web UI Redesign`（`https://www.figma.com/design/Xsw1qqEkPraqVJCIGkl41Y`）<br>
规范参考：`../../ledger-two-fresh-light-implementation-spec-2026-07-13.md` 与 `../../ledger-two-frame-manifest.json`

本目录依据当前 Fresh Light Figma 工作版本的设计方向，并结合仓库内实施规格和 Frame Manifest 构建。它是便于 Git、浏览器和 Codex 查阅的本地审阅包，不是原始 `.fig` 文件、Figma 节点全量导出或线上同步完成证明。

## 文件

- `fresh-light-preview.html`：生成审阅文件，提供浏览器入口和 29 个 Frame 清单。
- `fresh-light-all-frames.svg`：生成审阅文件，可直接在 GitHub 或浏览器查看全套预览画板。
- `fresh-light-preview-manifest.json`：生成审阅元数据，记录来源、Frame、尺寸、用途和隐私声明。
- `generate_previews.py`：审阅工具，在本地生成 PNG、PDF 和 SHA-256 清单；不负责生成 Figma 节点。

预览包覆盖规范清单中的全部 29 个 Frame。当前审阅稿使用了三个页面简称：`02 Daily Use`、`03 Import`、`06 Future`；对应的规范名称分别是 `02 Fresh Light Daily Use`、`03 Import Workbench`、`06 Future Exploration`，映射记录在 Preview Manifest 的 `page_aliases` 中。

## 生成二进制预览

在仓库根目录运行：

```bash
python docs/ui/figma/local-review/fresh-light-2026-07-13/generate_previews.py
```

需要 Chromium 或 Chrome。输出：

```text
fresh-light-all-frames.png
fresh-light-preview.pdf
sha256sums.txt
```

这些输出默认可用于本地审阅；是否提交 PNG/PDF 应由维护者根据仓库体积策略决定。

## 审阅边界

- 全部成员、金额、商户和时间均为脱敏示例数据。
- 预览是设计目标和信息架构说明，不代表 React 前端已经实现。
- 预览不是 Figma 节点、Variables 或 Auto Layout 已同步的证据。
- 预览中的页面简称和视觉细节用于审阅；正式命名、状态和实现范围以根目录规范文件为准。
- 金额、分摊、结算、权限、导入和备份规则仍以 PRD、技术契约和代码为准。
