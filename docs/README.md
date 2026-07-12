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
13. `docs/prd/30-prd-v1.2-xlsx-import-special.md`
14. `docs/prd/27-acceptance-case-matrix.md`
15. `docs/prd/28-transaction-caliber-supplement.md`
16. `docs/tech/18-short-mid-architecture-slices.md`
17. `docs/tech/24-v1.2-xlsx-import-implementation-plan.md`
18. `docs/api/API_INVENTORY.md`
19. `docs/api/API_CONVENTIONS.md`
20. `docs/api/openapi.yaml`
21. `docs/ui/14-v1.1-v1.2-module-flows.md`
22. `docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`
23. `docs/project_analysis/2026-07-12-local-wsl-xlsx-csv-preview-acceptance.md`
24. `docs/project_analysis/v1.2-task49x-ui-acceptance-2026-07-12/README.md`
25. `docs/project_analysis/2026-07-12-real-wechat-bill-import-readiness.md`
26. `docs/project_analysis/2026-07-12-v1.2-xlsx-special-predevelopment-review.md`
27. `docs/codex_tasks/README.md`
28. 进入具体业务模块文档。

当前项目已完成 Task01-Task49。Task49X 核心实现、本机 schema 19 和真实 CSV/XLSX preview 已完成，但支付宝真实 XLSX、视觉验收和 NAS schema 19 发布尚未关闭。后续开发以 `docs/project_analysis/2026-07-12-local-wsl-xlsx-csv-preview-acceptance.md`、`docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`、专项 PRD/DEV 为主要入口。

## AI 开发使用方式

让 AI 编码时，不要让它一次性实现全项目。推荐提示：

```text
请先阅读 docs/README.md、docs/prd/README.md、docs/tech/README.md，
然后只实现【某一个模块】。输出计划后等待确认，不要直接开始大范围修改。
```

