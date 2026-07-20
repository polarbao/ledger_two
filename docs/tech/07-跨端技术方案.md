# 技术：跨端技术方案

## 1. 目标

跨端技术方案用于保证 LedgerTwo 的后端 API 能被 Web、PWA、移动端 App、桌面端等复用。

## 2. 分阶段路线

| 阶段 | 技术方案 | 说明 |
|---|---|---|
| v0.1-v0.3 | 响应式 Web | 当前主线，React 实现 |
| v0.6 | PWA | 添加到桌面、缓存静态资源、移动端优化 |
| v0.7+ | IndexedDB | 缓存最近账单、分类、标签 |
| v0.8+ | 离线草稿 | 离线创建草稿，联网后提交 |
| v1.x | React Native / Expo | 真正移动端 App，复用 API |
| v1.x | Tauri | 桌面端可选，不是优先方向 |

## 3. API 约束

所有客户端必须使用同一套 REST JSON API。

约束：

- 金额统一使用 int64 cents。
- 时间统一 ISO8601。
- DTO 不暴露数据库表结构。
- 错误码稳定。
- 分页、筛选、排序参数稳定。
- 上传接口使用 multipart/form-data。

## 4. 认证策略

### 4.1 当前 Web

使用 HttpOnly Cookie Session。

### 4.2 后续移动端

预留 Token 认证：

- Access Token。
- Refresh Token。
- Token 绑定设备。
- 可撤销设备登录。

短期不必实现，但 API 和 Auth 模块不要写死只适配浏览器 Cookie。

## 5. PWA 策略

PWA 初期只做：

- manifest。
- icon。
- service worker 缓存静态资源。
- 离线提示。

暂不做：

- 离线写入。
- 后台同步。
- Push 通知。

## 6. 本地缓存策略

后续 IndexedDB 缓存：

- 当前用户。
- 分类。
- 标签。
- 账户。
- 最近流水。
- 最近 Dashboard 快照。

缓存只能提升体验，最终数据以后端为准。

## 7. 冲突处理

后续离线写入时，账单表需要增加：

- version。
- updated_at。
- client_id。
- sync_status。

冲突策略：

- 如果本地草稿未提交，直接提交。
- 如果同一账单服务端已变更，提示用户选择保留本地或服务端。
- 金额和分摊方式冲突必须人工确认。

## 8. 前端工程预留

建议前端保持：

```text
src/api        API 层
src/types      DTO 类型
src/pages      页面层
src/components 组件层
src/stores     UI 状态
src/utils      金额、时间等工具
```

React Native 后续可以复用：

- API client。
- types。
- utils。
- 部分业务 hooks。

不直接复用 Web DOM 组件。

## 9. 验收标准

- Web API 不依赖浏览器专有能力。
- 前端金额、时间转换逻辑集中在 utils。
- PWA 在移动端可以添加到桌面。
- 离线时不会误导用户以为保存成功。
