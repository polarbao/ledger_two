# LedgerTwo AI 编码任务计划（Task30 已完成后的基础框架阶段）

状态：Task01-Task30 已完成  
当前阶段：Foundation before v1.1  
注意：v1.1 具体业务规划尚未审核完成，本文件仅定义进入 v1.1 之前必须补足的基础框架任务。

## 1. 背景

早期 `docs/18_POST_DEMO_AI_CODING_TASKS.md` 规划了从 Demo 后续到 v1.0 的 Task01-Task30，包括稳定性、安全、备份、导入、附件、多账本、多成员、PWA、离线草稿、迁移、恢复和发布文档。

当前项目已经完成 Task30，并进入一个新的阶段：

1. 不能继续把早期 Demo 约束作为开发事实源。
2. 不能立即实现未审核的 v1.1 业务需求。
3. 需要先把 v1.0/MVP 中已经暴露的基础架构缺口补齐。
4. 后续所有 AI 开发任务应迁移到 `docs/codex_tasks/` 管理。

## 2. 当前任务入口

后续开发任务请从以下文件开始：

```text
docs/codex_tasks/README.md
docs/codex_tasks/00-ai-development-workflow.md
docs/codex_tasks/01-repository-code-style.md
docs/codex_tasks/04-testing-quality-gates.md
docs/codex_tasks/05-foundation-task-plan.md
```

## 3. Foundation Task 列表

| 任务 | 名称 | 目标 |
|---|---|---|
| Task31 | 文档事实源收口 | 对齐 README、PRD、Tech、UI、Task 文档，消除 Demo/v0.3/v1.0 冲突 |
| Task32 | 配置与部署安全框架 | 统一环境变量、JWT/Session 密钥、Cookie/HTTPS 策略、生产启动校验 |
| Task33 | LedgerContext 与 RBAC | 统一账本上下文、成员角色校验、权限矩阵和后端 Guard |
| Task34 | API 契约与 OpenAPI | 明确 `/api` 兼容策略、`/api/v1` 规划、DTO、错误码、分页、OpenAPI |
| Task35 | 分类/标签/支付账户管理基础 | 补齐新增、编辑、排序、归档、恢复、历史保留的基础能力 |
| Task36 | 前端账本状态与 Query Key | 移除切换账本强制 reload，统一 query key，增加 PermissionGate |
| Task37 | 设置页信息架构重组 | 将设置页拆为数据安全、成员/账本、分类/标签/账户、系统诊断等区域 |
| Task38 | 迁移、测试与质量门禁 | 增加迁移测试、多账本隔离、权限矩阵、导出附件权限、安全回归 |
| Task39 | 附件访问控制 | 从静态公开目录访问升级为受权限保护的附件访问 API |
| Task40 | 审计与诊断中心 | 统一审计日志查询、系统健康诊断、配置检查和运维可见性 |

具体任务说明见：

```text
docs/codex_tasks/05-foundation-task-plan.md
```

## 4. 通用执行规则

每个 Foundation Task 必须遵守：

1. 先阅读 `docs/00_DOCUMENT_INDEX.md` 和本任务指定文档。
2. 先输出实现计划和预计修改文件，等待确认。
3. 不实现任务范围外的 v1.1 业务能力。
4. 不修改历史 migration，除非任务明确要求新增 migration。
5. 不能破坏 v1.0 已完成的登录、记账、共同支出、分摊、结算、统计、导入、备份、恢复、导出、离线草稿功能。
6. 完成后运行相关测试和构建。

## 5. 提交建议

每个任务单独提交，提交信息使用 Conventional Commits：

```text
docs: align post-task30 documentation
chore(config): validate production secrets
refactor(auth): introduce ledger context guard
feat(category): add category management APIs
test(rbac): add ledger role regression tests
```
