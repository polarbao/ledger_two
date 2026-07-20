# 仓库通用代码风格规范

状态：供审核  
适用范围：LedgerTwo 全仓库，包括 Go 后端、React/TypeScript 前端、SQL migration、Markdown 文档、Docker 和 GitHub Actions。

## 1. 参考规范

本规范结合以下开源项目和官方生态的通用实践整理：

1. Go 官方 Effective Go 和 Go Code Review Comments：强调 gofmt、短包名、清晰命名、早返回、错误显式处理。
2. React 官方 Rules of Hooks：强调 Hooks 只能在 React 函数组件或自定义 Hook 顶层调用。
3. TypeScript 官方文档与 typescript-eslint：强调静态类型、lint 和类型安全。
4. TanStack Query 官方文档：强调 query key 必须唯一描述服务端数据来源。
5. Conventional Commits：强调提交信息机器可读，便于 changelog 和版本管理。
6. GitHub Actions 官方文档：强调 CI 工作流作为质量门禁。

这些外部规范不会被照搬，而是按 LedgerTwo 的 Go + SQLite + React + TypeScript + NAS 私有化部署场景裁剪。

## 2. 总原则

1. 可读性优先于技巧。
2. 数据安全优先于开发速度。
3. 明确边界优先于复用幻觉。
4. 小函数、小模块、小提交。
5. 测试和文档是功能的一部分。
6. AI 生成代码必须经过人类审核。

## 3. 目录约定

```text
backend/
  cmd/server
  internal/
  migrations/
frontend/
  src/
  public/
docs/
  prd/
  tech/
  ui/
  codex_tasks/
deploy/
  docker/
```

新增代码必须放入对应领域模块，不允许把所有新功能继续堆进单个 service 或单个页面。

## 4. 命名规范

| 类型 | 规范 |
|---|---|
| Go package | 小写、短名、无下划线 |
| Go exported type | MixedCaps |
| Go private name | mixedCaps |
| TypeScript type/interface | PascalCase |
| React component | PascalCase |
| Hook | useXxx |
| 文件名 | 前端 kebab-case 或现有项目风格，后端按 Go 包风格 |
| API JSON 字段 | snake_case |
| 数据库字段 | snake_case |
| 金额字段 | `*_cents` |
| 时间字段 | `*_at` 或明确 date 字段 |

## 5. 金额和时间

1. 金额在后端、数据库、API 中统一为整数分。
2. 前端展示元，提交前转换为分。
3. 禁止使用 float 做金额存储和核心计算。
4. 时间统一 ISO8601，纯日期字段使用 `YYYY-MM-DD`。
5. 统计按后端时间口径计算，前端不自行聚合最终金额。

## 6. 错误处理

1. 后端错误统一使用 AppError。
2. 不把内部堆栈和 SQL 细节返回前端。
3. 前端 API client 统一解析 `success=false`。
4. 用户可理解的错误信息使用中文。
5. 日志和审计信息不得包含密码、token、secret。

## 7. Commit 规范

采用 Conventional Commits：

```text
<type>(optional-scope): <description>
```

常用 type：

```text
feat      新功能
fix       修复
refactor  重构，不改变外部行为
docs      文档
style     格式，不影响逻辑
test      测试
chore     构建、依赖、工具
ci        CI/CD
perf      性能优化
```

示例：

```text
docs: align post-task30 foundation plan
refactor(auth): introduce ledger context guard
feat(category): add category archive API
test(rbac): cover viewer write denial
fix(config): reject default jwt secret in production
```

## 8. PR 规范

每个 PR 必须包含：

1. 背景。
2. 修改内容。
3. 不做事项。
4. 验证命令。
5. 风险和回滚方式。
6. 截图，若改 UI。

## 9. Markdown 文档规范

1. 标题层级从 `#` 开始，不跳级。
2. 每个需求文档必须包含：目标、范围、不做事项、验收标准。
3. 每个技术文档必须包含：设计原则、接口/数据结构、风险、测试。
4. AI 任务必须包含：目标、输入文档、禁止事项、测试要求、验收标准。
5. 文档中的版本状态必须明确：已完成 / 进行中 / 待审核 / 废弃。

## 10. 安全规范

1. 不提交 `.env`。
2. 不提交数据库、备份、上传文件。
3. 不在日志中输出 token、password、secret。
4. 上传文件必须校验大小和类型。
5. 附件访问必须经过权限校验。
6. 导出和备份属于高风险操作，需要权限和审计。

## 11. AI 生成代码限制

AI 不得：

1. 删除测试来让 CI 通过。
2. 修改已应用 migration。
3. 引入未审核的大型依赖。
4. 重写整个项目。
5. 只改前端隐藏按钮而不改后端权限。
6. 生成未登记 API。
7. 使用伪造测试结果。
