# NAS 环境分级与真实数据保全契约

状态：已冻结<br>
日期：2026-07-20<br>
适用范围：LedgerTwo NAS 开发联调、发布体验、升级与公网验收

## 1. Environment decision

| Port | Role | Channel | Data class | LAN | Public HTTP |
|---:|---|---|---|---|---|
| 38088 | 发布/真实体验环境 | `production` | 真实数据，初始化后永久保全 | `http://192.168.0.115:38088` | `http://nas.polarrrr.top:38088` |
| 38092 | 开发联调/验收环境 | `staging` | 测试数据，可重建 | `http://192.168.0.115:38092` | `http://nas.polarrrr.top:38092` |

本地 WSL2 仍是开发者个人开发环境；NAS 38092 是共享开发联调和发布前验收环境，技术 channel 保持 `staging`，不伪装成 production。

环境角色一经确定，不允许互换数据库、密钥、uploads、backups 或 logs。38088 不能用于 Fixture、自动清库和破坏性测试；38092 禁止保存真实账单或用户正式账号。

## 2. Domain fact

规范域名为 `nas.polarrrr.top`：公共 DNS 返回 `101.71.237.198`。2026-07-20 NAS-R1 重建后再次使用外部多地区 HTTP 节点验证：38088 为 5/5 HTTP 200；38092 首轮 4/5 HTTP 200、1 个节点超时，立即复验为 5/5 HTTP 200。

`nas.polarrr.top` 少一个 `r`，公共 DNS 状态为 NXDOMAIN；当前主机经本地代理访问时返回 502。文档、环境变量和验收记录不得使用该拼写。

公网验证只证明 DNS、端口转发和 HTTP health 可达，不等于安全发布。目前两个入口均为明文 HTTP，真实账号初始化、登录和账单访问只允许走 LAN，直到反向代理、可信 TLS 证书和 secure cookie 完成独立门禁。

## 3. Release data rules

38088 首次真实初始化后：

1. `data/ledger.db`、uploads、backups、logs 和 production `.env` 归类为不可丢弃数据。
2. 应用升级只能挂载原 production 数据目录执行向前 migration，禁止用空库、38092、WSL 或构建上下文覆盖。
3. 每次升级前生成 SQLite 一致性备份，并在 NAS 内外各保留一份及 SHA-256。
4. 升级门禁必须比较用户、账本、交易、split、settlement、导入引用、附件引用、金额和 import hash。
5. 回滚使用固定旧镜像与升级前数据库成对恢复，不执行 migration down。
6. 任何清库、恢复、删除用户数据或重建 production 都必须获得新的明确确认。

## 4. Staging rules

38092 只使用匿名 Fixture 或专门 QA 账号：

1. 可随任务重建数据库，但操作前必须确认 channel、port、container 和 runtime root。
2. 不复制 38088 数据；需要迁移测试时使用仓库匿名 Fixture 或经审批的脱敏副本。
3. 候选先在 38092 完成 schema、health、浏览器、回滚和权限验收，再安排 38088 维护窗口。
4. staging 的成功不能替代 production 备份、迁移和数据守恒验证。

## 5. Public testing boundary

允许通过公网执行无状态 health、静态页面和无真实凭据的浏览器检查。以下操作在 HTTPS 门禁完成前禁止经公网 HTTP 执行：

1. `/init` 创建真实账号和密码。
2. 登录、Cookie 会话和账单增删改查。
3. CSV/XLSX 上传、附件、导出或备份下载。
4. 管理员诊断、恢复和任何包含个人数据的接口。

## 6. Current deployment baseline

| Item | 38088 production | 38092 staging |
|---|---|---|
| image | `ledger-two:1.3.0-rc-task53-98c3b14` | `ledger-two:1.3.0-rc-task53-98c3b14` |
| runtime root | `/volume1/docker/ledger-two` | `/volume1/docker/ledger-two-development` |
| container | `ledger-two-v13-production` | `ledger-two-v13-development` |
| health | schema 22 / suggest / db ok | schema 22 / suggest / db ok |
| initialization | `true`，真实数据保全已生效 | `false`，保持空 QA 环境 |
| permissions | root 0700，env/db 0600 | root 0700，env/db 0600 |

旧 38089 和旧 Task53 staging runtime 已下线并删除。38088 已完成 LAN 初始化并建立 NAS 内外 schema 22 基线备份，本契约第 3 节的数据保全规则已经生效；38092 仍保持未初始化的可重建 QA 环境。

## 7. Source of truth

1. 当前环境与数据保全：本文。
2. 一次性清库换代：`../codex_tasks/20-NAS-R1真实体验发布与数据保全计划.md`。
3. Task53 staging 证据：`../project_analysis/2026-07-20-Task53-NAS预发布部署与权限加固.md`。
4. 发布升级与回滚：`../releases/v1.3.0-rc-升级与回滚指南.md`。
5. NAS-R1 实际执行记录：`../project_analysis/2026-07-20-NAS-R1双环境重建与真实数据入口.md`。
