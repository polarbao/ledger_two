# 技术文档模块目录（Post-Task30 Foundation）

本目录用于描述 LedgerTwo 当前 v1.0 后的技术架构状态，以及进入 v1.1 前必须补足的基础框架。

## 文件列表

```text
00-current-architecture-after-task30.md       Task30 后当前架构说明
01-architecture-stack.md                      原总体架构与技术选型，保留参考
02-backend-modules.md                         原后端模块设计，后续需要按 Foundation 文档更新
03-frontend-modules.md                        原前端模块设计
04-database-api.md                            原数据库与 API 设计
05-settlement-algorithm.md                    分摊与结算算法
06-import-export-backup.md                    导入、导出、备份恢复
07-cross-platform-tech.md                     跨端技术方案
08-nas-deployment.md                          NAS 部署方案
09-test-quality.md                            测试与质量保障
13-foundation-framework-before-v1.1.md        v1.1 前基础技术框架总览
14-configuration-security-deployment.md       配置安全与部署一致性
15-ledger-context-rbac.md                     LedgerContext 与 RBAC 权限框架
16-api-contract-openapi-error.md              API 契约、OpenAPI、错误码
17-data-migration-test-quality.md             数据迁移、测试与质量门禁
```

## 技术原则

1. 后端：Go + SQLite + REST JSON。
2. 前端：React + TypeScript + Vite。
3. 金额：统一 int64 cents，禁止 float 金额计算。
4. 结算：只生成 settlement 记录，不修改历史账单。
5. 删除：soft delete。
6. 统计：以后端聚合为准，前端只展示。
7. 账本上下文：所有业务 API 必须明确 ledger。
8. 权限：后端统一校验，前端只做体验层控制。
9. 数据安全：备份、恢复、导出、附件访问必须有权限和审计策略。
