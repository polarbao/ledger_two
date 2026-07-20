# NAS-R1：真实体验发布与数据保全专项计划

状态：环境分级、域名和删除范围已确认；NAS-R1 破坏性执行已获授权<br>
创建日期：2026-07-20<br>
优先级：插入 Task51P.1 之前执行<br>

## 1. Goal

清理 NAS 上历史演示/测试数据，让用户在最新已验收候选上自行创建真实账号和账本；首次初始化后，production 数据进入长期保全状态，后续版本升级只能迁移和验证，不得用空库、staging 数据或新镜像覆盖。

## 2. Scope

1. 分类 NAS 上的 production、staging 和误建目录。
2. 对现有 production 执行一次经确认的清库换代，并让用户通过 `/init` 自行创建账号和密码。
3. 删除或重置两个 staging 的匿名/演示数据库。
4. 建立 production 数据备份、升级、回滚和禁止操作规则。
5. 保持 staging 可丢弃、production 不可丢弃的物理边界。

## 3. Non-goals

1. 本专项不替用户创建账号、设置密码或导入真实账单。
2. 不把 `1.3.0-rc` 宣称为稳定版；它是当前最新、已通过 WSL2/NAS staging 的固定候选。
3. 本专项只验证现有公网 DNS/HTTP，不配置 TLS 或反向代理。
4. 真实账号不通过公网明文 HTTP 初始化；用户只从 LAN `/init` 输入密码。

## 4. Current state

版本性质以 `DEPLOYMENT_CHANNEL`、端口、runtime root 和数据责任共同判定，不能仅凭版本字符串是否带 `rc` 判断。

| Runtime | Health | Data | Classification | Handling |
|---|---|---|---|---|
| `/volume1/docker/ledger-two` / 38088 | `1.2.0-rc` / production / schema 18 | 2 users、1 ledger、7 transactions、1 settlement | 当前生产实例，尽管版本名为 RC | 仅本次按用户明确要求重置；之后永久保全 |
| `/volume1/docker/ledger-two-staging` / 38089 | `1.2.0-rc` / staging / schema 18 | 2/1/7/1 | 旧测试实例 | 可停止并删除 runtime/data |
| `/volume1/docker/ledger-two-task53-staging` / 38092 | `1.3.0-rc` / staging / schema 22/suggest | 2 users、2 ledgers、835 transactions、2 settlements | 当前 Task53 匿名测试实例 | 可重置为空库，继续作为 QA staging |
| `/volume1/docker/ledger-two\r` | 无服务、无数据库 | 空 | 历史误建目录 | 可删除 |

当前不存在 v1.3 stable tag 或已发布的 v1.3 production。建议把固定镜像 `ledger-two:1.3.0-rc-task53-98c3b14` 作为“真实体验 production 候选”切换到 38088，但发布文档仍保持 RC 口径。

环境角色已冻结：38088 是发布/真实体验 production；38092 是开发联调/验收 staging。规范公网域名是 `nas.polarrrr.top`，不是 `nas.polarrr.top`。公共 DNS 返回 `101.71.237.198`，两个端口均已由 5 个外部 HTTP 节点验证为 200。

## 5. Proposed approach

### 5.1 One-time production reset

1. 停止旧 `ledger-two` production，阻止 SQLite 继续写入。
2. 使用 SQLite 一致性备份把旧 production 数据、环境配置和 SHA-256 保存到仓库外的本机 NAS 管理备份目录。
3. 删除 NAS 上旧 production 的 data/backups/uploads/logs 和历史部署残留，重新创建 0700 runtime；`.env`、数据库和备份固定 0600。
4. 使用新的 production 专用 secret、`DEPLOYMENT_CHANNEL=production`、38088 和固定 Task53 镜像启动空 schema 22 数据库。
5. 验证 health 为 production/schema 22/suggest/db ok，`/api/init/status` 为 `initialized=false`。
6. 由用户访问 `http://192.168.0.115:38088/init` 自行创建账本、两个账号和密码。
7. 初始化成功后立即生成第一份 production 基线备份；从此禁止再次执行清库流程。

旧 production 备份保存在 NAS 外，不进入 Git。用户完成新账号验收后，是否销毁该临时备份需要第二次明确确认。

### 5.2 Staging cleanup

1. 停止并删除旧 38089 容器及 `/volume1/docker/ledger-two-staging`，不再维护重复的 v1.2 staging。
2. 停止 38092，删除匿名数据库、升级备份、uploads/logs/evidence，再以同一固定镜像创建空 schema 22 staging。
3. 38092 保持 `DEPLOYMENT_CHANNEL=staging` 和独立 secret；不得使用用户在 38088 创建的真实账号或数据库。
4. 删除空的误建目录 `/volume1/docker/ledger-two\r`。

