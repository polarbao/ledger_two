# LedgerTwo 文档索引与 AI 实现阅读顺序 v0.3

本文档用于让人类开发者、Codex、Cursor、Copilot 或其他 AI 编码模型快速理解项目并按正确顺序实现代码。

## 1. 当前结论

v0.2 文档已经足够支持产品讨论、UI 原型和总体技术选型，但对于“直接交给 AI 模型连续生成可运行代码”还不够。主要缺口是：

1. 缺少明确的 MVP 裁剪边界，AI 容易一次性实现过多功能。
2. 缺少按模块的后端 handler / service / repository / migration 实现细则。
3. 缺少统一 DTO、错误码、校验规则和权限矩阵。
4. 缺少前端页面级组件树、表单字段、API 对接策略和响应式规则。
5. 缺少可执行测试用例和验收命令。
6. 缺少 AI 编码任务切片和提示词模板。
7. Mac Air 与 Windows PC 双开发环境的配置步骤不够细。

v0.3 文档包补齐以上缺口。Demo 版本按 v0.3 执行，AI 模型可以分模块完成代码编写，但仍建议每个阶段由人类开发者 review 数据库迁移、金额计算和权限控制。

## 2. 推荐阅读顺序

AI 编码模型必须按以下顺序阅读：

```text
00_DOCUMENT_INDEX.md
01_PRD.md
02_UI_INTERACTION_DESIGN.md
03_TECH_DESIGN.md
04_TECH_IMPLEMENTATION.md
07_DATABASE_API.md
13_DEMO_SCOPE_LOCK.md
14_BACKEND_MODULE_SPEC.md
15_FRONTEND_MODULE_SPEC.md
16_TEST_ACCEPTANCE_SPEC.md
17_AI_CODING_TASKS.md
09_DEV_ENV_MAC.md 或 10_DEV_ENV_WINDOWS.md
11_VSCODE_CODEX_WORKFLOW.md
```

## 3. 文档清单

| 文件 | 作用 |
|---|---|
| `01_PRD.md` | 产品需求、角色、页面、功能、验收 |
| `02_UI_INTERACTION_DESIGN.md` | UI 交互、桌面/移动端页面、交互状态 |
| `03_TECH_DESIGN.md` | 架构、选型、模块边界、安全、跨端预留 |
| `04_TECH_IMPLEMENTATION.md` | 后端/前端实现路线、工程结构、运行方式 |
| `05_FRONTEND_DESIGN.md` | 前端页面、组件、状态、表单、样式方案 |
| `06_NAS_DEPLOYMENT.md` | 群晖 NAS Docker 部署、备份、恢复 |
| `07_DATABASE_API.md` | 数据库 schema、索引、API 合同 |
| `08_MVP_ROADMAP.md` | 里程碑、版本计划、开发顺序 |
| `09_DEV_ENV_MAC.md` | Mac Air 开发环境详细配置 |
| `10_DEV_ENV_WINDOWS.md` | Windows PC + WSL2 开发环境详细配置 |
| `11_VSCODE_CODEX_WORKFLOW.md` | VSCode + Codex 工作流 |
| `12_LOCAL_UPLOAD_GUIDE.md` | 本地上传到 GitHub 仓库说明 |
| `13_DEMO_SCOPE_LOCK.md` | Demo 版本范围锁定，防止 AI 过度实现 |
| `14_BACKEND_MODULE_SPEC.md` | 后端模块级实现规格，AI 可直接按模块编码 |
| `15_FRONTEND_MODULE_SPEC.md` | 前端页面/组件级实现规格，AI 可直接按页面编码 |
| `16_TEST_ACCEPTANCE_SPEC.md` | 自动化测试、手工验收、核心业务测试用例 |
| `17_AI_CODING_TASKS.md` | AI 编码任务拆分与提示词模板 |

## 4. AI 实现约束

AI 编码模型必须遵守：

1. Demo 版本只做固定双人账本，不做多租户、多家庭、多成员。
2. 金额全部用整数分 `amount_cent` / `amount_cents`，禁止 float。
3. 后端采用 Go + SQLite + REST JSON。
4. 前端采用 React + TypeScript + Vite。
5. 删除账单必须软删除。
6. 共同支出必须生成 split 记录。
7. 结算必须生成 settlement 记录，不允许直接修改历史账单抵消金额。
8. private 账单对方不可见。
9. 业务逻辑放 service 层，不要塞进 handler。
10. 所有金额修改写 audit log。

## 5. 推荐开发分支

```bash
git checkout -b docs/ai-ready-v0.3
```

或直接提交到 main：

```bash
git add .
git commit -m "docs: add AI-ready implementation specs and dev setup"
git push origin main
```
