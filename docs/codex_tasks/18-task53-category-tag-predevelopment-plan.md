# Task53 分类、标签与导入智能归类专项准备计划

状态：Task53P.1-P.6 准备包已关闭；Task53.1-Task53.4C 已完成，下一实现切片为 Task53U，随后执行 Task53.5<br>
创建日期：2026-07-16<br>
当前准入：Task53.1 已落地 schema 22 与默认元数据；Task53.2 已按 TDD 完成纯分类器、候选读取和默认关闭开关；Task53.3 准备完整<br>
环境边界：只在临时测试数据库执行 migration 022；不得升级现有 WSL/NAS 数据库

## 1. Goal

把用户提出的自动分类、自动标签和默认元数据需求转化为闭环的产品、技术、数据、API、UI、测试与发布准备包，并在不破坏 Task49 导入可信边界的前提下，为后续原子开发提供唯一入口。

## 2. Priority and coordination

执行优先级：

```text
Task53P 准备完成
-> Task50.5-Task50.6 已关闭
-> Task53.1 数据与默认元数据已完成
-> Task53.2 纯分类器与规则兼容
-> Task51P.1 继续非约束性证据收集
```

说明：Task53 编号晚于 Task51/Task52，但准备工作已提前完成。Task53.1 已关闭；后续继续按 53.2 -> 53.3 -> 53.4 -> 53U -> 53.5 串行推进。Task51 仍只是非约束性证据准备，Task52 仍延后。

## 3. Preparation gates

### Task53P.1：现状与竞品研究

产物：

- `docs/project_analysis/2026-07-16-category-tag-competitive-research.md`

完成标准：当前实现、钱迹、小星记账、Actual Budget 和 Firefly III 的可吸收机制、非目标和来源可追溯。

当前状态：已完成并纳入准备基线。

### Task53P.2：PRD

产物：

- `docs/prd/34-prd-v1.3-category-tag-intelligence.md`

冻结问题：

1. 分级自动化边界。
2. 默认分类/标签包。
3. 显式学习和规则优先级。
4. 标签上限、兜底分类和归档行为。
5. Owner 权限与本地隐私。

当前状态：已完成评审基线；用户已确认采用分级自动化，其他默认项按本文推荐值冻结，仍可在编码前提出调整。

### Task53P.3：Tech/API/Migration

产物：

- `docs/tech/26-v1.3-category-tag-intelligence-contract.md`
- `docs/tech/27-v1.3-category-tag-migration-review.md`
- `docs/api/openapi-v1.3-category-tag-draft.yaml`

完成标准：模块边界、候选/决策模型、历史规则兼容、schema 22、错误码、API 和回滚逻辑互相一致。

当前状态：草案已建立；YAML、schema 引用和 migration 静态评审纳入本轮验证。

### Task53P.4：Fixture/Acceptance

产物：

- `docs/fixtures/category-tag/README.md`
- `docs/fixtures/category-tag/expected/*.json`
- `docs/project_analysis/2026-07-16-task53-predevelopment-readiness.md`

完成标准：规则排序、冲突、标签上限、默认包、学习、跨账本和 migration 守恒均有确定性用例。

当前状态：Fixture 规格与 4 个 expected JSON 已建立。

### Task53P.5：UI/UX/Figma

产物：

- `docs/ui/17-v1.3-category-tag-intelligence-flows.md`
- `docs/ui/figma/task53-v1.3-category-tag/`
- required Frame manifest、组件状态矩阵和本地审阅证据

完成标准：375/390/430/1440 覆盖自动、建议、兜底、手工、冲突、学习、批量接受和基础包流程。

当前状态：交互规格、Frame Manifest 和组件状态矩阵已建立；视觉审阅稿与线上 Figma 同步仍未验证，Task53U 前补证据。

### Task53P.6：Detailed implementation readiness

前置：P1-P5 形成一致基线。

必须产出：

1. 原子任务、文件所有权和提交边界。
2. TDD failing tests、实现顺序和验证命令。
3. schema 21 -> 22 临时库/开发库演练方案。
4. Task53 独立端口、数据库、镜像标签和 feature flag。
5. WSL/NAS 不部署声明与最终发布准入。

