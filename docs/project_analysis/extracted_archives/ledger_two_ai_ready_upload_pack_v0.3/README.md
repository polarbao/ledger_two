# LedgerTwo

LedgerTwo 是一个面向两个人使用的私有化共享记账 Web 工具，目标部署在本地 NAS，支持个人流水、共同支出、分摊、结算、统计、CSV/JSON 导出与 Docker 部署。

## 当前状态

当前仓库处于 Demo 设计与工程初始化阶段。v0.3 文档已经补齐 AI 编码实现所需的模块规格、测试规格和 Mac/Windows 双开发环境配置。

## 文档入口

请从这里开始阅读：

```text
docs/00_DOCUMENT_INDEX.md
```

核心文档：

```text
docs/01_PRD.md
docs/02_UI_INTERACTION_DESIGN.md
docs/03_TECH_DESIGN.md
docs/04_TECH_IMPLEMENTATION.md
docs/07_DATABASE_API.md
docs/13_DEMO_SCOPE_LOCK.md
docs/14_BACKEND_MODULE_SPEC.md
docs/15_FRONTEND_MODULE_SPEC.md
docs/16_TEST_ACCEPTANCE_SPEC.md
docs/17_AI_CODING_TASKS.md
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

Demo 版本只做：

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

## 本地启动，后续代码实现后

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

## AI 编码规则

AI/Codex 开始编码前必须阅读：

```text
AGENTS.md
docs/00_DOCUMENT_INDEX.md
docs/13_DEMO_SCOPE_LOCK.md
```
