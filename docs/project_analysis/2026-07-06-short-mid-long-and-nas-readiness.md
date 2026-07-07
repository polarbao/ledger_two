# LedgerTwo 短中长期规划与 NAS 部署就绪复盘

日期：2026-07-06

## 1. 结论

短期 Foundation before v1.1 已完成基础冻结能力，可以标记完成并进入中期 v1.1。

当前已完成或基本完成：

| 任务 | 状态 | 证据 |
|---|---|---|
| Task31 文档事实源收口 | 基本完成 | `docs/00_DOCUMENT_INDEX.md`、`docs/README.md`、`docs/codex_tasks/` 已成为入口 |
| Task32 配置与部署安全 | 基本完成 | `backend/internal/config/config.go`、`.env.example`、`docker-compose.yml`、配置测试 |
| Task33 LedgerContext 与 RBAC | 基本完成 | `internal/ledger`、路由 `WithLedgerContext`、权限测试 |
| Task34 API 契约与 OpenAPI | 基本完成 | `docs/api/API_INVENTORY.md`、`API_CONVENTIONS.md`、`openapi.yaml` |
| Task35 分类/标签/账户管理基础 | 基本完成 | metadata API、归档/恢复、设置页入口 |
| Task36 前端 LedgerProvider 与 Query Key | 基本完成 | ledger store、query key、PermissionGate |
| Task37 设置页信息架构重组 | 基本完成 | 设置页分区、元数据管理入口 |
| Task38 迁移、测试与质量门禁 | 完成 | migration 回归、R01/R02、CI compose gate |
| Task39 附件访问控制 | 完成 | `/api/attachments/{filename}` 受保护访问，裸 `/uploads/*` 关闭，R03 覆盖 |
| Task40 审计与系统诊断中心 | 完成 | `GET /api/admin/diagnostics`、设置页诊断面板、Owner-only 权限和脱敏回归 |

因此当前状态应定义为：

```text
Foundation before v1.1: 已完成基础冻结。
可进入 NAS 试部署/内测。
中期 v1.1 已解锁，并已从 Task44 分类、标签、账户管理体验收口开始；首个切片为元数据排序能力。
```

## 2. 中长期规划充分性

现有中长期规划已经具备体系：

- `docs/prd/21-roadmap-short-mid-long.md`
- `docs/prd/24-short-mid-module-breakdown.md`
- `docs/prd/25-prd-v1.1-module-specs.md`
- `docs/prd/26-prd-v1.2-import-module-specs.md`
- `docs/prd/27-acceptance-case-matrix.md`
- `docs/prd/28-transaction-caliber-supplement.md`
- `docs/tech/18-short-mid-architecture-slices.md`
- `docs/tech/19-short-mid-implementation-readiness.md`
- `docs/codex_tasks/09-task41-49-detailed-plan.md`
- `docs/codex_tasks/10-task33-40-detailed-plan.md`

判断：

1. v1.1、v1.2 的 PRD/DEV 框架足够支撑继续开发。
2. 模块边界、非目标、依赖顺序和验收样例已明确。
3. 长期 v1.3+ 仍保持方向级规划是合理的，不应过早写死详细 schema。

仍需补齐的逻辑：

| 缺口 | 处理结论 |
|---|---|
| Foundation 当前完成状态 | 已在本文和任务文档中补充 |
| NAS 部署当前可行性 | 需要以“试部署可行，冻结前需 Task40”表述 |
| Task40 细化程度 | 需要补诊断接口、诊断面板、审计规范的实施清单 |
| 旧 NAS 文档冲突 | 旧 `docs/06_NAS_DEPLOYMENT.md` 应标记为历史文档 |

## 3. NAS 部署就绪判断

当前项目可以部署到远程 NAS 进行内测，但需要满足条件。

可部署条件：

1. NAS 已安装 Container Manager 或 Docker Compose。
2. 使用当前根目录 `docker-compose.yml` 和 `.env.example` 派生 `.env`。
3. `JWT_SECRET` 已替换为强随机值。
4. 挂载目录可写：`data/`、`backups/`、`uploads/`、`logs/`。
5. 访问方式为局域网或 Tailscale；当前本机与 NAS 已处于同一局域网，可优先使用 `192.168.0.115` 进行内网访问验证；公网必须使用 HTTPS 反向代理。
6. 部署前完成一次本地或 CI Docker build 验证。

当前限制：

1. 本机未安装 Docker CLI，本轮无法在本机实际执行 `docker compose config` 或 `docker compose build`。
2. 局域网地址 `192.168.0.115` 已由用户确认可用于本机直接访问 NAS；后续试部署应优先以 `http://192.168.0.115:38088` 作为内网访问地址。
3. Tailscale 地址 `100.68.103.94` 仍可作为异地或备用访问路径，但当前本机已在 NAS 同一局域网内，局域网验证优先级更高。
4. 历史检查中 `curl --noproxy "*" http://100.68.103.94:38088/api/healthz` 返回旧版健康检查，仅包含 `db/status/version`，没有当前版本应有的 `schema_version`；首页 `Last-Modified` 为 2026-06-11，说明当时 NAS 运行的是旧部署。该结论在重新部署前仍作为旧部署风险记录保留。
5. 早先 `admin@100.68.103.94` 与 `root@100.68.103.94` 的 SSH 免密认证均不可用；如后续恢复部署，应优先按本机与 NAS 同局域网条件重新验证 SSH、Docker Compose 和服务端口。
6. 当前代码部署后，`GET /api/healthz` 应返回基础状态和 schema version；Owner 登录后可通过设置页系统诊断或 `GET /api/admin/diagnostics` 检查备份/上传/日志目录状态。

