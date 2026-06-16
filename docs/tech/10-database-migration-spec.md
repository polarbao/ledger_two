# 技术：数据库迁移与版本发布规范

## 1. 目标

保证 LedgerTwo 在 NAS 长期部署、迭代升级中，数据库能够安全、平滑地更新 Schema，避免因升级失败导致历史数据损毁或丢失。

## 2. 数据库迁移规范 (Migration)

### 2.1 工具约定
- 本项目采用 `pressly/goose` 进行 SQLite 的 Schema 管理。
- 所有的迁移脚本统一存放在 `backend/migrations` 目录下。

### 2.2 脚本命名与格式
- **命名规范**：`[顺序号]_[变更描述].sql`，例如 `008_add_budget_table.sql`。
- **文件结构**：必须同时包含向前的升级脚本 (`-- +goose Up`) 和向后的降级脚本 (`-- +goose Down`)。

```sql
-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE budgets (
    id TEXT PRIMARY KEY,
    amount_cents INTEGER NOT NULL
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE budgets;
```

### 2.3 SQLite 迁移特性与限制
- **不支持复杂的 ALTER TABLE**：SQLite 不支持 `DROP COLUMN` 和直接的 `ALTER COLUMN`。
- **安全的表结构变更流程**：如果必须删除或修改字段：
  1. 创建拥有新结构的临时表。
  2. 将老表的数据复制到临时表。
  3. 删除老表。
  4. 将临时表重命名为老表的名称。

### 2.4 PR 合并红线
- 凡涉及 `migrations/` 目录的 Pull Request，在 Review 阶段必须经过干跑测试 (Dry-run)，证明向后兼容性。

## 3. 发布与升级检查机制

### 3.1 自动版本检查与数据拦截
- 在应用启动 (`db.Init`) 时，系统会自动调用 `goose.Up` 进行迁移。
- **安全防线**：为了防止执行迁移崩溃导致无法挽回的后果，引擎会在检测到有新版本 migration 将要执行前，在 `/app/data/` (或配置的数据库路径) 生成一个包含时间戳的 `.bak` 备份文件（如 `ledger.db.pre_migrate_xxx.bak`）。若迁移失败产生破坏，用户可以直接还原。

### 3.2 版本暴露探测
- 服务端的 `/api/healthz` 接口不仅返回存活状态，还会返回当前 SQLite 的 `schema_version` (通过 `goose.GetDBVersion` 暴露)。
- 客户端与监控脚本可以定期读取此接口，确认服务当前版本与数据库结构是否对齐。

## 4. 给用户的升级前备份说明

为了最大限度保障您的财产账务数据安全，在执行 Docker 容器版本更新或二进制文件替换之前，请严格遵守以下步骤：

1. **手动备份数据文件**：进入 NAS 文件管理，将挂载的 `data/ledger.db` 直接复制一份作为离线冷备份。
2. **下载云端备份**：在系统的 Web 界面进入 “系统设置” -> “数据安全”，点击“立即生成全量备份”并下载 `.csv` 或 `.json` 至本地电脑。
3. **安全更新**：执行 `docker-compose pull` 然后 `docker-compose up -d` 启动新版容器。
4. **验证状态**：进入浏览器打开系统，检查首页与历史流水是否显示正常。如果发生 500 等异常，说明迁移失败或遭遇 Bug。
5. **灾难恢复**：如果在升级后遭遇数据乱码或直接无法启动，不要进行二次重启，立即将刚才备份的 `ledger.db` 覆盖回 `data` 目录并退回之前的镜像版本。
