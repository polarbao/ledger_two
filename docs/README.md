# LedgerTwo 文档总入口

本文档目录按「PRD / UI / 技术」三类重新组织，并在每一类下按业务模块拆分，便于后续让 AI/Codex 按模块开发、评审和补充。

## 文档目录

```text
docs/
  prd/      产品需求文档，按业务模块拆分
  ui/       UI 交互设计文档，按页面和交互模块拆分
  tech/     技术设计与实现文档，按工程模块拆分
  api/      API inventory、OpenAPI 草案和接口契约冻结资料
  releases/ v1.2 发布说明、升级回滚说明和发布验收清单
  codex_tasks/      Task31+ 与版本冻结阶段的 AI/Codex 任务入口
  project_analysis/ 项目分析、压缩包解压内容和文档冲突评估
```

## 推荐阅读顺序

1. `../CHANGELOG.md` (版本发布说明)
2. `docs/releases/README.md`
3. `docs/prd/README.md`
4. `docs/prd/00-product-roadmap.md`
5. `docs/prd/20-product-retrospective-and-positioning.md`
6. `docs/prd/21-roadmap-short-mid-long.md`
7. `docs/prd/22-prd-v1.1-trust-and-daily-use.md`
8. `docs/prd/23-feature-priority-and-deferral-decisions.md`
9. `docs/prd/24-short-mid-module-breakdown.md`
10. `docs/prd/25-prd-v1.1-module-specs.md`
11. `docs/prd/26-prd-v1.2-import-module-specs.md`
12. `docs/prd/29-prd-v1.2-module-business-service-breakdown.md`
13. `docs/prd/27-acceptance-case-matrix.md`
14. `docs/prd/28-transaction-caliber-supplement.md`
15. `docs/tech/18-short-mid-architecture-slices.md`
16. `docs/api/API_INVENTORY.md`
17. `docs/api/API_CONVENTIONS.md`
18. `docs/api/openapi.yaml`
19. `docs/ui/14-v1.1-v1.2-module-flows.md`
20. `docs/project_analysis/2026-07-06-short-mid-long-and-nas-readiness.md`
21. `docs/reviews/2026-06-17-task30-current-progress-vs-docs-review.md`
22. `docs/codex_tasks/README.md`
23. `docs/project_analysis/2026-07-09-v1.2-freeze-readiness.md`
24. 进入具体业务模块文档。

当前项目已完成 Task01-Task49，Foundation、v1.1 和 v1.2 导入模块均已收口，当前进入 `1.2.0-rc` 冻结阶段。早期 Demo / v0.3 文档仍保留为历史约束和实现背景，但后续开发任务以 `docs/codex_tasks/`、`docs/prd/24-29`、`docs/tech/18-21` 和最新 `docs/project_analysis/` 冻结记录为主要入口。

## AI 开发使用方式

让 AI 编码时，不要让它一次性实现全项目。推荐提示：

```text
请先阅读 docs/README.md、docs/prd/README.md、docs/tech/README.md，
然后只实现【某一个模块】。输出计划后等待确认，不要直接开始大范围修改。
```

