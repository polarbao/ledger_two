# Task49X NAS schema 19 发布就绪检查

日期：2026-07-13
范围：本机与 NAS 只读预检、staging 发布保护脚本；未执行远程迁移或真实批次提交

## 1. 当前结论

Task49X 核心代码和本机 schema 19 仍可工作；NAS staging/production 均保持 schema 18。本轮补齐了 staging schema 19 自动回滚脚本，但 NAS Docker daemon 需要维护者交互式 sudo，因此没有绕过权限启动、停止或替换任何远程容器。

支付宝真实 XLSX 样本仍未提供，Task49X.4 继续保持部分完成。Figma 改走 `docs/ui/figma/local-review/` 接收本地设计文件，不影响本轮 NAS 技术门禁判断。

## 2. 本机基线

```text
URL: http://localhost:38088
version: 1.2.0-rc
schema_version: 19
deployment_channel: staging
import_xlsx_enabled: true
db: ok
PRAGMA quick_check: ok
transactions: 40
```

本机正式账单数量未变化。当前仍保留既有 preview 批次，不执行真实批次 commit。

## 3. NAS 只读预检

局域网 health：

```text
production http://192.168.0.115:38088: schema 18 / production / db ok
staging    http://192.168.0.115:38089: schema 18 / staging / db ok
```

隔离条件：

1. production 位于 `/volume1/docker/ledger-two`。
2. staging 位于 `/volume1/docker/ledger-two-staging`。
3. 两边使用不同数据库文件；staging 数据库当前大小为 352256 字节。
4. `/volume1/docker` 可用空间约 5.3 TiB，空间不是当前阻塞。
5. SSH 免密连接正常；普通用户不能访问 Docker socket，`sudo -n` 不可用。

本轮没有读取或记录 JWT_SECRET、Cookie、密码 Hash、真实账单内容和完整 `.env` 值。

## 4. 新增发布保护

`deploy/nas/promote-staging.sh` 在修改 staging 前强制检查：

1. channel 必须为 `staging`，端口必须为 `38089`。
2. `IMPORT_XLSX_ENABLED` 必须显式为 `true`。
3. 当前与回滚数据库必须都是 schema 18 且 `quick_check=ok`。
4. schema 18 staging health 必须正常，候选镜像和旧容器必须存在。
5. 候选启动后必须回读 schema 19、staging、XLSX 开关开启和 db ok。
6. 失败时保存故障数据库，恢复 schema 18 数据库和旧 staging 容器。
7. 可通过 `ENV_FILE`、`COMPOSE_FILE` 使用旁路候选配置，不在维护窗口前覆盖当前 schema 18 配置。

配套测试覆盖成功升级和 health 失败自动回滚，使用临时数据库与 fake Docker，不接触真实 NAS 数据。

## 5. 已准备的发布材料

固定候选提交：

```text
commit: af42edf00999565eb1262ad6f2b45c0df3e87c34
remote bundle: /volume1/docker/ledger-two-staging/incoming/af42edf00999
```

候选包中的源码、Compose、环境样例和发布脚本均通过 SHA-256 校验；远端解包后的两个 Shell 脚本通过 `sh -n`。仓库新增 `.gitattributes`，强制 `*.sh` 在 Git 归档中使用 LF，避免 Windows 打包后的 CRLF 在 NAS `/bin/sh` 下解析失败。候选随后补入 `ENV_FILE`、`COMPOSE_FILE` 旁路能力并以 `af42edf00999` 重新打包，旧包不用于发布。

staging schema 18 一致性备份：

```text
NAS: /volume1/docker/ledger-two-staging/backups/predeploy/task49x-schema18-0139d16-20260713-112816
schema_version: 18
row_counts users|ledgers|transactions|settlements: 2|1|7|1
quick_check: ok
ledger.db SHA-256: 695a41eb37544f64e016f3868ac5e368cc53e17850f041e94f61a5410f9cd01f
```

备份已复制到本机仓库外的 `nas_management_docs/ledger_two_backups/`，数据库和附件压缩包 SHA-256 均复核一致。候选源码已准备到 staging 的旁路 `app/`，并生成 `.env.schema19`、`docker-compose.schema19.yml`；当前 `docker-compose.yml`、`.env`、数据库和运行容器均未替换。

## 6. 下一执行窗口

维护者需要在 NAS 交互式输入 sudo，顺序为：

1. 已创建 staging schema 18 一致性备份并在 NAS 外保存、核对 SHA-256。
2. 已同步固定提交候选包并准备旁路 app/Compose/env，没有复制本机数据库或覆盖当前配置。
3. 维护者交互式 sudo 构建 `ledger-two:1.2.0-rc` 候选镜像。
4. 更新 staging Compose 和 `.env` 后运行 `promote-staging.sh`，验证 schema 19 health 和 quick_check。
5. 完成登录、历史数据、CSV/XLSX preview、重启持久化和数量守恒。
6. 继续保持 production schema 18 与 `IMPORT_XLSX_ENABLED=false`。

在该窗口完成前，不关闭 RC05 的 NAS schema 19 门禁，也不安排 production schema 19。
