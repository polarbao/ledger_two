# 技术文档模块目录

状态：当前技术事实源入口
最近更新：2026-07-15

本目录按照工程模块拆分 LedgerTwo 的技术设计与实现方案。

## 总览判断

当前技术文档已有总览文档，不需要新增平行的架构总览：

| 层级 | 文件 | 作用 |
|---|---|---|
| 当前架构总览 | `00-current-architecture-after-task30.md` | Task30 后后端、前端、数据库、部署和风险总览 |
| 短中期架构切片 | `18-short-mid-architecture-slices.md` | v1.1/v1.2 后端、前端、数据模型、API、服务层和测试切片 |
| 实施就绪评审 | `19-short-mid-implementation-readiness.md` | 判断文档充分性、执行顺序和不建议启动的任务 |
| 导入实施契约 | `20-v1.2-import-implementation-contract.md` | v1.2 导入 API、状态机、DTO、权限、数据模型和回滚策略 |
| 部署隔离总览 | `23-v1.2-deployment-environment-isolation.md` | development/staging/production 物理隔离、发布顺序和运行开关 |
| XLSX 专项方案 | `24-v1.2-xlsx-import-implementation-plan.md` | Task49X reader、migration 019、安全、测试和回滚 |

后续技术规划应优先更新这些总览和契约。只有当 v1.3 新能力完成 PRD 范围冻结，且现有总览无法承载新的架构边界时，才新增独立技术总览或 ADR。

## 当前技术阶段

截至 2026-07-14，当前架构判断如下：

1. Go + SQLite + REST JSON + React/Vite 的总体选型继续成立，不需要整体推倒重写。
2. `development`、`staging`、`production` 应继续通过部署实例、物理目录、端口、密钥、数据库文件隔离；不能为了“统一部署”共享数据目录。
3. v1.2 RC 的关键技术门禁是 schema 19 staging、XLSX 开关、备份链、health 校验和回滚脚本。
4. Fresh Light 属于前端体验专项，不应改变后端金额、权限、导入、结算或 migration 契约。
5. v1.3 前应重新评审多账本、多成员、多人分摊的数据模型、权限矩阵和 migration 策略。
6. Task50 当前只进入 PRD/Tech/UI/OpenAPI/migration 准备；现有 LedgerContext 和成员 API 是盘点基线，不等于多账本生命周期已经冻结。

当前已知技术债：

1. 交易表单、交易 service/repository、导入工作台仍需按领域逐步拆分。
2. API/OpenAPI、实际 handler、前端类型需要随每个版本继续同步。
3. 大包体和前端分包是性能专项，不应混入业务发布门禁。
4. 旧 Demo 文档和历史压缩包不能作为当前实现依据。

## 文件列表

```text
01-architecture-stack.md       总体架构与技术选型
02-backend-modules.md          后端模块设计
03-frontend-modules.md         前端模块设计
04-database-api.md             数据库与 API 设计
05-settlement-algorithm.md     分摊与结算算法
06-import-export-backup.md     导入、导出、备份恢复
07-cross-platform-tech.md      跨端技术方案
08-nas-deployment.md           NAS 部署方案
23-v1.2-deployment-environment-isolation.md v1.2 staging/production 与数据库物理隔离
09-test-quality.md             测试与质量保障
13-foundation-framework-before-v1.1.md Foundation before v1.1 技术方案
14-configuration-security-deployment.md 配置、安全与部署
15-ledger-context-rbac.md      LedgerContext 与 RBAC
16-api-contract-openapi-error.md API 契约、OpenAPI 与错误码
17-data-migration-test-quality.md 数据迁移、测试与质量门禁
18-short-mid-architecture-slices.md 短中期模块架构切片
19-short-mid-implementation-readiness.md 短中期实施就绪评审
20-v1.2-import-implementation-contract.md v1.2 导入模块实施契约
21-v1.2-import-migration-review.md v1.2 导入模块 Migration 评审
22-v1.2-import-task47-implementation-plan.md v1.2 Task47 导入预览实施计划
24-v1.2-xlsx-import-implementation-plan.md v1.2 Task49X XLSX 导入专项实施方案
```

## 技术原则

- 后端：Go + SQLite + REST JSON。
- 前端：React + TypeScript + Vite。
- 金额：统一 int64 cents，禁止 float。
- 结算：只生成 settlement 记录，不修改历史账单。
- 删除：soft delete。
- 统计：以后端聚合为准，前端只展示。
- 部署：staging/production 必须物理隔离，schema 与镜像成对升级和回滚。
- UI：Figma 和 Fresh Light 只能约束表现层，不得覆盖金额、权限、导入、结算和备份契约。

## 冲突处理

技术文档发生冲突时按以下顺序判断：

1. 当前代码、migration、测试和已执行命令结果。
2. 最新发布/验收记录。
3. 本目录 `00`、`18-24` 当前技术事实源。
4. `docs/prd/` 当前 PRD 与验收口径。
5. `docs/ui/` 当前 UI/UX 规范。
6. 早期 `01-17` 技术文档和根目录 Demo 文档。

## 当前推荐入口

Task30 后的技术规划建议优先阅读：

1. `00-current-architecture-after-task30.md`
2. `13-foundation-framework-before-v1.1.md`
3. `18-short-mid-architecture-slices.md`
4. `19-short-mid-implementation-readiness.md`
5. `20-v1.2-import-implementation-contract.md`（进入 Task47-Task49 前必读）
6. `21-v1.2-import-migration-review.md`（进入 Task47-Task49 前用于确认 migration 切片）
7. `22-v1.2-import-task47-implementation-plan.md`（Task47 开工时用于确认 parser、repository、service、handler 和前端切片）
8. `../prd/29-prd-v1.2-module-business-service-breakdown.md`（进入 Task47-Task49 前用于确认业务对象、服务边界和 UI 工作台）
9. `24-v1.2-xlsx-import-implementation-plan.md`（Task49X 开发前用于确认依赖、reader、migration 019、安全和测试边界）
