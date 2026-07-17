# LedgerTwo 发布文档入口

状态：Task50 `v1.3.0-rc/schema 21` 独立 staging 已通过；Task53 目标 schema 22 的准备完整但尚未部署验收；现有 v1.2 staging 和 NAS 发布线保持独立

本目录只维护当前发布候选的用户可见变化、升级回滚步骤和验收记录。产品范围与实现细节仍分别以 `docs/prd/`、`docs/tech/` 和 `docs/codex_tasks/` 为事实源。

## 当前文件

1. `v1.3.0-rc-release-notes.md`：Task50 已验收能力以及 Task53 未完成边界。
2. `v1.3.0-rc-upgrade-guide.md`：schema 19 -> 21 已验证路径、Task53 schema 21 -> 22 待执行路径和 NAS 边界。
3. `v1.3.0-rc-checklist.md`：Task50.6 已完成门禁与 Task53 待验收入口。
4. `v1.3.0-task53-rc-acceptance-template.md`：Task53 schema 22 自动化、浏览器、指标和回滚验收模板，当前 `not_run`。
5. `../project_analysis/2026-07-17-task50-6-release-closure.md`：Task50 全模块、升级、回滚和浏览器证据。
6. `v1.2.0-rc-*`：继续保留 v1.2/NAS schema 18/19 历史发布线，不与 v1.3 staging 混用。

## 状态规则

1. `rc` 只表示实现和本机门禁通过，不等同于 NAS 稳定版发布。
2. NAS 升级前必须先生成并下载可恢复备份。
3. Task50 staging 必须确认 schema 21；Task53 专用 staging 必须确认 schema 22、端口 38092 和准确的 `import_classification_mode`，两者不得共用数据库目录。
4. 任一阻断级问题出现时停止发布并执行镜像/数据库成对回滚；不要让旧镜像连接更高 schema。
5. 正式版确认后再把候选文档复制为稳定版本记录并创建对应 Git tag。
6. 本机 schema 19 可用于受控 CSV/XLSX preview；NAS schema 18 仍只能按 CSV 能力验收，不得把本机结果视为 NAS 已发布。
7. preview 只创建导入批次，不等于正式同步；任何真实账单 commit 仍需逐批用户确认。
8. Task50 本机候选通过不等于 Task53 或 NAS 已部署；Task53.5 与 NAS 仍需分别确认。
