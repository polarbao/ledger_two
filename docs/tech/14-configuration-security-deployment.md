# 技术：配置安全与部署一致性

状态：供审核  
目标：统一环境变量、生产安全校验、Cookie/HTTPS 策略和部署诊断，避免 v1.1 之前继续积累安全和运维风险。

## 1. 当前问题

当前仓库已具备 Docker Compose、Dockerfile、健康检查和 NAS 部署文档。但配置层需要进一步统一：

1. Docker Compose 与后端 config 读取的变量名需要完全一致。
2. 生产环境不能使用默认开发密钥。
3. Cookie 的 Secure 策略必须与 HTTP/HTTPS 访问方式匹配。
4. `.env.example`、部署文档、config.go、README 必须一致。

## 2. 推荐环境变量

建议统一为：

```text
APP_ENV=production
DEPLOYMENT_CHANNEL=production
IMPORT_XLSX_ENABLED=false
HTTP_ADDR=:8080
APP_PORT=38088
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

说明：

| 变量 | 必填 | 默认值 | 说明 |
|---|---|---|---|
| APP_ENV | 是 | development | production 时启用严格校验 |
| DEPLOYMENT_CHANNEL | 是 | 跟随 APP_ENV | 数据用途，只允许 development/staging/production |
| IMPORT_XLSX_ENABLED | 否 | development/staging=true，production=false | 微信 XLSX preview 运行门禁，只接受 true/false；不改变支付宝 CSV-only 边界 |
| HTTP_ADDR | 是 | :8080 | 后端监听地址 |
| APP_PORT | Docker 部署需要 | 38088 | 宿主机暴露端口，仅供 Docker Compose 端口映射使用 |
| APP_BASE_URL | 建议 | 空 | 用于生成链接、诊断和未来通知 |
| DB_DSN | 是 | data/ledger.db | SQLite 文件路径 |
| BACKUP_DIR | 是 | data/backups | 备份目录 |
| UPLOAD_DIR | 是 | data/uploads | 上传目录 |
| LOG_DIR | 建议 | data/logs | 日志目录 |
| JWT_SECRET | production 必填 | 无 | JWT 签名密钥，生产缺失必须启动失败 |
| COOKIE_SECURE | production 建议明确 | false | HTTPS 反代时为 true，HTTP 局域网时为 false |
| COOKIE_SAMESITE | 是 | Lax | Cookie SameSite 策略 |
| TZ | 是 | Asia/Shanghai | 容器时区 |

## 3. 生产启动校验

后端启动时必须执行配置校验：

```text
APP_ENV=production 时：
- JWT_SECRET 不得为空。
- JWT_SECRET 不得等于 dev 默认值。
- JWT_SECRET 长度建议 >= 32 字节，推荐 >= 64 字符。
- DB_DSN 必须存在父目录且可写。
- BACKUP_DIR 必须可创建且可写。
- UPLOAD_DIR 必须可创建且可写。
- COOKIE_SECURE 必须显式配置。
```

如果校验失败，服务应拒绝启动，并输出明确错误日志。

## 4. Cookie 策略

### 4.1 局域网 HTTP 模式

适合：仅在 NAS 局域网或 Tailscale 内访问。

```text
APP_ENV=production
APP_BASE_URL=http://NAS_IP:38088
COOKIE_SECURE=false
COOKIE_SAMESITE=Lax
```

风险说明：HTTP 不加密，建议只在可信局域网/Tailscale 使用。

### 4.2 HTTPS 反向代理模式

适合：使用域名和 HTTPS 访问。

```text
APP_ENV=production
APP_BASE_URL=https://ledger.example.com
COOKIE_SECURE=true
COOKIE_SAMESITE=Lax
```

要求：反向代理必须正确转发 `X-Forwarded-Proto`，后续可扩展可信代理配置。

## 5. 健康检查

`GET /api/healthz` 建议返回：

```json
{
  "status": "ok",
  "version": "1.2.0-rc",
  "schema_version": 18,
  "deployment_channel": "production",
  "db": "ok",
  "env": "production",
  "storage": {
    "data": "ok",
    "backups": "ok",
    "uploads": "ok"
  }
}
```

敏感信息不得返回：

- JWT_SECRET。
- Cookie 值。
- 真实文件绝对路径，可只返回状态。
- 用户数据。

## 6. Docker Compose 建议

```yaml
services:
  ledger-two:
    image: ledger-two:latest
    restart: unless-stopped
    ports:
      - "38088:8080"
    env_file:
      - .env
    volumes:
      - ./data:/app/data
      - ./backups:/app/backups
      - ./uploads:/app/uploads
      - ./logs:/app/logs
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

## 7. 验收标准

1. Docker Compose、`.env.example`、config.go、部署文档变量完全一致。
2. 生产环境使用默认密钥时服务启动失败。
3. HTTP 模式可登录，HTTPS 模式可登录。
4. `/api/healthz` 能反映数据库和目录状态。
5. README 中说明如何生成随机密钥。
6. CI 或测试能覆盖 config validation。
