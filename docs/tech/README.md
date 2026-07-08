# 技术文档模块目录

本目录按照工程模块拆分 LedgerTwo 的技术设计与实现方案。

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
```

## 技术原则

- 后端：Go + SQLite + REST JSON。
- 前端：React + TypeScript + Vite。
- 金额：统一 int64 cents，禁止 float。
- 结算：只生成 settlement 记录，不修改历史账单。
- 删除：soft delete。
- 统计：以后端聚合为准，前端只展示。

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
