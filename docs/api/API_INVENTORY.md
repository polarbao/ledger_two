# API Inventory

状态：Task34.1 现状盘点  
来源：`backend/internal/http/router/router.go`  
当前实现基路径：`/api`  
目标版本基路径：`/api/v1`，尚未实现 alias  
更新时间：2026-07-06

## 1. 总体约定

响应格式：

```json
{
  "success": true,
  "data": {}
}
```

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "金额必须大于 0",
    "details": null
  }
}
```

认证方式：

- Web 当前使用 HttpOnly Cookie 中的 `token`。
- 受保护业务接口必须先通过 `RequireAuth`。
- 显式账本请求使用 `X-Ledger-Id`。

账本要求：

- `none`：不需要账本上下文。
- `optional`：当前兼容未传 `X-Ledger-Id`，服务层 fallback 到用户第一个账本；v1.1 冻结前应逐步收紧。
- `required`：请求必须明确账本或路径内账本，且必须校验 membership。
- `path`：账本 ID 来自 URL path。

稳定性：

- `stable`：可作为 v1.1 契约冻结。
- `transitional`：当前可用，但有 fallback、错误码、字段或权限治理债务。
- `deprecated`：历史兼容或存在安全风险。
- `internal`：非业务客户端契约。

## 2. 公共与初始化

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/healthz` | no | none | internal | inline | 健康检查，返回服务、数据库和 schema version。 |
| GET | `/api/init/status` | no | none | stable | `init.HandleStatus` | 初始化状态。 |
| POST | `/api/init/setup` | no | none | stable | `init.HandleSetup` | 初始化系统、用户和初始账本。 |

## 3. Auth

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/auth/login` | no | none | stable | `auth.HandleLogin` | 登录并写入 Cookie token。 |
| POST | `/api/auth/logout` | no | none | stable | `auth.HandleLogout` | 清理 Cookie token。 |
| GET | `/api/auth/me` | yes | none | stable | `auth.HandleMe` | 当前用户信息，可能包含默认 ledger id。 |

## 4. Ledgers 与成员

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/ledgers/` | yes | none | stable | `ledger.CreateLedger` | 创建账本，并将创建者设为 owner。 |
| GET | `/api/ledgers/` | yes | none | stable | `ledger.ListUserLedgers` | 列出当前用户加入的账本。 |
| GET | `/api/ledgers/{id}/members` | yes | path | transitional | `ledger.GetLedgerMembers` | 查看账本成员，当前 path 账本权限由 service 校验。 |
| POST | `/api/ledgers/{id}/members` | yes | path | transitional | `ledger.AddMember` | 添加成员，仅 owner。 |
| PUT | `/api/ledgers/{id}/members/{userId}` | yes | path | transitional | `ledger.UpdateMemberRole` | 修改成员角色，仅 owner。 |
| DELETE | `/api/ledgers/{id}/members/{userId}` | yes | path | transitional | `ledger.RemoveMember` | 移除成员，仅 owner。 |

