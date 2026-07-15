# Task50P.6 开发准入评审

状态：有条件通过，仅允许在独立本地 development 环境启动 Task50.1<br>
评审日期：2026-07-15<br>
适用范围：v1.3 Task50.1-Task50.6 开发准入，不代表 v1.2 NAS 发布门禁已关闭

## 1. 结论

Task50P.6 已完成，可以从下一原子任务开始 Task50.1，但必须同时遵守以下边界：

1. Task50 只在独立本地 `development` 数据目录开发和迁移，不连接当前 WSL2 staging、NAS staging 或 NAS production 数据库。
2. 当前本机 WSL2 `http://localhost:38088` 继续作为 v1.2.0-rc/schema 19 验收实例，不执行 migration 020/021。
3. NAS schema 19 staging 与 production 发布门禁仍由 v1.2 RC05/Task49X 发布计划管理，未因本评审自动关闭。
4. UI-FL-01 至 UI-FL-10 已全部完成，Fresh Light 为新会话默认体验，Dark Glass 保留为显式回退主题。
5. Task50.1 只实现 migration、model、repository 与升级预检，不提前实现生命周期 API、前端页面或 NAS 部署。

这是一项“隔离开发放行”，不是“候选环境迁移放行”或“生产发布放行”。

## 2. 准入条件核对

| P.6 条件 | 证据 | 结论 |
|---|---|---|
| NAS schema 19 门禁关闭，或书面决定仅独立开发 | NAS 门禁仍开放；本文明确选择独立本地 development 路径 | 通过，有条件 |
| UI-FL-10 完成 | `docs/project_analysis/ui-fl-10-global-2026-07-15/README.md`；实现提交 `c8d63fe` | 通过 |
| PRD/Tech/UI/OpenAPI/Migration/验收冻结 | PRD 31、Tech 25、UI 16、OpenAPI v1.3 草案、Fixture 32 与 Task50 Figma handoff 均已冻结 | 通过 |
| development/staging/production 物理隔离 | `docs/tech/23-v1.2-deployment-environment-isolation.md` 已冻结实例、目录和 channel 规则 | 通过，开发时持续检查 |
| 工作树无未归属真实数据和密钥 | 评审前工作树干净；Git 未追踪 `.db/.sqlite/.xlsx/.xls/.env`、真实账单、上传目录或 Figma 原始包 | 通过 |

## 3. 冻结事实源

Task50 实现发生冲突时按以下顺序判断：

1. `docs/prd/31-prd-v1.3-multi-ledger.md`
2. `docs/tech/25-v1.3-multi-ledger-implementation-contract.md`
3. `docs/api/openapi-v1.3-ledger-draft.yaml`
4. `docs/prd/32-v1.3-task50-acceptance-fixtures.md`
5. `docs/ui/16-v1.3-multi-ledger-flows.md`
6. `docs/ui/figma/task50-v1.3-multi-ledger/`
7. `docs/codex_tasks/15-v1.3-task50-detailed-implementation-plan.md`

早期 Demo 文档、生成预览和未验证线上 Figma 节点不得覆盖上述事实源。实现发现契约冲突时先回到文档评审，不能在代码中自行选择新业务规则。

## 4. 独立开发环境决策

### 4.1 运行边界

| 项目 | Task50 development | 当前 WSL2 staging |
|---|---|---|
| 浏览器入口 | `http://127.0.0.1:5173` | `http://localhost:38088` |
| 后端 | Windows/WSL 原生 Go `127.0.0.1:8080` | WSL2 Docker |
| `APP_ENV` | `development` | `production` |
| `DEPLOYMENT_CHANNEL` | `development` | `staging` |
| schema | Task50.1 后最多 21 | 固定 19 |
| 数据库 | `backend/data/development/task50/ledger.db` | 仓库根目录受控 `data/` 挂载 |
| 上传/备份/日志 | `backend/data/development/task50/*` | staging 独立挂载目录 |

开发浏览器统一使用 `127.0.0.1`，而 staging 使用 `localhost`。当前认证 Cookie 名为 `token`，Cookie 不按端口隔离；区分主机名可以避免两个环境在同一浏览器中误共享会话。

### 4.2 推荐启动变量

在 `backend` 目录启动 Task50 development：

```powershell
$env:APP_ENV='development'
$env:DEPLOYMENT_CHANNEL='development'
$env:DB_DSN='data/development/task50/ledger.db'
$env:BACKUP_DIR='data/development/task50/backups'
$env:UPLOAD_DIR='data/development/task50/uploads'
$env:LOG_DIR='data/development/task50/logs'
go run ./cmd/server
```

前端在 `frontend` 目录启动，并从 `http://127.0.0.1:5173` 访问：

```powershell
corepack pnpm dev --host 127.0.0.1
```

禁止复制或软链接以下数据到 development：

- WSL staging 的 schema 19 主库。
- NAS staging/production 数据库和上传目录。
- `E:\__Project_Data` 或仓库 `data/` 中的真实支付账单。
- 生产 `.env`、JWT 密钥、Cookie 密钥和备份口令。

需要升级样本时，只能使用匿名 Fixture 或经确认生成的脱敏数据库副本，并记录来源、校验和和销毁方式。

## 5. Migration 与回滚门禁

1. migration 020/021 只能在 Task50 development 新库、自动化临时库和明确命名的脱敏副本执行。
2. Task50.1 必须覆盖 `001 -> 021`、`019 -> 021`、异常 Owner 数据预检、索引/trigger 和数据守恒。
3. production 不使用 `goose down`。发布后回滚依赖升级前一致性备份、应用/数据库成对恢复和向前修复 migration。
4. development migration 失败时，只允许删除或重建 `backend/data/development/task50/` 下的可丢弃数据；不得清理 staging 或 NAS 路径。
5. Task50.6 之前不得把 schema 21 镜像连接到 staging；Task50.6 也只准备独立 v1.3 staging 候选，不自动部署 NAS。

## 6. 每个 Task 的持续准入检查

每个 Task50 原子任务开始前都必须确认：

1. 上一 Task 已独立提交且工作树中的其他修改已归属。
2. 当前任务只连接 `development` 数据目录，health 返回 `development` channel。
3. 不读取、提交或覆盖真实数据库、账单、密钥、上传和原始 Figma 文件。
4. 任务绑定 `T50-*` 验收 ID、预计文件、测试和回滚方式。
5. API、migration、权限或产品规则变化先更新并重新评审事实源。
6. Task50.5 复用已关闭 UI-FL 组件，不反向重写 v1.2 业务页面。

## 7. 未关闭事项

以下事项继续存在，但不阻塞独立开发：

1. NAS schema 19 staging 与 production 发布窗口尚未完成。
2. v1.3 schema 21 staging、升级演练和 NAS 发布证据归 Task50.6，当前不得执行。
3. 线上 Figma 同步状态仍为 `not_verified`；本地 handoff 足以指导代码，但不能宣称已同步到指定账号。
4. 前端单包约 676 kB 的构建告警保留为 P2 性能专项，不混入 Task50.1。

## 8. 放行结果

Task50P.6 以“仅独立 development”条件关闭。下一任务是 `Task50.1`，执行入口为 `docs/codex_tasks/15-v1.3-task50-detailed-implementation-plan.md`。在 Task50.1 提交并通过 migration/model/repository 门禁前，Task50.2-Task50.6 保持未开始。
