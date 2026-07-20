# UI-FL-05E 账单编辑契约与验收记录

状态：代码实现与自动化门禁完成；本机运行态真实账号回归待固定后端镜像更新后执行<br>
日期：2026-07-14

## 1. 目标与边界

UI-FL-05E 为流水工作台补齐真正的原账单编辑能力，不使用“复制一笔”伪装编辑，不改变账单类型、历史结算或已冻结金额口径。

本任务不新增 migration、第三方依赖或业务状态；普通账单继续调用 `PATCH /api/transactions/{id}`，共同支出调用 `PATCH /api/shared-expenses/{id}`。

## 2. 冻结契约

1. 只有账本 `owner/editor` 且为账单创建者时可编辑；他人创建的可读账单保持只读。
2. settlement 不能在流水编辑器中修改；账单类型在编辑态锁定。
3. 已保存账单离线时禁止编辑，不得将修改静默转成离线草稿。
4. 前端只提交发生变化的字段；无变化时直接关闭，不产生空 PATCH。
5. 普通账单附件只在列表发生变化时提交完整路径列表；删除只解除账单引用，物理孤儿清理由附件专项负责。
6. 共同支出快捷编辑只支持 `equal/payer_only`，且历史参与人必须与当前账本成员一致；自定义分摊或历史成员不一致时明确禁用入口。
7. 共同支出的金额、付款人或分摊方式变化时由服务端重算 splits；只改标题、备注、分类或标签时必须保留原 split 行和金额。
8. 已归档分类、账户和标签允许在历史账单中保留，但不能作为新值选择；只有显式改动标签时才重建关联。
9. 编辑态不提供模板套用、存为模板或“保存并继续”；关闭脏表单继续复用统一放弃确认。
10. 保存失败在抽屉内展示 API 错误；金额变化与删除继续写审计日志，普通字段更新沿用现有 update 审计记录。

## 3. 实现摘要

- 前端增加编辑源状态、流水表格/详情入口、原账单回填、差异 PATCH、归档元数据保留和离线禁用。
- 后端补齐付款人、分类、账户、可见性和普通/共同字段边界校验。
- repository 仅在 `tag_names` 显式出现时重建标签关系，避免普通字段编辑恢复归档标签。
- service 仅在共同支出分摊相关字段变化时重算 splits，元数据更新不再删除重建历史 split。
- OpenAPI 补齐普通/共同支出更新请求，并将 `SplitInput` 对齐实际 `{user_id, value}` 契约。

## 4. 验证结果

已通过：

- `frontend: corepack pnpm lint`
- `frontend: corepack pnpm test`，18 个测试文件、70 个测试通过
- `frontend: corepack pnpm build`
- `backend: go test ./internal/transaction/... -count=1`
- `backend: CGO_ENABLED=1 + Qt MinGW gcc go test ./internal/http/handler/... ./internal/transaction/... -count=1`
- `docs/api/openapi.yaml` 使用 PyYAML 解析通过
- `git diff --check`

补充说明：默认 PATH 中的 gcc 会使 `go-sqlite3` 目标文件识别失败；显式使用 `C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe` 后 handler 和 transaction 测试均通过。`http://localhost:38088/api/healthz` 仍可返回 `staging / schema 19 / db ok`，但运行实例尚未重建，因此还不包含本任务后端修改。

## 5. 发布门禁

在把 UI-FL-05E 标记为“运行验收完成”前，还需在固定镜像中使用真实账号回归普通账单编辑、共同支出重算、归档元数据保留、附件移除、软删除、批量标签和 CSV 导出。数据库不需要 migration，更新镜像时保留现有 schema 19 数据卷。
