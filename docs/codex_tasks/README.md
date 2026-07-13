# Codex / Gemini 开发任务入口

状态：当前任务入口
适用阶段：Task01-Task49 已完成；v1.2 RC 通过 Task49X 重新打开 XLSX 输入范围；Fresh Light 波次 A（UI-FL-01/02）已完成，UI-FL-03 待启动

## 1. 目标

本目录用于给 Codex、Gemini、Cursor、Copilot 或其他 AI 编码工具提供明确、可执行、可验收的开发任务和代码风格规范。

后续所有 AI 开发任务都应从本目录开始，而不是直接让 AI 阅读零散文档后自由发挥。

## 2. 文件列表

```text
00-ai-development-workflow.md   AI 开发工作流和通用提示词
01-repository-code-style.md     仓库通用代码风格和提交规范
02-backend-go-style.md          Go 后端代码风格
03-frontend-react-ts-style.md   React + TypeScript 前端代码风格
04-testing-quality-gates.md     测试与质量门禁
05-foundation-task-plan.md      Task31-Task40 基础框架任务计划
06-review-checklist.md          人类审核清单
07-reference-style-sources.md  代码风格参考来源
08-product-roadmap-dev-plan.md  产品路线对应的 Task41+ DEV 任务计划
09-task41-49-detailed-plan.md   Task41-Task49 细化开发任务规格
10-task33-40-detailed-plan.md   Task33-Task40 细化开发任务规格
11-v1.2-release-hardening-plan.md v1.2 RC 环境隔离、NAS staging 与 production 升级任务
12-v1.2-xlsx-import-special-plan.md v1.2 微信/支付宝 XLSX 导入专项任务
13-fresh-light-ui-interaction-plan.md v1.2 收口后的 Fresh Light UI/UX 协同开发计划
../releases/                    v1.2 发布说明、升级回滚和发布验收清单
```

## 3. AI 开发强制流程

1. 读取 `docs/00_DOCUMENT_INDEX.md`。
2. 读取 `docs/prd/11-foundation-framework-before-v1.1.md`。
3. 产品规划类任务读取 `docs/prd/20-product-retrospective-and-positioning.md` 到 `docs/prd/23-feature-priority-and-deferral-decisions.md`。
4. 短中期业务开发读取 `docs/prd/24-short-mid-module-breakdown.md`、`docs/prd/25-prd-v1.1-module-specs.md`、`docs/prd/26-prd-v1.2-import-module-specs.md`。
5. 短中期冻结或开发前读取 `docs/prd/27-acceptance-case-matrix.md` 和 `docs/prd/28-transaction-caliber-supplement.md`。
6. v1.2 Task47-Task49 开发前额外读取 `docs/prd/29-prd-v1.2-module-business-service-breakdown.md` 和 `docs/tech/20-v1.2-import-implementation-contract.md`。
7. Task49X 开发前必须读取 `docs/prd/30-prd-v1.2-xlsx-import-special.md`、`docs/tech/24-v1.2-xlsx-import-implementation-plan.md` 和 `docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`。
8. 读取 `docs/tech/18-short-mid-architecture-slices.md`、`docs/tech/19-short-mid-implementation-readiness.md` 和 `docs/ui/14-v1.1-v1.2-module-flows.md`。
9. Fresh Light 或后续业务 Task 涉及 UI 时，读取 `docs/ui/figma/ledger-two-fresh-light-implementation-spec-2026-07-13.md` 和 `docs/codex_tasks/13-fresh-light-ui-interaction-plan.md`，登记共享组件归属和并行冲突。
10. Foundation 开发读取 `docs/codex_tasks/10-task33-40-detailed-plan.md`。
11. 读取本目录代码风格文档。
12. 读取对应任务。
13. 输出计划和预计修改文件，等待确认。
14. 只实现当前任务。
15. 运行测试和构建。
16. 输出变更摘要、验证命令、风险和下一步建议。

## 4. 禁止事项

1. 禁止一次性实现多个 Foundation Task。
2. 禁止实现未审核 v1.1 业务需求。
3. 禁止把权限判断只放在前端。
4. 禁止使用 float 计算金额。
5. 禁止修改历史 migration。
6. 禁止提交真实数据库、备份、上传文件和密钥。
7. 禁止绕过测试直接声称完成。
