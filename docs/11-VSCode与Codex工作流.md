# VSCode + Codex 工作流

## 1. 推荐规则

Codex 可以辅助生成代码、重构、测试和文档，但账务系统的核心规则必须固定：

- 金额使用整数分，禁止 float。
- 付款人、承担人、记账人、归属人必须区分。
- 共同支出必须生成 split 记录。
- 结算必须生成 settlement 记录，不直接改旧账单。
- private 账单不能被对方看到。

## 2. 项目文件

仓库根目录放置：

```text
AGENTS.md
.codex/config.toml
.vscode/settings.json
.vscode/extensions.json
.vscode/tasks.json
```

## 3. 推荐工作流

1. 先让 Codex 阅读 `docs/01-产品需求文档.md`、`docs/03-技术设计.md`、`docs/07-数据库与API设计.md`、`AGENTS.md`。
2. 一次只让 Codex 实现一个模块。
3. 每次实现后要求 Codex 说明改动文件并运行测试。
4. 数据库 migration、金额计算、结算逻辑必须人工复查。

## 4. 示例提示词

```text
请阅读 docs/01-产品需求文档.md、docs/03-技术设计.md、docs/07-数据库与API设计.md 和 AGENTS.md。
先实现 backend 的 users/auth/categories 基础模块。
要求 Handler 不写业务逻辑，Service 与 Repository 分层，完成后运行 go test ./...
```

```text
请实现 shared_expense 创建逻辑。
要求金额使用 int64 cents，支持 equal 和 payer_only，必须写 transaction_splits，不允许使用 float。
```
