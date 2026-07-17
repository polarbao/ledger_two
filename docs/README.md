# LedgerTwo 文档总入口

状态：当前文档入口
最近更新：2026-07-16

本文档用于判断 `docs/` 下哪些资料是当前事实源、哪些只是历史参考或验收证据。后续产品、架构、UI/UX、部署和 AI 开发任务都应先从这里进入，再按具体模块读取。

## 文档目录

```text
docs/
  prd/              当前产品定位、路线图、模块 PRD、验收口径
  tech/             当前架构、模块技术方案、部署与 migration 策略
  ui/               UI/UX 规范、页面流程、Fresh Light 和 Figma 配套资料
  api/              API inventory、OpenAPI 草案和接口契约冻结资料
  releases/         v1.2 发布说明、升级回滚说明和发布验收清单
  codex_tasks/      Task31+ 与版本冻结阶段的 AI/Codex 任务入口
  fixtures/         导入、验收和测试用例所需的非敏感样例说明
  project_analysis/ 阶段分析、验收证据、截图记录和历史归档
```

## 文档角色

后续判断文档时按以下角色处理，避免旧资料覆盖当前事实：

| 角色 | 目录或文件 | 使用规则 |
|---|---|---|
| 当前入口 | `docs/README.md`、`docs/00_DOCUMENT_INDEX.md` | 先读，用于确定阶段、阅读顺序和事实源优先级 |
| 产品事实源 | `docs/prd/README.md`、`docs/prd/00-product-roadmap.md`、`docs/prd/20-34` | 当前产品定位、路线、范围和验收口径 |
| 技术事实源 | `docs/tech/README.md`、`docs/tech/00-current-architecture-after-task30.md`、`docs/tech/18-27` | 当前架构、实施契约、部署隔离和迁移策略 |
| UI/UX 事实源 | `docs/ui/README.md`、`docs/ui/14-17`、`docs/ui/figma/README.md` | 当前页面流程、长期体验专项、Figma 配套规范 |
| 任务入口 | `docs/codex_tasks/README.md` 和当前任务文件 | 只用于执行已确认任务，不替代 PRD/Tech/UI 事实源 |
| 发布证据 | `docs/releases/`、`docs/project_analysis/2026-*` | 记录已执行验收和发布状态，不单独定义新需求 |
| 历史资料 | 根目录早期 `01-18` 文档、`project_analysis/extracted_archives`、旧 zip | 背景参考；不得用于推翻 Task30 后的新能力 |

`docs/ui/figma/` 是当前 UI/UX 体系的一部分，不属于普通历史附件。该目录下的 Token、Variables、Frame Manifest、handoff 和本地审阅包必须按 `docs/ui/figma/README.md` 的事实源层级使用；本地预览不能反向覆盖已冻结的金额、权限、结算、导入和备份规则。

## 当前总览文档

当前已经存在产品和技术总览，不需要再新增平行的总览文档：

| 类型 | 总览入口 | 说明 |
|---|---|---|
| 全局文档入口 | `docs/README.md`、`docs/00_DOCUMENT_INDEX.md` | 判断阶段、阅读顺序和历史/当前边界 |
| PRD 总览 | `docs/prd/README.md`、`docs/prd/00-product-roadmap.md` | 产品定位、版本路线、优先级和 PRD 事实源 |
| 技术总览 | `docs/tech/README.md`、`docs/tech/00-current-architecture-after-task30.md` | 架构现状、技术栈、模块边界和技术事实源 |
| UI/UX 总览 | `docs/ui/README.md`、`docs/ui/15-ledgertwo-ux-optimization-program.md`、`docs/ui/figma/README.md` | 页面流程、长期 UI/UX 专项和 Figma 规范 |

后续如需新增文档，应优先补充到对应目录的 README 或现有总览中；只有新版本、新模块或新发布窗口无法被现有总览承载时，才新增独立正式文档。

## 推荐阅读顺序