产物：`docs/codex_tasks/19-v1.3-task53-detailed-implementation-plan.md`。

当前状态：已完成。实际编码仍需用户确认进入 Task53.1。

## 4. Candidate implementation slices

以下切片已在 Task 19 中细化；本表保留为快速导航：

| Task | Candidate scope | Dependency |
|---|---|---|
| Task53.1 | migration 022、默认 profile 定义、初始化/新账本原子 seed | P1-P6 |
| Task53.2 | classifier 纯函数、规范化、规则候选、built-in/fallback | 53.1 |
| Task53.3 | preview persistence、DTO、解释、reclassify | 53.2 |
| Task53.4 | 批量调整、显式 learn、规则来源/apply mode | 53.3 |
| Task53U | 导入状态、行编辑、批量 UI、规则管理、基础包 UI | 53.3/53.4 |
| Task53.5 | Fixture、migration、浏览器、feature flag、回滚与发布验收 | 53U |

每个实现 Task 单独提交；不得把 53.1-53.5 一次性提交。

## 5. File ownership risks

Task53 实现与已冻结 Task50 边界的协调点：

| File/module | Task53 need | Coordination |
|---|---|---|
| `backend/internal/ledger/repo.go` | 新账本默认 profile | Task53.1 独占；不得改写 Task50 账本/成员不变量 |
| `backend/internal/http/router/router.go` | 新 API | Task50 已关闭；Task53.1 开始时按详细计划登记文件所有权 |
| `docs/api/openapi-v1.3-ledger-draft.yaml` | LedgerCreate metadata extension | 先保持独立 Task53 draft，最终评审再合并 |
| `frontend` ledger create UI | profile selector | Task50.4/50.5 已完成；Task53U 适配现有最终契约 |
| `frontend/src/pages/ImportPage.tsx` | 分类状态与批量操作 | Task53 所有权 |
| `backend/internal/importer/*` | classifier 集成 | Task53 所有权 |

## 6. Environment

准备命名建议：

```text
APP_ENV=development
DB_PATH=<repo-external>/task53-development.db
IMPORT_CLASSIFICATION_MODE=graded
IMAGE_TAG=task53-dev-<commit>
```

规则：

1. 不复用 v1.2 production、NAS staging 或当前 38091 Task50 schema 21 staging 数据库。
2. 不把真实微信/支付宝账单复制为 Fixture。
3. schema 22 只从匿名 schema 21 副本和独立 development DB 演进。
4. Task53.5 只部署到新的独立 WSL staging 做验收，不覆盖当前 Task50 staging；NAS 仍需单独维护窗口。

## 7. Validation for preparation

1. 文档入口、PRD、Tech、API、UI、Fixture、Task 双向链接。
2. OpenAPI YAML 可解析且 `$ref` 无悬空项。
3. 默认 profile system_key 唯一、支出/收入兜底各一项。
4. PRD 与 Tech 对 apply/suggest/fallback/learn 定义一致。
5. Task50、Task53.1 和 Task53.2 已关闭，Task53.3 已具备开发准入；Task51/Task52 门禁保持一致。
6. `git diff --check`、真实数据/数据库/密钥审计。

## 8. Rollback

准备文档可在用户评审后修订或标记 defer。Task53.1 已创建只增量的 schema 22；回滚遵循 migration review 的备份恢复和应用前滚策略，不对真实数据库执行 down。

## 9. Next review

编码前默认采用以下已推荐基线；用户仍可明确调整：

1. 默认分类与标签名单是否符合使用习惯。
2. 用户规则历史默认保持“仅建议”是否接受。
3. 兜底分类是否必须存在且不可无替代归档。
4. 标签上限 8 个是否接受。
5. Task53 完成后是否继续 Task51P.1 证据评审，仍由届时证据决定。

详细原子开发计划已经形成，下一任务为 Task53.3；不得跳过 DTO/持久化 snapshot failing tests、commit 不重分类证明或独立环境边界。后续准备细则见 `../tech/28-v1.3-task53-post-classifier-readiness.md`。
