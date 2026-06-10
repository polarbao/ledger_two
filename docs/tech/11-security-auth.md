# 技术：认证、安全与权限设计

## 1. 文档目标

本文件定义 LedgerTwo 的认证、Session、安全策略、权限判断和敏感数据保护规则。

LedgerTwo 是私有化部署工具，但仍然存储真实账务数据，因此不能因为部署在 NAS 或局域网就忽略安全设计。

## 2. 当前阶段安全边界

v0.2-v0.4 阶段：

- 单账本。
- 双用户。
- 不开放公开注册。
- 使用账号密码登录。
- 使用 HttpOnly Cookie Session。
- 推荐局域网或 Tailscale 访问。

不做：

- 第三方 OAuth。
- 多设备 Token 管理。
- 短信验证码。
- 邮件找回密码。
- 企业级 RBAC。

## 3. 初始化安全

系统首次启动时进入初始化流程。

初始化要求：

1. 只能在 `initialized=false` 时执行。
2. 初始化必须创建两个用户。
3. 初始化必须创建默认分类、标签、账户。
4. 密码必须 hash 后保存。
5. 初始化成功后写入 `app_settings.initialized=true`。
6. 重复初始化返回 `APP_ALREADY_INITIALIZED`。

## 4. 密码存储

推荐：

- bcrypt，成本参数建议 12。
- 或 Argon2id，后续可选。

禁止：

- 明文保存密码。
- 使用 MD5/SHA1/SHA256 直接 hash 密码。
- 在日志中打印密码或 hash。

用户表建议字段：

```text
id
username
display_name
password_hash
status
created_at
updated_at
last_login_at
```

## 5. Session 设计

Demo 和 Web 阶段建议使用数据库 Session。

Session 表建议字段：

```text
id
user_id
session_token_hash
user_agent
ip_address
expires_at
created_at
revoked_at
```

注意：数据库中保存 session token hash，不保存明文 token。

Cookie 设置：

```text
HttpOnly=true
SameSite=Lax
Secure=true，HTTPS 下必须开启
Path=/
MaxAge=按配置
```

本地开发可以暂时 `Secure=false`，生产环境必须开启 HTTPS 后设置 Secure。

## 6. 认证中间件

后端应提供认证中间件：

```text
RequireAuth
OptionalAuth
RequireAdmin，后续可选
```

所有业务 API 默认需要登录。

公开 API 仅包括：

- GET /api/healthz
- GET /api/init/status
- POST /api/init/setup，初始化完成后不可再用
- POST /api/auth/login

## 7. 权限模型

当前双人账本阶段：

- 创建人可编辑自己的普通账单。
- private 账单仅创建人可见。
- partner_readable 账单对方可见但不可编辑。
- shared 账单双方可见。
- settlement 记录双方可见。

后续多账本阶段：

- 每次查询必须校验 ledger membership。
- owner 可管理账本成员和数据导出。
- editor 可创建和编辑自己有权限的账单。
- viewer 只读。

## 8. 可见性规则

| visibility | 创建人 | 对方 | 说明 |
|---|---|---|---|
| private | 可见可编辑 | 不可见 | 个人账单 |
| partner_readable | 可见可编辑 | 可见不可编辑 | 给对方查看 |
| shared | 可见 | 可见 | 共同账单 |

API 查询时必须在 SQL 或 service 层过滤，不允许只在前端隐藏。

对于不可见资源，建议返回 `NOT_FOUND`，避免暴露资源存在性。

## 9. 高风险操作

以下操作必须写入 audit log：

- 修改金额。
- 修改分摊方式。
- 删除账单。
- 创建结算。
- 删除结算，后续可选。
- 导出数据。
- 触发备份。
- 恢复备份，后续。

以下操作必须前端二次确认：

- 删除账单。
- 生成结算。
- 手动备份。
- 恢复备份。
- 批量导入。
- 批量删除。

## 10. CSRF 策略

如果使用 Cookie Session，需考虑 CSRF。

短期私有部署可采用：

- SameSite=Lax。
- API 仅接受 JSON Content-Type。
- 非 GET 请求校验 Origin/Referer。

后续增强：

- CSRF Token。
- 双提交 Cookie。

## 11. CORS 策略

开发环境：

```text
http://localhost:5173
```

生产环境：

- 只允许实际访问域名。
- 不允许 `*` 搭配 credentials。
- 不允许任意来源携带 Cookie。

## 12. 敏感信息保护

不得提交到 Git：

- 真实 `.env`。
- SQLite 真实数据库。
- 备份文件。
- 上传附件。
- SESSION_SECRET。

`.gitignore` 必须覆盖：

```text
.env
data/
backups/
uploads/
*.db
*.sqlite
```

## 13. 日志安全

日志中可以记录：

- request_id。
- user_id。
- API path。
- error_code。
- duration。

日志中禁止记录：

- 密码。
- Session 明文 token。
- SESSION_SECRET。
- 完整 Cookie。
- 导出文件完整内容。

## 14. 限流与暴力破解防护

v0.2 可先做简单登录失败限制：

- 同一 username 连续失败 5 次后短暂锁定。
- 同一 IP 连续失败过多时返回 RATE_LIMITED。

私有部署下不是 P0，但建议 v1.0 前补齐。

## 15. 跨端认证预留

未来移动端可以增加 Token 认证：

- access token 短期。
- refresh token 长期。
- device_id 绑定。
- 用户可撤销设备登录。

当前不实现，但 Auth service 设计不要与浏览器 Cookie 强耦合。

## 16. 验收标准

- 密码不明文保存。
- Session Cookie 为 HttpOnly。
- 未登录访问业务 API 返回 UNAUTHORIZED。
- private 账单对方无法通过 API 获取。
- partner_readable 账单对方不能编辑。
- 删除账单、结算、导出、备份写入 audit log。
- 生产环境不允许任意 CORS 来源携带 Cookie。