## 5. Metadata 基础查询

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/categories` | yes | optional | transitional | `transaction.HandleListCategories` | 列出当前账本分类。 |
| GET | `/api/accounts` | yes | optional | transitional | `transaction.HandleListAccounts` | 列出当前账本支付账户。 |
| GET | `/api/metadata/{kind}/` | yes | optional | transitional | `metadata.List` | 元数据列表，kind 为 categories/tags/accounts，支持 include_archived，返回 `sort_order` 与 `usage_count`。 |
| POST | `/api/metadata/{kind}/` | yes | optional | transitional | `metadata.Create` | 创建分类、标签或账户，仅 owner。 |
| POST | `/api/metadata/{kind}/reorder` | yes | optional | transitional | `metadata.Reorder` | 调整分类、标签或账户排序，仅 owner。 |
| PATCH | `/api/metadata/{kind}/{id}` | yes | optional | transitional | `metadata.Update` | 更新分类、标签或账户，仅 owner。 |
| POST | `/api/metadata/{kind}/{id}/archive` | yes | optional | transitional | `metadata.Archive` | 归档分类、标签或账户，仅 owner。 |
| POST | `/api/metadata/{kind}/{id}/restore` | yes | optional | transitional | `metadata.Restore` | 恢复归档分类、标签或账户，仅 owner。 |

说明：旧 `/api/categories`、`/api/accounts` 是选择器兼容接口；新增 `/api/metadata/{kind}` 是 Task35 管理基础接口。

## 6. Transactions

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/transaction-defaults` | yes | optional | transitional | `transaction.HandleGetTransactionDefault` | 读取当前用户在当前账本下的快捷记账默认值，自动剔除已归档分类、账户和标签。 |
| GET | `/api/transactions/` | yes | optional | transitional | `transaction.HandleList` | 账单列表，支持筛选。 |
| POST | `/api/transactions/` | yes | optional | transitional | `transaction.HandleCreate` | 创建普通收入/支出。 |
| POST | `/api/transactions/batch-tag` | yes | optional | transitional | `transaction.HandleBatchTag` | 批量打标签。 |
| GET | `/api/transactions/{id}` | yes | optional | transitional | `transaction.HandleGetByID` | 获取账单详情，必须遵守可见性。 |
| PATCH | `/api/transactions/{id}` | yes | optional | transitional | `transaction.HandleUpdate` | 更新账单，需校验编辑权限和审计。 |
| DELETE | `/api/transactions/{id}` | yes | optional | transitional | `transaction.HandleDelete` | 软删除账单，需审计。 |

## 7. Import

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/transactions/import/parse` | yes | optional | transitional | `transaction.HandleParseCSV` | 解析 CSV 文件。 |
| POST | `/api/transactions/import/analyze` | yes | optional | transitional | `transaction.HandleAnalyzeImport` | 预览、匹配规则和重复检测。 |
| POST | `/api/transactions/import/commit` | yes | optional | transitional | `transaction.HandleCommitImport` | 提交导入批次。 |
| POST | `/api/import-rules/` | yes | optional | transitional | `transaction.HandleCreateImportRule` | 创建导入规则。 |
| GET | `/api/import-rules/` | yes | optional | transitional | `transaction.HandleListImportRules` | 列出导入规则。 |
| DELETE | `/api/import-rules/{id}` | yes | optional | transitional | `transaction.HandleDeleteImportRule` | 删除导入规则。 |

## 8. Templates 与周期账单

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/transaction-templates/` | yes | optional | transitional | `transaction.HandleCreateTemplate` | 创建交易模板。 |
| GET | `/api/transaction-templates/` | yes | optional | transitional | `transaction.HandleListTemplates` | 模板列表，默认仅未归档；`include_archived=true` 返回管理列表。 |
| GET | `/api/transaction-templates/{id}` | yes | optional | transitional | `transaction.HandleGetTemplate` | 模板详情。 |
| PUT | `/api/transaction-templates/{id}` | yes | optional | transitional | `transaction.HandleUpdateTemplate` | 更新模板。 |
| POST | `/api/transaction-templates/{id}/archive` | yes | optional | transitional | `transaction.HandleArchiveTemplate` | 归档模板，不再出现在快捷填入中。 |
| POST | `/api/transaction-templates/{id}/restore` | yes | optional | transitional | `transaction.HandleRestoreTemplate` | 恢复已归档模板。 |
| DELETE | `/api/transaction-templates/{id}` | yes | optional | deprecated | `transaction.HandleDeleteTemplate` | 历史兼容入口，当前等同软归档。 |
| POST | `/api/recurring-rules/` | yes | optional | transitional | `transaction.HandleCreateRecurringRule` | 创建周期规则。 |
| GET | `/api/recurring-rules/` | yes | optional | transitional | `transaction.HandleListRecurringRules` | 周期规则列表。 |
| DELETE | `/api/recurring-rules/{id}` | yes | optional | transitional | `transaction.HandleDeleteRecurringRule` | 删除周期规则。 |
| GET | `/api/recurring-reminders/` | yes | optional | transitional | `transaction.HandleListRecurringReminders` | 待确认周期提醒。 |
| POST | `/api/recurring-reminders/{id}/confirm` | yes | optional | transitional | `transaction.HandleConfirmReminder` | 确认提醒并生成账单。 |
| POST | `/api/recurring-reminders/{id}/ignore` | yes | optional | transitional | `transaction.HandleIgnoreReminder` | 忽略提醒。 |

