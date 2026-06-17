# LedgerTwo Task30 后基础框架补全文档包

生成日期：2026-06-17  
适用阶段：Task01-Task30 已完成、v1.1 具体业务需求尚未审核定稿之前  
输出方式：仅输出本地审核文档，不修改 GitHub 仓库

## 1. 本包目标

本包用于在进入 v1.1 业务开发之前，把当前 Task30 后的 LedgerTwo 项目文档事实源、基础架构要求、UI 基础框架、AI/Codex/Gemini 开发规范和后续基础框架任务计划统一整理清楚。

本包不把之前讨论过的 v1.1 业务规划视为已确认需求。v1.1 中的具体业务能力，例如账号创建模式、邀请机制、共同支付同步策略、分类标签增强等，仍等待产品审核。当前文档仅把 v1.1 作为后续演进大纲，并优先补足 MVP/v1.0 阶段缺失的基础框架。

## 2. 审核原则

审核时建议按以下顺序查看：

1. `docs/reviews/2026-06-17-task30-current-progress-vs-docs-review.md`
2. `docs/00_DOCUMENT_INDEX.md`
3. `docs/prd/00-product-roadmap.md`
4. `docs/prd/11-foundation-framework-before-v1.1.md`
5. `docs/tech/00-current-architecture-after-task30.md`
6. `docs/tech/13-foundation-framework-before-v1.1.md`
7. `docs/ui/12-foundation-framework-ui.md`
8. `docs/codex_tasks/README.md`
9. `docs/codex_tasks/05-foundation-task-plan.md`

## 3. 输出文件结构

```text
docs/
  00_DOCUMENT_INDEX.md
  18_POST_DEMO_AI_CODING_TASKS.md
  reviews/
    2026-06-17-task30-current-progress-vs-docs-review.md
  prd/
    README.md
    00-product-roadmap.md
    11-foundation-framework-before-v1.1.md
    12-current-progress-gap-analysis.md
  tech/
    README.md
    00-current-architecture-after-task30.md
    13-foundation-framework-before-v1.1.md
    14-configuration-security-deployment.md
    15-ledger-context-rbac.md
    16-api-contract-openapi-error.md
    17-data-migration-test-quality.md
  ui/
    README.md
    12-foundation-framework-ui.md
    13-settings-management-redesign.md
  codex_tasks/
    README.md
    00-ai-development-workflow.md
    01-repository-code-style.md
    02-backend-go-style.md
    03-frontend-react-ts-style.md
    04-testing-quality-gates.md
    05-foundation-task-plan.md
    06-review-checklist.md
patches/
  README_apply_after_approval.md
data/
  document-change-set_foundation_before_v1.1.json
```

## 4. 建议应用方式

审核通过后，建议新建分支再同步文档：

```bash
git checkout main
git pull origin main
git checkout -b docs/foundation-before-v1.1
```

然后把本包中的 `docs/` 内容复制覆盖或新增到仓库 `docs/` 目录中。建议本次只提交文档，不改业务代码。

推荐提交信息：

```text
docs: align post-task30 foundation architecture before v1.1
```

## 5. 需要你审核确认的问题

1. 是否接受把 Task01-Task30 标记为已完成，后续任务从 Foundation Task31 开始。
2. 是否接受在 v1.1 业务开发前先做基础框架补齐，而不是直接进入 v1.1 功能实现。
3. 是否接受 `docs/codex_tasks/` 作为后续 Codex/Gemini 的主要任务入口。
4. 是否接受旧的 `docs/18_POST_DEMO_AI_CODING_TASKS.md` 从“后续任务计划”调整为“Task30 完成状态 + 新任务入口说明”。
5. 是否接受把当前 v1.1 具体业务规划继续标记为“未审核，不进入开发”。
