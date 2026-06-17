# 技术：LedgerContext 与 RBAC 权限框架

状态：供审核  
目标：进入 v1.1 前统一账本上下文和角色权限，避免多账本/多成员后发生数据串账和权限散落。

## 1. 背景

当前系统已经存在 `ledgers` 和 `ledger_members`，前端也会通过 `X-Ledger-Id` 传递 active ledger。但业务 service 中仍存在自行查询用户第一个账本作为 fallback 的模式。进入 v1.1 前必须收口。

## 2. 核心对象

```go
type LedgerContext struct {
    UserID      string
    LedgerID    string
    Role        string
    IsExplicit  bool // 是否来自明确的 X-Ledger-Id 或 URL ledgerId
}
```

## 3. 解析规则

### 3.1 必须明确 ledger 的接口

以下接口必须携带明确 ledger：

- 账单。
- 共同支出。
- 结算。
- 分类、标签、账户。
- 统计。
- 导入导出。
- 备份恢复审计上下文。
- 附件访问。

### 3.2 可不携带 ledger 的接口

以下接口可不携带 ledger：

- 登录。
- 退出。
- 获取当前用户。
- 获取我的账本列表。
- 获取我收到的邀请，待 v1.1 确认。

### 3.3 禁止 fallback 规则

长期目标：业务写接口不得自动 fallback 到用户第一个账本。过渡期可以保留兼容，但必须有 warning 日志，并在文档中标记弃用。

## 4. 权限矩阵

| 操作 | owner | editor | viewer |
|---|---:|---:|---:|
| 查看账本 | 是 | 是 | 是 |
| 查看成员 | 是 | 是 | 是 |
| 修改账本名称 | 是 | 否 | 否 |
| 管理成员 | 是 | 否 | 否 |
| 新增账单 | 是 | 是 | 否 |
| 编辑自己创建的账单 | 是 | 是 | 否 |
| 编辑他人账单 | 可选，默认否 | 否 | 否 |
| 删除自己创建的账单 | 是 | 是 | 否 |
| 创建共同支出 | 是 | 是 | 否 |
| 生成结算记录 | 是 | 是 | 否 |
| 查看统计 | 是 | 是 | 是 |
| 导出 CSV/JSON | 是 | 可配置，默认否 | 否 |
| 手动备份 | 是 | 否 | 否 |
| 恢复备份 | 是 | 否 | 否 |
| 管理分类/标签/账户 | 是 | 可配置 | 否 |

## 5. 可见性规则

| visibility | 多成员语义 |
|---|---|
| private | 仅创建者、owner_user、payer_user 可见；不得被其他成员导出或看到附件 |
| partner_readable | 旧双人语义为对方可读；多成员阶段需重命名或升级为 member_readable / selected_members |
| shared | 账本内成员可见，并参与共同结算，除非后续引入参与人可见性 |

Foundation 阶段先不重命名字段，但必须在文档中标记：`partner_readable` 是历史兼容名，多成员后需重新定义或迁移。

## 6. 后端实现建议

```go
func ResolveLedgerContext(r *http.Request) (*LedgerContext, error)
func RequireLedgerRole(roles ...Role) func(http.Handler) http.Handler
func CanViewTransaction(ctx LedgerContext, tx Transaction) bool
func CanEditTransaction(ctx LedgerContext, tx Transaction) bool
```

Service 方法建议从：

```go
Create(ctx context.Context, currentUserID string, req Request)
```

演进为：

```go
Create(ctx context.Context, lc LedgerContext, req Request)
```

## 7. 审计日志

所有高风险操作必须记录：

```text
ledger_id
actor_user_id
actor_role
action
entity_type
entity_id
before_json
after_json
created_at
```

建议 migration 给 audit_logs 增加 `actor_role`，或在 after_json 中记录 role。

## 8. 测试要求

1. A 账本 owner 不可访问 B 账本数据。
2. editor 可以新增账单，但不能管理成员。
3. viewer 不能新增、编辑、删除账单。
4. private 账单不会被其他成员列表、详情、导出、附件 API 看到。
5. 切换 `X-Ledger-Id` 到非成员账本返回 403 或 404。
6. 未携带 ledger 的业务写接口返回明确错误。

## 9. 验收标准

1. 所有业务写接口使用统一 LedgerContext。
2. RolePolicy 有单元测试。
3. 业务 service 不再复制 `SELECT role FROM ledger_members`。
4. 前端 PermissionGate 与后端权限矩阵一致。
