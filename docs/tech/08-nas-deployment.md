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
- 不提交真实 SESSION_SECRET。
- 设置 restart unless-stopped。

## 4. 环境变量

```text
APP_ENV=production
HTTP_ADDR=:8080
DB_PATH=/app/data/ledger.db
BACKUP_DIR=/app/backups
UPLOAD_DIR=/app/uploads
SESSION_SECRET=please-change-me
TZ=Asia/Shanghai
```

## 5. 访问方式

### 5.1 局域网访问

```text
http://NAS_IP:8088
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

## 8. 升级流程

1. 手动备份当前数据库。
2. 拉取新镜像或上传新代码。
3. 执行数据库迁移。
4. 重启容器。
5. 检查 /api/healthz。
6. 登录 Web 验证核心功能。

## 9. 风险提示

如果 NAS 存储池异常或 degraded，不建议把账务数据作为唯一副本。先修复阵列或配置第二备份。

## 10. 验收标准

- NAS 上 Docker Compose 可启动。
- 重启容器后数据不丢。
- 备份目录有文件生成。
- 局域网和 Tailscale 可访问。
- 错误日志可查看。
