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

## 本地测试与质量门禁

为了确保业务计算、权限和结算逻辑的稳定性，建议在提交代码前进行本地测试与静态校验。

### 1. 后端测试 (Go)
由于后端使用 SQLite 且在编译时开启了 CGO，本地测试推荐且必须在 **WSL2** 或提供 GCC 编译的 Linux 虚拟机环境下运行：
```bash
# 进入后端目录，在 WSL2 下执行全部集成与单元测试
wsl bash -c "cd backend && /usr/local/go/bin/go test -v ./..."
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