1. `../CHANGELOG.md` (版本发布说明)
2. `docs/releases/README.md`
3. `docs/prd/README.md`
4. `docs/tech/README.md`
5. `docs/ui/README.md`
6. `docs/prd/00-product-roadmap.md`
7. `docs/prd/20-product-retrospective-and-positioning.md`
8. `docs/prd/21-roadmap-short-mid-long.md`
9. `docs/prd/22-prd-v1.1-trust-and-daily-use.md`
10. `docs/prd/23-feature-priority-and-deferral-decisions.md`
11. `docs/prd/24-short-mid-module-breakdown.md`
12. `docs/prd/25-prd-v1.1-module-specs.md`
13. `docs/prd/26-prd-v1.2-import-module-specs.md`
14. `docs/prd/29-prd-v1.2-module-business-service-breakdown.md`
15. `docs/prd/30-prd-v1.2-xlsx-import-special.md`
16. `docs/prd/27-acceptance-case-matrix.md`
17. `docs/prd/28-transaction-caliber-supplement.md`
18. `docs/tech/00-current-architecture-after-task30.md`
19. `docs/tech/18-short-mid-architecture-slices.md`
20. `docs/tech/23-v1.2-deployment-environment-isolation.md`
21. `docs/tech/24-v1.2-xlsx-import-implementation-plan.md`
22. `docs/api/API_INVENTORY.md`
23. `docs/api/API_CONVENTIONS.md`
24. `docs/api/openapi.yaml`
25. `docs/ui/14-v1.1-v1.2-module-flows.md`
26. `docs/ui/15-ledgertwo-ux-optimization-program.md`
27. `docs/ui/figma/README.md`
28. `docs/codex_tasks/README.md`
29. `docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`
30. `docs/codex_tasks/13-fresh-light-ui-interaction-plan.md`
31. `docs/codex_tasks/14-v1.3-task50-predevelopment-plan.md`
32. `docs/codex_tasks/15-v1.3-task50-detailed-implementation-plan.md`
33. `docs/codex_tasks/16-v1.3-task50-3-readiness-and-post-task50-entry.md`
34. `docs/prd/34-prd-v1.3-category-tag-intelligence.md`（Task53 已完成准备，代码排期待后续复评）
35. `docs/tech/26-v1.3-category-tag-intelligence-contract.md`
36. `docs/tech/27-v1.3-category-tag-migration-review.md`
37. `docs/api/openapi-v1.3-category-tag-draft.yaml`
38. `docs/ui/17-v1.3-category-tag-intelligence-flows.md`
39. `docs/fixtures/category-tag/README.md`
40. `docs/codex_tasks/18-task53-category-tag-predevelopment-plan.md`
41. `docs/ui/figma/task53-v1.3-category-tag/README.md`
42. `docs/codex_tasks/19-v1.3-task53-detailed-implementation-plan.md`
43. `docs/project_analysis/2026-07-16-task53-predevelopment-readiness.md`
44. `docs/prd/33-task51-scenario-evidence-and-scope-questions.md`（仅 Task51 非约束性发现准备）
45. `docs/project_analysis/2026-07-16-task50-preparation-completeness-and-task51-p1-kickoff.md`
46. `docs/project_analysis/task51_p1/README.md`（Task51P.1 匿名证据工作区）
45. `docs/codex_tasks/17-task51-predevelopment-plan.md`（Task50 技术门禁已满足，正式范围仍等待真实证据）
46. 进入具体业务模块文档。

当前项目已完成 Task01-Task49。Task49X 核心实现、运行开关、本机 schema 19、微信 XLSX/支付宝 CSV 真实 preview 和移动端视觉验收已完成；支付宝当前仍只导出 CSV。后续发布收口聚焦 NAS schema 19 staging、production 一致性备份与逐批导入确认，开发入口以 `docs/project_analysis/2026-07-12-local-wsl-xlsx-csv-preview-acceptance.md`、`docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`、专项 PRD/DEV 为准。

2026-07-17 更新：Task50.1-Task50.6 已完成，独立本机 v1.3/schema 21 staging、回滚和浏览器证据已闭环，NAS 未部署。Task53.1 schema 22 与默认元数据已在本地代码/临时数据库完成，下一实现任务为 Task53.2；Task51P.1 真实证据仍为 0，Task52 继续保持调研门禁。

## AI 开发使用方式

让 AI 编码时，不要让它一次性实现全项目。推荐提示：

```text
请先阅读 docs/README.md、docs/prd/README.md、docs/tech/README.md，
然后只实现【某一个模块】。输出计划后等待确认，不要直接开始大范围修改。
```
