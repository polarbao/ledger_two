# 测试与质量门禁规范

## 1. 必须运行的命令

### 1.1 后端任务

```bash
cd backend
go test -v ./...
```

### 1.2 前端任务

```bash
cd frontend
pnpm lint
pnpm test
pnpm build
```

### 1.3 部署相关任务

```bash
docker build -t ledger-two:ci -f deploy/docker/Dockerfile .
docker compose config
```

## 2. 测试分层

| 层级 | 目标 |
|---|---|
| 单元测试 | 纯计算、权限策略、工具函数 |
| 集成测试 | API + SQLite + auth + ledger isolation |
| 前端组件测试 | 表单、状态、错误展示 |
| E2E | 登录、记账、结算、导出、备份、权限 |
| 迁移测试 | v1.0 数据升级到新 schema |

## 3. Foundation 阶段新增必测项

1. LedgerContext 解析。
2. owner/editor/viewer 权限矩阵。
3. 多账本数据隔离。
4. private 账单隔离。
5. 导出隔离。
6. 附件访问隔离。
7. 分类/标签/账户归档。
8. migration 可升级。
9. production config validation。
10. query key 包含 ledgerId。

## 4. 禁止行为

1. 不得删除测试来让 CI 通过。
2. 不得降低核心权限测试断言。
3. 不得在测试里使用真实数据库路径。
4. 不得依赖测试执行顺序。
5. 不得把失败测试标记 skip，除非在任务说明里解释并获得确认。

## 5. 完成输出格式

```text
验证命令：
- cd backend && go test -v ./...
- cd frontend && pnpm lint && pnpm test && pnpm build

验证结果：
- 通过 / 未通过

未验证项：
- 未运行 docker build，原因：...
```
