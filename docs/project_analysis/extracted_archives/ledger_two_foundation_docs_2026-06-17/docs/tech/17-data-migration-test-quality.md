# 技术：数据迁移、测试与质量门禁

状态：供审核  
目标：保障 v1.0 数据安全演进到 Foundation/v1.1，不因基础框架补齐破坏真实账务数据。

## 1. 数据迁移原则

1. 已应用 migration 不得随意修改。
2. 新增 migration 必须同时提供 Up 和 Down。
3. migration 不得静默删除账务数据。
4. 所有破坏性字段变更必须先备份。
5. SQLite 备份优先使用安全在线备份方式。
6. schema version 必须能在 healthz 或诊断接口中查看。

## 2. Foundation 期间可能新增 migration

| 能力 | 表/字段 |
|---|---|
| 配置诊断 | app_settings 增加 version/build 信息，可选 |
| RBAC 审计 | audit_logs 增加 actor_role，可选 |
| 分类管理 | categories 增加 is_archived、archived_at、updated_by_user_id |
| 标签管理 | tags 增加 sort_order、is_archived、archived_at、updated_by_user_id |
| 账户管理 | accounts 增加 sort_order、archived_at、updated_by_user_id |
| 附件权限 | transaction_attachments 表或 attachment 元数据表 |
| API 幂等 | 高风险写接口 idempotency key，可后续 |

## 3. 测试矩阵

### 3.1 后端单元测试

必须覆盖：

- Money 元/分转换。
- SplitCalculator equal/payer_only/amount/ratio/shares。
- SettlementCalculator 多人净额和建议转账。
- RolePolicy owner/editor/viewer。
- LedgerContext 解析。
- Category/Tag/Account 归档规则。

### 3.2 后端集成测试

必须覆盖：

1. 初始化后可以登录。
2. 用户只能访问自己所属账本。
3. active ledger 切换后数据隔离。
4. viewer 不能新增账单。
5. editor 不能管理成员。
6. owner 可以管理基础配置。
7. private 账单不出现在他人列表、导出和附件访问中。
8. 备份/恢复准备写审计日志。

### 3.3 前端测试

必须覆盖：

1. API client 错误解析。
2. query key 包含 ledgerId。
3. PermissionGate 按角色隐藏/禁用操作。
4. 切换账本不 reload 的长期方案。
5. 分类/标签/账户管理表单校验。
6. 离线草稿不进入正式统计。

### 3.4 E2E 测试，建议

使用 Playwright 后续覆盖：

1. 登录。
2. 创建账本。
3. 切换账本。
4. 新增普通账单。
5. 新增共同支出。
6. 查看结算。
7. 导出 CSV。
8. 创建备份。
9. 分类归档。
10. viewer 权限拦截。

## 4. CI 门禁

当前 CI 应至少执行：

```bash
cd backend && go test -v ./...
cd frontend && pnpm lint
cd frontend && pnpm test
cd frontend && pnpm build
docker build -t ledger-two:ci -f deploy/docker/Dockerfile .
```

Foundation 期间建议增加：

- migration test。
- markdown link check，可选。
- OpenAPI validate。
- docker compose config validate。
- secret scan，可选。

## 5. 本地开发门禁

每个 Codex/Gemini 任务完成后必须输出：

```text
验证命令：
- go test -v ./...
- pnpm lint
- pnpm test
- pnpm build
- docker build ...
```

如果只改文档，可以运行：

```bash
git diff --check
```

## 6. 验收标准

1. v1.0 数据可迁移到 Foundation schema。
2. 核心记账、结算、导入、导出、备份不回归。
3. 多账本和权限测试覆盖。
4. CI 失败时禁止合并。
5. 每次 Foundation Task 都有验证命令和风险说明。
