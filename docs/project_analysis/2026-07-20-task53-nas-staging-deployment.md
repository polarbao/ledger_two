# Task53 NAS staging 部署与权限加固记录

日期：2026-07-20<br>
结论：Task53 独立 NAS staging 部署成功；LAN health 通过，production 未变

## 1. Deployment

| Item | Result |
|---|---|
| runtime root | `/volume1/docker/ledger-two-task53-staging` |
| container | `ledger-two-v13-task53-nas-staging` |
| image | `ledger-two:1.3.0-rc-task53-98c3b14` |
| LAN URL | `http://192.168.0.115:38092` |
| channel/schema/mode | staging / 22 / suggest |
| health | db ok / XLSX enabled / version 1.3.0-rc |
| production | unchanged |

部署使用独立目录、端口、Compose project、容器、密钥和匿名 schema 22 数据库，没有覆盖 `/volume1/docker/ledger-two`、既有 Task50 staging 或 v1.2 数据。

NAS evidence `task53-20260720-161837` 记录 `before_schema=22`、`after_schema=22`、quick_check/foreign_key_check 为 ok，业务 invariants 与 import hashes diff 均为 0。LAN 端口 38092 从当前主机可建立 TCP 并返回正确 health。

## 2. Access boundary

1. 已验证：`http://192.168.0.115:38092` 可从当前局域网访问。
2. 未验证：公网域名没有在本轮配置。
3. Tailscale 地址 `http://100.68.103.94:38092` 从当前主机返回 HTTP 502，因此不能声明可用；SSH alias `nas` 仍可通过 Tailscale 管理。
4. classification mode 固定为 suggest；合格 learned-rule 样本不足时不得改为 graded。

## 3. Permission finding and fix

首次 NAS 验收发现 Synology 目录继承让 root 生成的 evidence 和升级备份呈现 0777。虽然当前使用匿名 staging 数据，这仍不符合发布资产最小权限要求。

根因是验证脚本依赖调用环境默认 umask，没有在创建备份/evidence 前设置私有权限。处理结果：

1. `verify-task53-staging.sh` 在任何运行目录创建前固定 `umask 077`。
2. 新增脚本安全契约测试，要求 umask 位于首个 `mkdir` 之前并通过 `sh -n`。
3. NAS 现有 runtime/data/evidence/backups 目录已收紧为 0700，`.env` 和 `ledger.db` 为 0600。
4. NAS 上用临时目录验证 `umask 077` 实际生成 0700 目录和 0600 文件；WSL2 专用验证脚本重新执行通过。

## 4. Remaining boundary

NAS staging 已可用于同局域网人工验收，但不等于 production 发布。生产升级仍需独立备份、维护窗口、目标域名/TLS、production 密钥和用户确认；不得直接把 staging 匿名数据库替换为生产数据。
