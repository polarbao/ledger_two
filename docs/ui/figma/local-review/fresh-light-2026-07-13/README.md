# LedgerTwo Fresh Light 全套本地预览

生成日期：2026-07-13  
分支：`docs/fresh-light-ui-localization-20260713`  
来源：`../../ledger-two-fresh-light-implementation-spec-2026-07-13.md` 与 `../../ledger-two-frame-manifest.json`

## 文件

- `fresh-light-preview.html`：浏览器入口和 29 个 Frame 清单。
- `fresh-light-all-frames.svg`：可直接在 GitHub 或浏览器查看的全套预览画板。
- `fresh-light-preview-manifest.json`：Frame、尺寸、用途和隐私声明。
- `generate_previews.py`：在本地生成 PNG、PDF 和 SHA-256 清单。

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
- 金额、分摊、结算、权限、导入和备份规则仍以 PRD、技术契约和代码为准。
