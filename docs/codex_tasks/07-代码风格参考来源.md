# 代码风格参考来源

状态：供审核

本项目代码风格规范参考了公开开源生态和官方文档，并按 LedgerTwo 的具体技术栈进行裁剪。

## 1. Go

- Effective Go：用于确定 gofmt、包名、命名、注释和错误处理的基础原则。
- Go Code Review Comments：用于补充 Go 代码审查中常见的命名、错误处理、context、接口设计和测试建议。

LedgerTwo 裁剪原则：

1. Go 代码必须 gofmt。
2. 后端保持 handler/service/repository 分层。
3. context.Context 作为服务方法的第一个参数。
4. 钱统一 int64 cents。
5. SQLite SQL 显式字段，不使用 SELECT *。

## 2. React / TypeScript

- React Rules of Hooks：用于约束 Hooks 调用位置和组件行为。
- TypeScript 官方文档：用于类型定义、DTO、union type 和类型安全实践。
- typescript-eslint：用于 lint 和类型感知规则。
- TanStack Query Query Keys：用于约束服务端状态缓存 key。

LedgerTwo 裁剪原则：

1. TanStack Query 管理服务端状态。
2. Zustand 只管理 UI 状态。
3. query key 必须包含 ledgerId。
4. 表单使用 React Hook Form + Zod。
5. 金额输入展示元，API 提交分。

## 3. Git / CI

- Conventional Commits：用于提交信息和 changelog。
- GitHub Actions 官方文档：用于 CI 工作流组织。

LedgerTwo 裁剪原则：

1. 每个任务单独分支。
2. 每个任务小步提交。
3. PR 必须包含验证命令。
4. CI 必须覆盖 backend test、frontend lint/test/build、docker build。
