# LedgerTwo 发布文档入口

状态：`v1.2.0-rc` 发布候选冻结

本目录只维护当前发布候选的用户可见变化、升级回滚步骤和验收记录。产品范围与实现细节仍分别以 `docs/prd/`、`docs/tech/` 和 `docs/codex_tasks/` 为事实源。

## 当前文件

1. `v1.2.0-rc-release-notes.md`：候选版本能力、范围边界和已知风险。
2. `v1.2.0-rc-upgrade-guide.md`：从现有 v1.0/v1.1 数据升级到 schema 18 的备份、升级、验证和回滚步骤。
3. `v1.2.0-rc-checklist.md`：本地质量门禁与 NAS 发布窗口的逐项验收记录。
4. `../project_analysis/2026-07-12-v1.2-nas-production-upgrade-acceptance.md`：NAS staging/production 升级、数据保留与备份证据。

## 状态规则

1. `rc` 只表示实现和本机门禁通过，不等同于 NAS 稳定版发布。
2. NAS 升级前必须先生成并下载可恢复备份。
3. 升级后必须确认 `/api/healthz` 返回 `version=1.2.0-rc`、`schema_version=18`、`db=ok`。
4. 任一阻断级问题出现时停止发布，恢复升级前数据库与旧镜像；不要仅回退镜像后继续使用 schema 18 数据库。
5. 正式版确认后再把候选文档复制为稳定版本记录并创建对应 Git tag。
