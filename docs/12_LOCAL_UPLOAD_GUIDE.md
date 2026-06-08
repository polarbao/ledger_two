# 本地上传 GitHub 指南

## 1. 上传路径

把本压缩包解压后的所有内容复制到 GitHub 仓库根目录：

```text
ledger_two/
```

也就是本地克隆后的目录：

```bash
git clone https://github.com/polarbao/ledger_two.git
cd ledger_two
```

最终路径应类似：

```text
ledger_two/
  README.md
  AGENTS.md
  .env.example
  docker-compose.yml
  .vscode/
  .codex/
  docs/
  backend/
  frontend/
  deploy/
```

## 2. 文档对应上传路径

| 文件 | 上传到仓库路径 |
|---|---|
| README.md | `README.md` |
| AGENTS.md | `AGENTS.md` |
| Codex 配置 | `.codex/config.toml` |
| VSCode 配置 | `.vscode/` |
| PRD | `docs/01_PRD.md` |
| UI 交互设计 | `docs/02_UI_INTERACTION_DESIGN.md` |
| 技术设计 | `docs/03_TECH_DESIGN.md` |
| 技术实现 | `docs/04_TECH_IMPLEMENTATION.md` |
| 前端设计 | `docs/05_FRONTEND_DESIGN.md` |
| NAS 部署 | `docs/06_NAS_DEPLOYMENT.md` |
| 数据库/API | `docs/07_DATABASE_API.md` |
| MVP 路线图 | `docs/08_MVP_ROADMAP.md` |
| Mac 环境 | `docs/09_DEV_ENV_MAC.md` |
| Windows 环境 | `docs/10_DEV_ENV_WINDOWS.md` |
| Codex 工作流 | `docs/11_VSCODE_CODEX_WORKFLOW.md` |

## 3. 提交命令

复制文件后执行：

```bash
git status
git add .
git commit -m "docs: add project documents and dev environment setup"
git push origin main
```

如果远程已有 README.md，可以直接覆盖或手动合并。建议使用本包中的 README.md 作为主 README。
