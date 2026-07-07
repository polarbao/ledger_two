# LedgerTwo WSL2 本地部署与 v1.1 后续方案

日期：2026-07-07

## 1. WSL2 本地部署判断

当前项目可以优先部署在本机 WSL2 中，作为 v1.1 之后的日常验收环境。

已验证环境：

- WSL2 发行版：`Ubuntu`
- Docker CLI：已安装
- Docker Compose：已安装
- Docker daemon：可通过 `sudo dockerd` 在 WSL2 中启动
- Go：WSL2 宿主未安装，但 Docker 构建不依赖宿主 Go
- Node：WSL2 宿主已安装，但 Docker 构建使用镜像内 Node/pnpm

结论：

1. 日常开发后优先部署到本机 `http://localhost:38088`。
2. 浏览器验收通过后，再同步部署到 NAS。
3. NAS 不再作为第一验证环境，避免每次小改动都消耗远端构建与真实数据风险。

## 2. 本轮本地部署结果

本轮已在 WSL2 Docker 中部署当前 v1.1 候选版本。

访问地址：

```text
http://localhost:38088
```

健康检查：

```text
GET http://localhost:38088/api/healthz
```

结果：

```text
version=1.1.0-rc
db=ok
schema_version=12
```

容器状态：

```text
ledger-two Up (healthy) 0.0.0.0:38088->8080/tcp
```

本地数据：

- SQLite：`data/ledger.db`
- 备份目录：`backups/`
- 上传目录：`uploads/`
- 日志目录：`logs/`

说明：本地库是独立内测库，不与 NAS 数据互通。

## 3. 本轮发现并修复的问题

### 3.1 Docker 构建上下文污染

现象：

- WSL2 Docker 首次构建失败。
- `COPY frontend/ ./` 将宿主机 `frontend/node_modules` 带入镜像，覆盖 Linux 容器内安装好的依赖。
- 结果是 `pnpm build` 找不到 `typescript/bin/tsc`。
- 构建上下文一度达到约 194MB。

处理：

- 新增根目录 `.dockerignore`。
- 排除 `frontend/node_modules`、`frontend/dist`、本地数据目录、Git/AI 归档和本地部署包。

结果：

- 构建上下文降至约 56KB。
- WSL2 Docker build 通过。

### 3.2 认证页移动端视口约束

现象：

- 初次 headless 截图显示登录页右侧疑似裁切。
- 进一步用 Chrome DevTools Protocol 验证发现普通 `--window-size=375` 并不等于真实 CSS 视口 375px。

处理：

- 为 `#root`、认证页容器和认证卡片补充 `width/min-width/max-width` 约束。
- 移除认证页标题负字距，并补充移动端标题字号与换行规则。

真实 375px CSS 视口验证：

```text
innerWidth=375
documentElement.scrollWidth=375
body.scrollWidth=375
login-card.left=16
login-card.right=359
login-card.width=343
```

结论：登录页在真实 375px CSS 视口下无横向溢出。

## 4. 本地验收建议

本地验收优先级：

1. 初始化/登录：本机库可使用本地测试账号验证，不使用 NAS 真实账号。
2. Dashboard：确认账本上下文、周期提醒入口、统计卡片不溢出。
3. 记账：普通支出、共同支出、保存并继续记。
4. 流水：筛选 bottom sheet、详情、删除确认、批量标签。
5. 模板/复制：复制一笔、从账单创建模板、模板生成账单。
6. 结算：解释字段、影响明细入口、复制结算文案。
7. 设置：分类/标签/账户管理、模板、周期账单、备份、系统诊断。
8. 移动端：375px、390px、430px 三个 CSS 视口宽度下截图或记录 `scrollWidth <= innerWidth`。

## 5. 大版本同步 NAS 策略

推荐流程：

```text
本地代码变更
-> 本地测试与 build
-> WSL2 Docker 本地部署
-> 浏览器验收
-> 提交
-> 大版本或候选版本同步 NAS
-> NAS healthz 与关键路径抽检
```

同步到 NAS 的触发条件：

- v1.1 冻结候选。
- v1.2 进入导入模块候选。
- 数据库 migration 发生变化。
- 安全、备份、附件、导出等部署相关模块发生变化。

不建议每个小 UI 修复都同步 NAS。

## 6. v1.1 后续改造方案

当前 v1.1 不应继续扩大范围，后续只做收口、验收和阻断缺陷修复。

### 6.1 必须完成

1. 本地浏览器验收闭环。
2. 移动端 375px/390px/430px 验收记录。
3. 业务流验收记录写回 `docs/project_analysis`。
4. 若发现阻断缺陷，按 Task41-Task46 对应模块修复并提交。
5. 全部通过后标记 v1.1 冻结。

### 6.2 可以顺手修复

1. 明显横向溢出、按钮遮挡、弹窗操作区不可点击。
2. 文案不清晰但不改变业务语义的提示。
3. 本地部署脚本/文档的小幅补强。

### 6.3 暂不进入

1. CSV 导入预览、去重、导入规则，即 Task47-Task49。
2. OCR、银行同步、直接通知共同支付。
3. 复杂性能重构，例如前端 chunk 拆分，除非实际验收受影响。

## 7. 阶段结论

v1.1 当前处于“本地 WSL2 可部署、NAS 候选已可运行、等待本地浏览器验收冻结”的阶段。

建议后续优先使用本机 WSL2 环境完成验收和小修复；只有当 v1.1 冻结候选稳定后，再同步部署到 NAS。
