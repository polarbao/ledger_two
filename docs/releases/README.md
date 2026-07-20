# LedgerTwo 发布文档入口

状态：NAS-R1 双环境重建和 production 基线备份已完成；38088 已初始化并进入真实数据保全，38092 是未初始化的开发联调/验收 staging

本目录只维护当前发布候选的用户可见变化、升级回滚步骤和验收记录。产品范围与实现细节仍分别以 `docs/prd/`、`docs/tech/` 和 `docs/codex_tasks/` 为事实源。

## 当前文件

1. `v1.3.0-rc-发布说明.md`：Task50 与 Task53 已验收能力、suggest-only 和 NAS 边界。
2. `v1.3.0-rc-升级与回滚指南.md`：schema 19 -> 21 与 Task53 schema 21 -> 22 路径、NAS 边界。
3. `v1.3.0-rc-验收清单.md`：Task50.6 和 Task53 已执行门禁。
4. `v1.3.0-Task53-RC验收记录.md`：Task53 schema 22 自动化、浏览器、指标、回滚和 `pass_with_suggest_only` 记录。
5. `v1.3.0-Task53-RC验收记录模板.md`：后续 Task53 复验模板，不代表当前实际状态。
6. `../project_analysis/2026-07-17-Task50.6全模块与发布收口验收.md`：Task50 全模块、升级、回滚和浏览器证据。
7. `v1.2.0-rc-*`：继续保留 v1.2/NAS schema 18/19 历史发布线，不与 v1.3 staging 混用。
8. `../project_analysis/2026-07-20-NAS-R1双环境重建与真实数据入口.md`：38088/38092 重建、离站备份、公网 health 与初始化门禁证据。

## 状态规则

1. `rc` 只表示实现和本机门禁通过，不等同于 NAS 稳定版发布。
2. NAS 升级前必须先生成并下载可恢复备份。
3. Task50 staging 必须确认 schema 21；Task53 专用 staging 必须确认 schema 22、端口 38092 和准确的 `import_classification_mode`，两者不得共用数据库目录。
4. 任一阻断级问题出现时停止发布并执行镜像/数据库成对回滚；不要让旧镜像连接更高 schema。
5. 正式版确认后再把候选文档复制为稳定版本记录并创建对应 Git tag。
6. 本机 schema 19 可用于受控 CSV/XLSX preview；NAS schema 18 仍只能按 CSV 能力验收，不得把本机结果视为 NAS 已发布。
7. preview 只创建导入批次，不等于正式同步；任何真实账单 commit 仍需逐批用户确认。
8. Task53 固定候选已通过 NAS-R1 单独授权部署到 38088 production；当前 `initialized=true` 且基线备份已建立，但仍不等于稳定版发布。
9. NAS 环境固定为 38088 production、38092 staging；规范公网域名为 `nas.polarrrr.top`。HTTPS 完成前，真实凭据和账单只能经 LAN 使用。
10. NAS-R1.7 已生成 NAS 内外双份 schema 22 基线备份并记录 SHA-256；后续升级不得再以空库替换 production。