## 9. Shared Expenses

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/shared-expenses/` | yes | optional | transitional | `transaction.HandleCreateSharedExpense` | 创建共同支出并写入 splits。 |
| GET | `/api/shared-expenses/{id}` | yes | optional | transitional | `transaction.HandleGetSharedExpenseByID` | 共同支出详情。 |
| PATCH | `/api/shared-expenses/{id}` | yes | optional | transitional | `transaction.HandleUpdateSharedExpense` | 更新共同支出。 |

## 10. Settlements

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/settlements/balance` | yes | optional | transitional | `settlement.HandleGetBalance` | 获取共同支出轧差。 |
| GET | `/api/settlements/` | yes | optional | transitional | `settlement.HandleList` | 结算记录列表。 |
| POST | `/api/settlements/` | yes | optional | transitional | `settlement.HandleCreate` | 创建结算记录和对应 transaction。 |

## 11. Safety、Backup 与 Export

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/admin/diagnostics` | yes | required | stable | `safety.HandleDiagnostics` | Owner-only 脱敏系统诊断，返回环境、数据库、schema、目录可写性、Cookie 策略、最近备份和审计动作计数。 |
| POST | `/api/admin/backup` | yes | optional | transitional | `safety.HandleManualBackup` | 手动备份。v1.1 前需确认角色要求。 |
| POST | `/api/admin/restore` | yes | optional | transitional | `safety.HandleRestoreBackup` | 恢复备份，高风险。v1.1 前需确认角色要求和二次确认。 |
| GET | `/api/admin/backups` | yes | optional | transitional | `safety.HandleGetBackups` | 备份列表。 |
| GET | `/api/admin/backups/{filename}` | yes | optional | transitional | `safety.HandleDownloadBackup` | 下载备份文件。 |
| GET | `/api/admin/backups/*` | yes | optional | transitional | `safety.HandleDownloadBackup` | 兼容带路径的备份下载。 |
| GET | `/api/export/transactions.csv` | yes | optional | transitional | `safety.HandleExportCSV` | 导出当前账本 CSV。 |
| GET | `/api/export/full.json` | yes | optional | transitional | `safety.HandleExportJSON` | 导出当前账本 JSON。 |

## 12. Reports 与 Dashboard

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/reports/monthly-summary` | yes | optional | transitional | `reports.HandleGetMonthlySummary` | 月度汇总。 |
| GET | `/api/reports/category-summary` | yes | optional | transitional | `reports.HandleGetCategorySummary` | 分类汇总。 |
| GET | `/api/reports/tag-summary` | yes | optional | transitional | `reports.HandleGetTagSummary` | 标签汇总。 |
| GET | `/api/reports/member-summary` | yes | optional | transitional | `reports.HandleGetMemberSummary` | 成员汇总。 |
| GET | `/api/dashboard` | yes | optional | transitional | `dashboard.HandleGetDashboard` | 首页聚合数据。 |

## 13. Attachments

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/attachments` | yes | optional | transitional | `transaction.HandleUploadAttachment` | 上传附件，返回历史兼容路径 `/uploads/{filename}`。 |
| GET | `/api/attachments/{filename}` | yes | required | stable | `transaction.HandleGetAttachment` | 受保护附件读取，根据关联账单可见性校验。 |
| GET | `/uploads/*` | no | none | disabled | router guard | 裸静态附件路径已关闭，必须通过 `/api/attachments/{filename}` 访问。 |

## 14. Task34 后续治理清单

1. Task34.2 新增 `docs/api/openapi.yaml`，至少覆盖本清单中核心路径。
2. Task34.3 已新增 `docs/api/API_CONVENTIONS.md`，后续代码治理必须按该文件执行。
3. Foundation 冻结前，业务写接口应从 `optional` 收紧为 `required` 或明确保留兼容窗口。
4. 统一 handler 使用 `response.WriteError`，避免手写 `response.Error` 和 `http.Error` 返回不一致。
5. Task39 已关闭裸 `/uploads/*`，后续前端展示附件时应使用 `/api/attachments/{filename}`，不要直接请求历史兼容路径。
