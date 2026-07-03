# 06 群晖 NAS 部署文档：LedgerTwo v0.2

## 1. 部署目标

将 LedgerTwo 作为 Docker 容器部署在群晖 NAS 上，使用 SQLite 保存数据，通过局域网或 Tailscale 访问。

推荐原则：

1. 不直接暴露公网端口。
2. 数据目录挂载到 NAS volume。
3. 每日自动备份。
4. 备份至少同步到 NAS 外第二位置。

## 2. 前置条件

群晖需要：

1. DSM 7.x。
2. Container Manager。
3. 可用共享文件夹，例如 `/volume1/docker`。
4. 可选：SSH。
5. 可选：Tailscale。

重要：如果 NAS 存储池处于 degraded 状态，不建议把账务数据作为唯一副本放在 NAS 上。应先修复阵列，或配置外部备份。

## 3. 目录规划

```text
/volume1/docker/ledger-two/
  docker-compose.yml
  .env
  data/
    ledger.db
  backups/
    daily/
    weekly/
    monthly/
  uploads/
  logs/
```

创建目录：

```bash
mkdir -p /volume1/docker/ledger-two/data
mkdir -p /volume1/docker/ledger-two/backups/daily
mkdir -p /volume1/docker/ledger-two/backups/weekly
mkdir -p /volume1/docker/ledger-two/backups/monthly
mkdir -p /volume1/docker/ledger-two/uploads
mkdir -p /volume1/docker/ledger-two/logs
```

## 4. docker-compose.yml

```yaml
services:
  ledger-two:
    image: ledger-two:latest
    container_name: ledger-two
    restart: unless-stopped
    ports:
      - "8088:8080"
    environment:
      APP_ENV: production
      HTTP_ADDR: ":8080"
      DB_PATH: /app/data/ledger.db
      BACKUP_DIR: /app/backups
      UPLOAD_DIR: /app/uploads
      SESSION_SECRET: "change-this-to-a-long-random-string"
      TZ: Asia/Shanghai
    volumes:
      - ./data:/app/data
      - ./backups:/app/backups
      - ./uploads:/app/uploads
      - ./logs:/app/logs
```

## 5. Dockerfile 示例

```dockerfile
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.22-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /app/frontend/dist ./web/dist
RUN CGO_ENABLED=1 go build -o /app/ledger-two ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata sqlite
WORKDIR /app
COPY --from=backend-builder /app/ledger-two /app/ledger-two
EXPOSE 8080
CMD ["/app/ledger-two"]
```

## 6. 启动服务

```bash
cd /volume1/docker/ledger-two
docker compose up -d
```

查看日志：

```bash
docker compose logs -f
```

停止服务：

```bash
docker compose down
```

升级服务：

```bash
docker compose pull
docker compose up -d
```

本地构建：

```bash
docker compose up -d --build
```

## 7. 访问方式

局域网：

```text
http://NAS-IP:8088
```

例如：

```text
http://192.168.0.115:8088
```

Tailscale：

```text
http://NAS内网IP:8088
```

如果你已经配置 Tailscale 子网路由，异地设备可以继续通过 NAS 局域网 IP 访问。

## 8. 群晖反向代理，可选

DSM 路径：

```text
控制面板 -> 登录门户 -> 高级 -> 反向代理
```

示例：

```text
来源：
协议：https
主机名：ledger.example.com
端口：443

目的地：
协议：http
主机名：127.0.0.1
端口：8088
```

如果只通过 Tailscale 访问，可以暂不配置公网反向代理。

## 9. 防火墙建议

推荐只允许：

1. 局域网网段，例如 `192.168.0.0/24`。
2. Tailscale 网段，例如 `100.64.0.0/10`。
3. Docker 内部网段。

不建议对公网开放 8088。

## 10. 备份方案

### 10.1 应用内备份

应用每天凌晨执行：

```sql
VACUUM INTO '/app/backups/daily/ledger-two-YYYY-MM-DD.db';
```

并导出 JSON：

```text
/app/backups/daily/ledger-two-YYYY-MM-DD.json
```

### 10.2 保留策略

| 类型 | 保留 |
|---|---|
| 每日 | 30 天 |
| 每周 | 12 周 |
| 每月 | 12 个月 |

### 10.3 第二备份位置

建议同步到：

- 另一台电脑
- 移动硬盘
- iCloud Drive
- OneDrive
- 另一台 NAS

## 11. 恢复流程

1. 停止容器。
2. 备份当前 `data/ledger.db`。
3. 将目标备份文件复制为 `data/ledger.db`。
4. 启动容器。
5. 登录检查账单和统计。

命令：

```bash
cd /volume1/docker/ledger-two
docker compose down
cp data/ledger.db data/ledger.db.before-restore
cp backups/daily/ledger-two-2025-04-30.db data/ledger.db
docker compose up -d
```

## 12. 健康检查

后端提供：

```http
GET /healthz
```

返回：

```json
{
  "status": "ok",
  "db": "ok",
  "version": "0.2.0"
}
```

## 13. 常见问题

### 13.1 页面打不开

检查：

1. 容器是否运行。
2. 端口是否映射。
3. 群晖防火墙是否放行。
4. Tailscale 是否连通。

### 13.2 数据不见了

检查：

1. volume 是否正确映射。
2. 是否误删 `data/ledger.db`。
3. 是否启动了不同目录下的 compose。

### 13.3 备份失败

检查：

1. backups 目录权限。
2. 容器日志。
3. NAS 磁盘空间。
