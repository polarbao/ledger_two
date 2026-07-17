# LedgerTwo 发布文档入口

状态：本机 `v1.3.0-rc/schema 21` 独立 staging 已通过；现有 v1.2 staging 和 NAS 发布线保持独立

本目录只维护当前发布候选的用户可见变化、升级回滚步骤和验收记录。产品范围与实现细节仍分别以 `docs/prd/`、`docs/tech/` 和 `docs/codex_tasks/` 为事实源。

## 当前文件

1. `v1.3.0-rc-release-notes.md`：Task50 多账本候选能力、范围和已知风险。
2. `v1.3.0-rc-upgrade-guide.md`：schema 19 -> 21、独立 staging、成对回滚和 NAS 边界。
3. `v1.3.0-rc-checklist.md`：Task50.6 本机候选门禁。
4. `../project_analysis/2026-07-17-task50-6-release-closure.md`：全模块、升级、回滚和浏览器证据。
5. `v1.2.0-rc-*`：继续保留 v1.2/NAS schema 18/19 历史发布线，不与 v1.3 staging 混用。

## 状态规则

1. `rc` 只表示实现和本机门禁通过，不等同于 NAS 稳定版发布。
2. NAS 升级前必须先生成并下载可恢复备份。
3. v1.3 staging 升级后必须确认 `/api/healthz` 返回 `version=1.3.0-rc`、`schema_version=21`、`deployment_channel=staging`、`db=ok`。
4. 任一阻断级问题出现时停止发布，恢复升级前 schema 19 数据库与旧镜像；不要让 v1.2 镜像连接 schema 21。
5. 正式版确认后再把候选文档复制为稳定版本记录并创建对应 Git tag。
6. 本机 schema 19 可用于受控 CSV/XLSX preview；NAS schema 18 仍只能按 CSV 能力验收，不得把本机结果视为 NAS 已发布。
7. preview 只创建导入批次，不等于正式同步；任何真实账单 commit 仍需逐批用户确认。
8. 本机 v1.3 候选通过不等于 NAS 已部署；NAS 仍需单独维护窗口。
