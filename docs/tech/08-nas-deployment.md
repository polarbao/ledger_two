# 技术：NAS 部署方案

## 1. 部署目标

LedgerTwo 目标部署在本地群晖 NAS，通过 Docker Compose 启动，优先在局域网和 Tailscale 内访问，不建议初期直接暴露公网。

## 2. 目录规划

建议在 NAS 上创建：

```text
/volume1/docker/ledger-two/
  docker-compose.yml
  .env
  data/
  backups/
  uploads/
  logs/
```

## 3. Docker Compose 原则

- SQLite 数据库挂载到 `/app/data`。
- 备份目录挂载到 `/app/backups`。
- 附件目录挂载到 `/app/uploads`。
- 不把数据库打进镜像。
- 不提交真实 JWT_SECRET。
- 设置 restart unless-stopped。

## 4. 环境变量

```text
APP_ENV=production
HTTP_ADDR=:8080
APP_BASE_URL=http://NAS_IP:38088
DB_DSN=/app/data/ledger.db
BACKUP_DIR=/app/backups
UPLOAD_DIR=/app/uploads
LOG_DIR=/app/logs
JWT_SECRET=<64 chars random secret>
COOKIE_SECURE=false
COOKIE_SAMESITE=Lax
TZ=Asia/Shanghai
```

生成 `JWT_SECRET` 时应使用随机字符串，不要使用示例值。局域网 HTTP 或 Tailscale 访问时 `COOKIE_SECURE=false`；HTTPS 反向代理访问时改为 `COOKIE_SECURE=true`，并确保代理转发 `X-Forwarded-Proto`。

## 5. 访问方式

### 5.1 局域网访问

```text
http://NAS_IP:38088
```

### 5.2 Tailscale 访问

推荐通过 Tailscale 访问 NAS，避免公网暴露。

### 5.3 反向代理

如需域名访问，可以使用群晖反向代理或 Nginx Proxy Manager。公网访问必须启用 HTTPS。

## 6. 备份要求

必须备份：

- data/ledger.db。
- backups/。
- uploads/。
- .env，注意安全保存。

建议至少保留 NAS 外第二备份，例如另一台电脑、移动硬盘或云盘。

## 7. 健康检查

后端提供：

```text
GET /api/healthz
```

健康检查应返回服务状态、数据库连接状态和版本号。

## 8. 升级与回滚流程

### 8.1 升级
1. **先在页面点击「手动备份」**，确保下载并妥善保管一份最新的 `backup.db`。
2. 拉取新镜像：`docker compose pull`。
3. 停止并移除旧容器：`docker compose down`。
4. 启动新容器：`docker compose up -d`。
5. （可选）如果使用代码构建：`docker compose up -d --build`。
6. 后端启动时会自动执行数据库 Migration，检查 `logs` 或直接访问 `/api/healthz` 确保状态正常。

### 8.2 回滚 (Rollback)
如果新版本出现致命错误，需要回滚：
1. 停止当前服务：`docker compose down`。
2. 将 `data/ledger.db` 重命名备份（如 `ledger.db.err`）。
3. 将升级前备份的 `backup.db` 放入 `data/` 目录并重命名为 `ledger.db`。
4. 修改 `docker-compose.yml` 中的镜像标签，指向上一稳定版本。
5. 启动服务：`docker compose up -d`。

## 9. 备份与恢复

### 9.1 数据备份
系统的所有持久化数据都在挂载的数据卷内。强烈建议配置群晖的 **Hyper Backup** 对以下目录进行定期外接硬盘/网盘备份：
- `data/`（包含核心账本 SQLite 数据）
- `backups/`（由系统内部手动或自动生成的备份文件）
- `uploads/`（用户上传的图片/发票等附件）
- `.env` 及 `docker-compose.yml`

### 9.2 灾难恢复
如果发生硬盘损坏等意外导致重装：
1. 在新环境中重新准备目录结构并写入原先的 `.env`。
2. 将最新的一份可用数据库文件恢复为 `data/ledger.db`。
3. 将照片和历史备份拷贝回 `uploads/` 和 `backups/`。
4. 执行 `docker compose up -d`。

## 10. 日志查看

排查问题时，经常需要查看运行日志：
- 查看实时日志：`docker compose logs -f`
- 查看末尾 100 行日志：`docker compose logs --tail=100`
- 如果后端日志输出了文件卷（如映射了 `logs/` 目录），也可以直接打开物理文件进行搜索。

## 11. 常见问题排查 (FAQ)

**Q1：网页加载出来但是显示网络错误？**
- **排查**：请确认浏览器的 IP 和端口是否能通畅连接后端。检查 Docker 暴露的端口是否被群晖的其他服务（如 Web Station 或其他容器）占用。检查 `docker-compose logs` 是否提示了 panic。

**Q2：提示 "database is locked"？**
- **排查**：SQLite 并发请求过高，或者处于不稳定的机械硬盘阵列上。尽量将 Docker 的 `data/` 挂载路径放置于群晖的 SSD 存储空间。如果在导出或全量备份期间出现此报错，这是因系统在执行写保护，请稍后重试。

**Q3：NAS 提示写入失败或附件无法上传？**
- **排查**：检查映射目录权限。建议在群晖 File Station 中右键 `docker/ledger-two` 目录，检查并赋予 Docker 运行用户（如 PUID/PGID）足够的读写权限。

## 12. 风险提示

如果 NAS 存储池异常或 degraded，不建议把账务数据作为唯一副本。先修复阵列或配置第二备份。

## 13. 验收标准

- 新用户能够根据此文档配置目录并一次性启动服务。
- 老用户明确升级前必须先备份，以及失败后的降级恢复手段。
- 在发生错误时，能够使用命令追踪 Docker 日志。