## 6. Production data retention policy

首次真实初始化后，production 必须遵守：

1. 版本升级前停止写入，执行 SQLite `.backup`、`quick_check`、外键、核心数量、金额和 hash 基线。
2. 备份至少一份留在 NAS backups，一份保存到 NAS 外；两份都记录 SHA-256。
3. 新镜像必须使用固定提交标签，先用匿名 staging 验证，再连接 production 原数据库执行向前 migration。
4. 升级只能保留并迁移原 `ledger.db`；禁止复制空库、staging 库或 WSL 数据覆盖 production。
5. 回滚采用“旧固定镜像 + 升级前数据库备份”成对恢复，不运行 migration down。
6. 用户、账本、交易、split、settlement、导入引用、附件和审计记录均纳入守恒检查。
7. production 的 data/backups/uploads/logs 不参与代码同步、镜像解压或目录清理。
8. 任何 production 清库、恢复或数据删除都必须再次获得用户明确确认。

## 7. Execution steps and completion gates

| Step | Action | Completion gate |
|---|---|---|
| NAS-R1.1 | 盘点版本、目录、端口、schema 和数量 | 已完成，本文件记录与 health 一致 |
| NAS-R1.2 | 用户确认精确删除和 production 切换范围 | 已完成；授权按推荐口径执行 |
| NAS-R1.3 | 旧 production NAS 外备份与停机 | quick_check=ok，SHA-256 可复核 |
| NAS-R1.4 | 清理旧 runtime，部署空 v1.3 RC production | 38088 health 正确，init=false |
| NAS-R1.5 | 清理/重建 staging 和误建目录 | 38089 下线，38092 空库且 init=false |
| NAS-R1.6 | 用户自行初始化 production | 用户确认可登录，账号不写入文档/日志 |
| NAS-R1.7 | 建立首份真实 production 基线备份 | 备份 hash、health、核心数量记录完整 |

## 8. Risks

| Risk | Control | Blocker |
|---|---|---|
| 删除了真实旧数据 | 先停写并保存 NAS 外一致性备份 | 备份/哈希未通过 |
| 把 staging 当 production | 强校验 channel/root/port/container | 任一环境字段不匹配 |
| 新镜像覆盖未来真实库 | 数据目录独立，升级脚本禁止复制 data | 发现空库替换 production |
| 用户密码进入命令或文档 | 只允许用户在 `/init` 页面输入 | 任何明文凭据出现在仓库/日志 |
| RC 被误称稳定版 | UI/文档继续标记 RC，另行稳定发布 | 未有 stable tag 却声明 stable |
| 清理时容器仍持有 SQLite | 先 stop，再 backup/delete/start | 容器未停止 |

## 9. Validation

1. 38088：production/schema 22/suggest/db ok，初始化前 `initialized=false`。
2. 用户初始化后：登录成功、默认元数据存在、空账本无历史演示交易。
3. 38089：端口关闭，旧 staging 容器和目录不存在。
4. 38092：staging/schema 22/suggest/db ok，初始化状态为 false 或仅保留明确 QA 账号。
5. production/staging 的数据库 inode、路径和 SHA-256 不同。
6. production runtime 目录为 0700，`.env`/数据库/备份为 0600。
7. 新初始化后的基线备份 quick_check、外键和恢复抽检通过。
8. 公网 `nas.polarrrr.top:38088/38092` health 可达；真实账号只从 LAN 初始化。

## 10. Rollback

在用户完成新 production 初始化前，如新候选启动或 init 失败，停止新容器，恢复旧 `1.2.0-rc` 固定镜像、旧 `.env` 和 NAS 外一致性备份。用户完成真实初始化并确认后，旧演示数据不再回灌；后续回滚只能使用新 production 的版本化备份。

## 11. Confirmation gate

用户已确认在完成环境和域名标识后执行以下破坏性范围：

1. 停止并替换 38088 当前 production。
2. 删除 NAS 上旧 production 活跃数据和历史部署目录。
3. 删除 38089 staging 容器/目录。
4. 删除并重建 38092 staging 数据库。
5. 删除带回车符的误建空目录。

执行口径：把最新固定 `v1.3.0-rc-task53-98c3b14` 切换为 38088 真实体验 production；旧 production 先备份到 NAS 外再从 NAS 删除；删除旧 38089、重建 38092，并删除误建空目录。用户凭据由用户在 LAN `/init` 页面设置。
