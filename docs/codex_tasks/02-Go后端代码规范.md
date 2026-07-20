# 后端 Go 代码风格规范

状态：供审核  
适用范围：`backend/`

## 1. 基础原则

1. 所有 Go 代码必须通过 `gofmt`。
2. 包名小写、简短、无下划线。
3. 业务逻辑放 service，HTTP 解析放 handler，SQL 访问放 repository。
4. `context.Context` 作为第一个参数传递。
5. 金额必须使用 `int64 cents`。
6. 不在 handler 中拼业务 SQL。
7. 不在 repository 中写复杂业务判断。
8. 不把内部错误细节返回前端。

## 2. 分层规范

### 2.1 Handler

职责：

- 读取路径参数。
- 读取 query 参数。
- 解析 JSON body。
- 获取 auth / ledger context。
- 调用 service。
- 输出 JSON 响应。

禁止：

- 直接操作数据库。
- 写分摊、结算、权限等业务逻辑。
- 返回未包装错误。

### 2.2 Service

职责：

- 参数校验。
- 权限判断。
- 业务规则。
- 事务编排。
- 审计日志触发。

要求：

- 高风险操作必须事务化。
- 业务错误用 `AppError`。
- 复杂计算拆成纯函数并写单元测试。

### 2.3 Repository

职责：

- SQL 查询。
- insert/update/delete。
- scan 数据。
- 不做业务决策。

要求：

- SQL 字段显式列出，不使用 `SELECT *`。
- 写操作返回 RowsAffected 时应检查是否实际影响数据。
- 所有 ledger scoped 查询必须带 `ledger_id`。

## 3. 错误处理

推荐：

```go
if err != nil {
    return nil, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "金额必须大于 0")
}
```

禁止：

```go
return nil, fmt.Errorf("sql failed: %w", err) // 直接透到前端
```

底层错误可记录日志，但 API 响应必须脱敏。

## 4. 事务规范

```go
dbTx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer dbTx.Rollback()

// writes...

if err := dbTx.Commit(); err != nil {
    return err
}
```

要求：

1. 同一业务操作内的 transaction、splits、tags、audit logs 必须在同一事务。
2. Commit 成功后不要再返回需要回滚的错误。
3. 审计日志失败时，高风险写操作应失败。

## 5. 金额计算

1. 数据库 INTEGER。
2. Go 使用 int64。
3. 比例和份数计算如必须使用小数，必须只作为临时权重，最终金额回到 int64，并校验合计等于总金额。
4. 禁止把 float 写入数据库。
5. 分摊余数规则必须固定并测试。

## 6. 权限规范

1. service 方法优先接收 `LedgerContext`。
2. 所有 ledger scoped 查询带 ledger_id。
3. private 账单不可通过列表、详情、导出、附件访问泄露。
4. 不可见资源优先返回 404，避免暴露资源存在性。
5. viewer 写操作返回 403。

## 7. Migration 规范

1. 使用 goose。
2. 新 migration 文件名必须递增。
3. 必须包含 Up 和 Down。
4. 不修改已应用 migration。
5. 破坏性迁移前需要备份和测试。

## 8. 测试规范

### 8.1 单元测试

文件名：

```text
*_test.go
```

覆盖：

- split calculator。
- settlement calculator。
- RolePolicy。
- LedgerContext resolver。
- money utils。

### 8.2 集成测试

使用临时 SQLite，覆盖：

- 初始化。
- 登录。
- 多账本隔离。
- 权限矩阵。
- 导入导出。
- 备份恢复准备。

## 9. 日志规范

1. 请求日志不输出 body 中的密码、token、secret。
2. 错误日志可以包含 request_id。
3. 审计日志记录 actor、action、entity、before、after。
4. 数据导出、备份、恢复、删除、结算必须审计。

## 10. 运行命令

```bash
cd backend
go fmt ./...
go test -v ./...
go build -o ledger-two ./cmd/server
```
