# Task30 后当前实现与 PRD / Tech / UI 文档差异审阅

日期：2026-06-17  
状态：供审核  
结论：当前代码已推进到 v1.0/MVP 完成状态，但仓库文档中仍存在大量 Demo/v0.3/v0.2 表述，后续开发前必须进行文档事实源和基础架构收口。

## 1. 审阅范围

本次审阅基于当前 GitHub 仓库中的以下内容：

- `README.md`
- `CHANGELOG.md`
- `docs/00_DOCUMENT_INDEX.md`
- `docs/prd/00-product-roadmap.md`
- `docs/tech/*`
- `docs/ui/*`
- `docs/18_POST_DEMO_AI_CODING_TASKS.md`
- 后端 `backend/internal/*`
- 前端 `frontend/src/*`
- Docker Compose、CI、Dockerfile

## 2. 总体结论

当前项目不需要推倒重写，但需要在进入 v1.1 之前完成基础架构补齐。

主要原因：

1. 项目代码已经完成 Task30 规划中的大量功能，包括多账本/成员雏形、导入、备份、恢复、PWA/离线草稿、多人分摊和结算建议等。
2. 文档仍混杂早期 Demo、v0.2、v0.3、v1.0 和待审核 v1.1 的描述。
3. 一些基础框架已经有雏形，但尚未达到可支撑 v1.1 长期开发的统一程度，例如账本上下文、RBAC、配置安全、API 合同、分类标签账户管理和测试矩阵。

## 3. 明显差异矩阵

| 领域 | 当前代码/仓库状态 | 当前文档状态 | 差异与风险 | 建议处理 |
|---|---|---|---|---|
| 项目状态 | README 和 CHANGELOG 已标记 v1.0.0 正式发布 | `docs/00_DOCUMENT_INDEX.md` 仍是 v0.3，`docs/prd/00-product-roadmap.md` 仍说下一阶段 v0.2 | AI 读取文档后会误判项目阶段 | Task31 先统一文档事实源 |
| 任务状态 | 用户确认 Task01-Task30 已完成 | `docs/18_POST_DEMO_AI_CODING_TASKS.md` 仍是原后续任务列表 | 后续任务入口不清楚 | 将 Task31+ 放入 `docs/codex_tasks/` |
| 多账本 | 已有 `ledgers`、`ledger_members`、账本切换和成员 API | 旧文档仍多处强调 Demo 固定双人/单账本 | 业务模型正在升级但文档未同步 | 明确当前是多账本雏形，不等于完整 v1.1 邀请机制 |
| 成员管理 | Owner 可按用户名直接添加成员 | v1.1 未审核草案要求“邀请已有用户后接受” | 当前是直接添加，不是邀请状态机 | v1.1 审核前先补 RBAC，不直接实现邀请 |
| 初始化 | 初始化仍一次创建两个用户和一个账本 | v1.1 尚未审核的新模式要求单人账号/双人账本并存 | 新创建模式与现有初始化冲突 | 基础框架阶段先设计账号/账本解耦，不实现未审业务 |
| 分类/标签 | 分类/账户目前主要提供查询，标签多由记账关联自动生成 | UI 文档已要求分类排序、归档、标签管理 | 文档要求高于实现 | Task35 补基础管理 API/UI |
| 配置安全 | Docker Compose 使用 `SESSION_SECRET`，后端读取 `JWT_SECRET`；生产无强密钥仍可能 fallback | 部署文档没有统一配置事实源 | 生产安全风险和登录 Cookie 风险 | Task32 统一配置和 Cookie 策略 |
| 前端账本状态 | active ledger 存在 Zustand，切换账本后 reload | 文档未明确多账本前端状态规范 | v1.1 多账本体验和缓存容易错乱 | Task36 引入 LedgerProvider 和 query key 工厂 |
| API 合同 | 当前实际路由是 `/api` | 技术文档曾要求 `/api/v1` 和 OpenAPI | 合同不稳定，不利于跨端 | Task34 建立 API 兼容与 OpenAPI 计划 |
| 附件 | 物理上传文件通过 `/uploads/*` 静态托管 | 隐私保护没有细化 | private 附件可能绕开账单权限 | Task39 增加受保护附件 API |
| 测试 | CI 已有后端、前端、Docker 构建 | v1.1 前缺少权限矩阵、多账本隔离、迁移回归 | 大模型后续开发容易破坏核心账务 | Task38 建立质量门禁 |

## 4. 对是否重构的判断

不建议整体重写。当前技术栈和部署方式仍适合个人/情侣/家庭私有化记账：

- Go + SQLite + REST JSON 适合 NAS 私有部署。
- React + TypeScript + Vite 适合 Web/PWA。
- Docker Compose + SQLite 单文件备份适合家庭场景。

但建议进行局部重构：

1. 后端拆清 `transaction`、`split`、`recurring`、`importer`、`category`、`tag`、`account`、`ledger`、`membership`、`safety` 等边界。
2. 统一 LedgerContext 和 RBAC，不让各 service 自行 fallback 到用户第一个 ledger。
3. 统一配置变量和生产启动校验。
4. 统一 API 合同和 OpenAPI 文档。
5. 统一前端 active ledger、query key、PermissionGate 和设置页结构。

## 5. 当前不建议做的事

1. 不要直接开始 v1.1 具体业务功能开发。
2. 不要一次性大重构全项目。
3. 不要在没有测试矩阵前重写账本和结算算法。
4. 不要继续在 `transaction.Service` 中堆叠更多业务。
5. 不要直接把“添加成员”当成“邀请机制”完成。

## 6. 推荐下一步

1. 合并本包文档到新分支 `docs/foundation-before-v1.1`。
2. 根据 `docs/codex_tasks/05-foundation-task-plan.md` 从 Task31 开始执行。
3. 每个任务只做基础框架补齐，不做 v1.1 未审核业务。
4. 待 v1.1 具体需求审核完成后，再根据新的 PRD 创建 v1.1 业务任务。