## 4. 推荐 NAS 部署方案

### 4.1 目录

```text
/volume1/docker/ledger-two/
  docker-compose.yml
  .env
  data/
  backups/
  uploads/
  logs/
```

### 4.2 配置

从仓库复制：

```bash
cp .env.example .env
```

生产必须修改：

```text
APP_ENV=production
APP_PORT=38088
APP_BASE_URL=http://192.168.0.115:38088
JWT_SECRET=<64 chars random secret>
COOKIE_SECURE=false
COOKIE_SAMESITE=Lax
```

如果使用 HTTPS 反向代理：

```text
APP_BASE_URL=https://ledger.example.com
COOKIE_SECURE=true
```

### 4.3 启动

```bash
docker compose -f docker-compose.yml config
docker compose up -d --build
docker compose logs -f
```

### 4.4 验收

1. 访问 `http://192.168.0.115:38088/api/healthz`。
2. 首次打开页面完成初始化。
3. 上传一张附件，确认 `uploads/` 目录出现文件。
4. 创建 private 账单并关联附件，确认另一用户无法访问附件。
5. 点击手动备份，确认 `backups/manual/` 出现 `.db` 文件。
6. 重启容器，确认账本和附件仍存在。

## 5. 后续执行

1. 短期计划已完成并标记冻结。
2. 中期处理从 Task44 开始，先稳定分类、标签、账户管理体验；当前优先完成排序能力、归档确认和移动端密度。
3. 在 CI、部署机或 NAS 上运行 Docker build 和 compose config。
4. 如需要公网访问，先完成域名解析、HTTPS 证书、反向代理和 Cookie Secure 配置，再开放外网。

## 6. 2026-07-06 NAS v1.1 试部署记录

本轮已在 NAS 上完成一次当前代码试部署。

部署前状态：

1. 本机存在 WSL2 `Ubuntu`，但缺少 `expect`；历史 `auto_deploy_expect.tcl` 不能直接执行。
2. SSH 免密直连 `polar@192.168.0.115` 不可用，但 SSH config 别名 `nas` 可通过专用 key 登录。
3. `http://192.168.0.115:38088/api/healthz` 返回旧版健康检查：`version=0.2.0`，且没有 `schema_version`。
4. 首页 `Last-Modified` 为 2026-06-11，确认 NAS 上运行的是旧部署。

部署动作：

1. 部署包只包含构建必需内容：`backend/`、`frontend/`、`deploy/`、`docker-compose.yml`、`.env.example`、`README.md`、`AGENTS.md`。
2. 排除了 `.git`、本地数据、历史 AI 工作区和带自动登录信息的脚本，避免把非运行资产同步到 NAS。
3. 部署前备份数据库到 `/volume1/docker/ledger-two/backups/predeploy/ledger-predeploy-20260706-183450.db`。
4. 修复旧库中同一账本下重复账户名称：将未被交易引用的重复账户从 `日常账户` 重命名为 `日常账户（重复保留）`。
5. 修复前额外备份数据库到 `/volume1/docker/ledger-two/backups/predeploy/ledger-repair-before-account-dedupe-20260706-184429.db`。

部署结果：

1. Docker build 在 NAS 上完成，容器 `ledger-two` 已重新创建并启动。
2. 迁移已执行到 schema version 12。
3. `http://192.168.0.115:38088/api/healthz` 返回 `db=ok`、`status=ok`、`schema_version=12`。
4. 容器状态为 `healthy`，端口映射为 `0.0.0.0:38088->8080/tcp`。
5. 首页 `Last-Modified` 更新为 2026-07-06，说明当前前端静态资源已更新。

剩余注意事项：

1. 本次只验证健康检查和首页可访问，尚未完成浏览器内登录、记账、附件、备份按钮等人工验收。
2. v1.1 冻结前仍需补 375px/390px/430px 移动端截图或手工验收记录。
3. 后续如要公网访问，仍需先完成 HTTPS 反向代理与 `COOKIE_SECURE=true` 配置。

## 7. 2026-07-07 v1.1 健康检查版本口径收口

本轮发现 `/api/healthz` 已能返回 `schema_version`，但服务版本仍硬编码为历史 `0.2.0`。该问题会影响 NAS 试部署后的版本判断，因此已将后端健康检查版本口径调整为 `1.1.0-rc`，并同步 OpenAPI 的 API 版本与 health 响应结构。

验证结果：

1. `CGO_ENABLED=0 go test ./internal/http/router -run TestHealthz -count=1` 通过。
2. `go test ./internal/http/router -count=1` 在 Windows 原生环境失败于 `github.com/mattn/go-sqlite3` cgo 对象解析，未进入本次 healthz 断言。
3. `CGO_ENABLED=0 go test ./internal/http/router -count=1` 可编译运行，但同包 RBAC 验收测试需要真实 SQLite migration，按预期失败于 sqlite3 cgo stub。

部署注意：

1. 下一次 NAS 部署后，`GET http://192.168.0.115:38088/api/healthz` 应返回 `version=1.1.0-rc`、`db=ok`、`schema_version=12`。
2. 若该接口仍返回 `version=0.2.0`，应优先判断 NAS 是否仍运行旧镜像或部署包未更新。
