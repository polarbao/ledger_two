# LedgerTwo 文档总入口

本文档目录按「PRD / UI / 技术」三类重新组织，并在每一类下按业务模块拆分，便于后续让 AI/Codex 按模块开发、评审和补充。

## 文档目录

```text
docs/
  prd/      产品需求文档，按业务模块拆分
  ui/       UI 交互设计文档，按页面和交互模块拆分
  tech/     技术设计与实现文档，按工程模块拆分
```

## 推荐阅读顺序

1. `docs/prd/README.md`
2. `docs/prd/00-product-roadmap.md`
3. `docs/tech/01-architecture-stack.md`
4. `docs/ui/01-layout-navigation.md`
5. 进入具体业务模块文档。

## AI 开发使用方式

让 AI 编码时，不要让它一次性实现全项目。推荐提示：

```text
请先阅读 docs/README.md、docs/prd/README.md、docs/tech/README.md，
然后只实现【某一个模块】。输出计划后等待确认，不要直接开始大范围修改。
```

