# LedgerTwo v1.1 本地移动端验收记录

日期：2026-07-07

## 1. 验收环境

- 访问地址：`http://localhost:38088`
- 部署方式：本机 WSL2 Docker Compose，容器名 `ledger-two`
- 候选版本：`1.1.0-rc`
- 数据库版本：schema `12`
- 浏览器驱动：本机 Chrome DevTools Protocol
- 移动端宽度：`375px`、`390px`、`430px`

## 2. 覆盖范围

本轮验收覆盖以下页面和关键入口：

- 登录页
- Dashboard
- 记账抽屉打开
- 流水列表
- 移动端高级筛选 Sheet
- 结算中心
- 系统设置
- 分类管理

截图证据位于 `screenshots/`，指标文件为 `metrics.json`。

## 3. 结论

本轮共生成 14 个截图点，最终指标如下：

- 横向溢出：0
- React Router 错误页：0
- 设置页可正常加载备份列表空状态
- 记账抽屉和流水筛选 Sheet 可在 375px 下打开
- 顶部栏移动端不再展示桌面用户信息，避免右侧内容挤压

## 4. 本轮发现并修复的问题

### 4.1 设置页空备份列表崩溃

现象：

- `/api/admin/backups` 在无备份文件时返回 `data: null`
- 设置页读取 `backups.length` 时触发 `Cannot read properties of null`
- 移动端设置页被 React Router 错误页替代

修复：

- 后端 `GetBackups` 空列表返回 `[]`
- 前端 `SettingsPage` 对异常空响应做数组防御

### 4.2 移动端顶部栏右侧挤压

现象：

- `desktop-user-info` 在移动端本应隐藏
- 组件内联 `display: flex` 覆盖媒体查询，导致右侧用户信息挤压屏幕

修复：

- 移除组件内联 display 样式
- 将桌面端 flex 布局回收到 `.desktop-user-info` CSS 类

## 5. 验证命令

```bash
corepack pnpm test
corepack pnpm build
wsl.exe -- bash -lc "cd /mnt/e/__Code/__Prj/ledge_two/ledger_two && sudo docker compose up -d --build"
curl.exe --noproxy "*" -sS -m 30 http://localhost:38088/api/healthz
```

结果：

- Vitest：5 个测试文件通过，16 个测试用例通过
- Build：通过；仅保留 Vite 大 chunk 性能提示
- Docker build：通过，容器健康
- Healthz：`db=ok`、`schema_version=12`、`version=1.1.0-rc`
- `/api/admin/backups`：无备份时返回 `data: []`

## 6. 未覆盖边界

本轮是本地部署后的移动端截图和关键入口验收，不等同于完整人工回归。以下仍建议在 v1.1 最终冻结前补充：

- 通过 UI 实际提交普通支出和共同支出
- 保存并继续记、复制一笔、模板生成账单的完整 UI 闭环
- 登记结算、复制结算文案和结算历史记录确认
- 附件上传与受控读取
- 手动备份创建、下载和恢复前二次确认
- NAS 地址下的同等浏览器验收
