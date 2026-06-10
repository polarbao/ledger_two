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
```

## 技术原则

- 后端：Go + SQLite + REST JSON。
- 前端：React + TypeScript + Vite。
- 金额：统一 int64 cents，禁止 float。
- 结算：只生成 settlement 记录，不修改历史账单。
- 删除：soft delete。
- 统计：以后端聚合为准，前端只展示。
