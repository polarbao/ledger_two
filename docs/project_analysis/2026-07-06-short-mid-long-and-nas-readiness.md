# LedgerTwo 短中长期规划与 NAS 部署就绪复盘

日期：2026-07-06

## 1. 结论

短期 Foundation before v1.1 尚未完全完成，不能标记冻结。

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
| Task40 审计与系统诊断中心 | 未完成 | 缺诊断接口/诊断面板/审计字段规范收口 |

因此当前状态应定义为：

```text
Foundation before v1.1: 90%+ 完成，但未冻结。
可进入 NAS 试部署/内测，不建议作为长期生产冻结版本。
进入 v1.1 业务开发前必须完成 Task40。
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
5. 访问方式为局域网或 Tailscale；公网必须使用 HTTPS 反向代理。
6. 部署前完成一次本地或 CI Docker build 验证。

当前限制：

1. 本机未安装 Docker CLI，本轮无法在本机实际执行 `docker compose config` 或 `docker compose build`。
2. Task40 未完成，部署后缺少完整系统诊断面板。
3. `GET /api/healthz` 目前能返回基础状态和 schema version，但还不能检查备份/上传/日志目录状态。

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
APP_BASE_URL=http://NAS_IP:38088
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

1. 访问 `http://NAS_IP:38088/api/healthz`。
2. 首次打开页面完成初始化。
3. 上传一张附件，确认 `uploads/` 目录出现文件。
4. 创建 private 账单并关联附件，确认另一用户无法访问附件。
5. 点击手动备份，确认 `backups/manual/` 出现 `.db` 文件。
6. 重启容器，确认账本和附件仍存在。

## 5. 后续冻结前必须完成

1. Task40.1：新增诊断接口，检查数据库、schema、备份目录、上传目录、日志目录、运行环境。
2. Task40.2：设置页诊断面板，面向非技术用户展示健康状态。
3. Task40.3：审计日志规范，至少统一备份、恢复、导入、结算、归档、金额修改、删除的审计字段。
4. 更新 OpenAPI 和 API inventory，记录受保护附件访问接口。
5. 在 CI 或部署机运行 Docker build 和 compose config。
