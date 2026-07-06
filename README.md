# LedgerTwo

LedgerTwo 是一个面向两个人使用的私有化共享记账 Web 工具，目标部署在本地 NAS，支持个人流水、共同支出、分摊、结算、统计、CSV/JSON 导出与 Docker 部署。

## 当前状态

🎉 **v1.0.0 正式版已发布！** 详情请查看 [CHANGELOG.md](./CHANGELOG.md)。

当前 **Foundation before v1.1 已完成基础冻结**。迁移回归、RBAC、CI 门禁、附件访问控制和 Task40「审计与系统诊断中心」已补齐；中期 v1.1 已从 Task44「分类、标签、账户管理体验」启动。

## 文档入口

请从这里开始阅读：

```text
docs/00_DOCUMENT_INDEX.md
CHANGELOG.md        # 版本变更与发布说明
```

核心文档：

```text
docs/prd/21-roadmap-short-mid-long.md
docs/prd/24-short-mid-module-breakdown.md
docs/prd/25-prd-v1.1-module-specs.md
docs/prd/26-prd-v1.2-import-module-specs.md
docs/tech/18-short-mid-architecture-slices.md
docs/tech/19-short-mid-implementation-readiness.md
docs/codex_tasks/10-task33-40-detailed-plan.md
docs/project_analysis/2026-07-06-short-mid-long-and-nas-readiness.md
```

开发环境：

```text
docs/09_DEV_ENV_MAC.md
docs/10_DEV_ENV_WINDOWS.md
docs/11_VSCODE_CODEX_WORKFLOW.md
```

## 推荐技术栈

```text
Frontend: React + TypeScript + Vite + Tailwind + TanStack Query
Backend: Go + SQLite + REST JSON
Deploy: Docker Compose on Synology NAS
AI Workflow: VSCode + Codex
```

## Demo 范围

历史 Demo 版本只做：

1. 固定两人共享账本。
2. 登录与初始化。
3. 普通支出/收入。
4. 共同支出。
5. 平均分摊和仅付款人承担。
6. 结算中心。
7. 首页、流水、统计、设置。
8. SQLite 数据持久化。
9. Docker/NAS 部署。

不做多账本、多成员、银行同步、OCR、预算、App 客户端。

## 本地启动与测试验证

### 本地启动

后端：

```bash
cd backend
go run ./cmd/server
```

前端：

```bash
cd frontend
pnpm install
pnpm dev
```

Docker：

```bash
docker compose up -d --build
```

NAS 部署请优先阅读：

```text
docs/tech/08-nas-deployment.md
.env.example
docker-compose.yml
```

### 2. 前端测试与编译 (React)
前端提供了 Lint 规约检查、TypeScript 静态类型检测以及单元测试：
```bash
cd frontend
# 运行 ESLint 规约检查
npx pnpm lint
# 运行前端单元测试 (Vitest)
npx pnpm test
# 运行前端打包构建编译
npx pnpm build
```

### 3. Docker 镜像本地构建校验
校验生产多阶段 Docker 构建的完整性与稳定性：
```bash
docker compose build
```

## AI 编码规则

AI/Codex 开始编码前必须阅读：

```text
AGENTS.md
docs/00_DOCUMENT_INDEX.md
docs/13_DEMO_SCOPE_LOCK.md
```
