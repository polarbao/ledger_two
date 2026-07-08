# LedgerTwo Figma 配套设计包

状态：建议采纳  
创建日期：2026-07-08  
适用阶段：v1.1 收口、v1.2 导入模块、v1.3+ 长期 UI/UX 专项

## 1. 目的

本目录用于把 LedgerTwo 的 PRD、UI 文档、当前 React 前端和 `docs/ui/lynntest(1).html` 原型沉淀成可供 Figma 建模、设计评审和前端实现使用的结构化文件。

本目录不是替代代码，也不是一次性换皮方案。它的作用是：

1. 给 v1.1 收口提供统一的 UI/UX 风格判断。
2. 给 v1.2 导入工作台提供逐屏设计稿和组件规格。
3. 给后续真正写入 Figma 文件时提供变量、页面、组件和 frame 清单。
4. 给前端实现提供 token、状态、断点和验收口径。

## 2. 文件清单

| 文件 | 用途 |
|---|---|
| `ledger-two-design-system-brief.md` | 设计方向、当前 UI 差异、v1.1 是否调整的判断 |
| `ledger-two.design-tokens.json` | 面向前端和设计系统的 token 草案 |
| `ledger-two.figma-variables.json` | 面向 Figma Variables 的变量集合草案 |
| `ledger-two-frame-manifest.json` | Figma 页面、Frame 和组件建模清单 |
| `component-library.md` | 组件库规格、状态和前端映射 |
| `v1.1-v1.2-ui-draft-spec.md` | v1.1/v1.2 逐屏 UI 设计稿说明 |
| `handoff-checklist.md` | 每次 UI 设计输出和开发交接检查清单 |

## 3. 使用顺序

1. 先阅读 `ledger-two-design-system-brief.md`，确认本轮视觉方向和阶段边界。
2. 将 `ledger-two.figma-variables.json` 转为 Figma Variables 的集合和模式。
3. 按 `ledger-two-frame-manifest.json` 建立 Figma 页面、Frame 和组件分组。
4. 使用 `component-library.md` 建立组件库。
5. 使用 `v1.1-v1.2-ui-draft-spec.md` 逐屏出设计稿。
6. 开发前用 `handoff-checklist.md` 做交接检查。

## 4. 阶段结论

v1.1 不建议在冻结前做大规模视觉迁移。当前深色玻璃体系已经完成多轮真实 UI 验收，适合继续作为 v1.1 收口基线。

`lynntest(1).html` 的浅色、清爽、绿色财务感方向更适合作为 v1.2 导入工作台和 v1.3+ 全局主题演进的设计基线。短期做法是先用 token 和 Figma 设计包统一语言，避免直接重写现有页面造成回归。

