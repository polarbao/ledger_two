# API Inventory

状态：Task50.6 正式契约已冻结；Task53.1-Task53.4C 已落地，下一阶段为 Task53U
来源：`backend/internal/http/router/router.go`  
当前实现基路径：`/api`  
目标版本基路径：`/api/v1`，尚未实现 alias  
更新时间：2026-07-20

> Task53.1-Task53.4C 已实现 schema 22、默认 profile、确定性 preview/reclassify、bulk-adjust、learn、规则生命周期、stale/reference/committed-hit 指标与兜底分类原子替代。Task53U 页面和 Task53.5 隔离 staging 尚未执行。

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
- `optional`：仅作为历史文档标签保留；当前生产账本内路由已无 optional/fallback 入口。
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
| GET | `/api/healthz` | no | none | internal | inline | 健康检查，返回服务、数据库、应用版本、schema version、deployment channel、XLSX 开关和 `import_classification_mode`。 |
| GET | `/api/init/status` | no | none | stable | `init.HandleStatus` | 初始化状态。 |
| POST | `/api/init/setup` | no | none | stable | `init.HandleSetup` | 原子初始化系统、用户、初始账本、账户与 `basic_cn_v1` 分类/标签。 |

## 3. Auth

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/auth/login` | no | none | stable | `auth.HandleLogin` | 登录并写入 Cookie token。 |
| POST | `/api/auth/logout` | no | none | stable | `auth.HandleLogout` | 清理 Cookie token。 |
| GET | `/api/auth/me` | yes | none | stable | `auth.HandleMe` | 当前用户与 `instance_admin` 能力；不返回或推断当前账本。 |

## 4. Ledgers 与成员

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/ledgers/` | yes | none | stable | `ledger.CreateLedger` | 原子创建 active/version 1 账本、唯一 Owner、审计与 `metadata_profile`；省略 profile 默认 `basic_cn_v1`，可选 `empty`。返回 201 与 ETag。 |
| GET | `/api/ledgers/` | yes | none | stable | `ledger.ListUserLedgers` | 按 `status=active/archived/all` 列表；默认 active。 |
| GET | `/api/ledgers/{id}` | yes | path | stable | `ledger.GetLedger` | 成员读取 active/archived 详情；返回 ETag。 |
| PATCH | `/api/ledgers/{id}` | yes | path | stable | `ledger.RenameLedger` | active Owner + If-Match 重命名。 |
| GET | `/api/ledgers/{id}/archive-preflight` | yes | path | stable | `ledger.GetArchivePreflight` | active Owner 只读预检未结清净额和未过期 ready 批次；不写审计。 |
| POST | `/api/ledgers/{id}/archive` | yes | path | stable | `ledger.ArchiveLedger` | active Owner + If-Match 归档；ready 阻断，未结清需显式确认。 |
| POST | `/api/ledgers/{id}/restore` | yes | path | stable | `ledger.RestoreLedger` | archived Owner + If-Match 恢复。 |
| GET | `/api/ledgers/{id}/members` | yes | path | stable | `ledger.GetLedgerMembers` | active/archived 成员读取；返回 ledger + members、joined_at 与 ETag。 |
| POST | `/api/ledgers/{id}/members` | yes | path | stable | `ledger.AddMember` | active Owner + If-Match 添加第二成员；必须确认历史可见性，返回 201、成员快照与新 ETag。 |
| PATCH | `/api/ledgers/{id}/members/{userId}` | yes | path | stable | `ledger.UpdateMemberRole` | active Owner + If-Match 在 editor/viewer 间调整；通用接口不接受 owner。 |
| PUT | `/api/ledgers/{id}/members/{userId}` | yes | path | deprecated | `ledger.UpdateMemberRole` | Task50.3B 兼容旧客户端；与 PATCH 调用同一 handler/service，后续删除需独立评审。 |
| DELETE | `/api/ledgers/{id}/members/{userId}` | yes | path | stable | `ledger.RemoveMember` | active Owner + If-Match 移除非 Owner；历史账务对象不改写。 |
| POST | `/api/ledgers/{id}/members/{userId}/transfer-owner` | yes | path | stable | `ledger.TransferOwner` | 当前 Owner 确认权限变化后原子移交；原 Owner 成为 Editor，只写一个审计事件。 |
| POST | `/api/ledgers/{id}/leave` | yes | path | stable | `ledger.LeaveLedger` | Editor/Viewer + If-Match 主动离开并返回提交后的 ETag；Owner 必须先移交。 |

## 5. Metadata 基础查询

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/categories` | yes | required | transitional | `transaction.HandleListCategories` | 列出当前账本分类；默认仅未归档，`include_archived=true` 用于历史账单展示归档分类名称。 |
| GET | `/api/accounts` | yes | required | transitional | `transaction.HandleListAccounts` | 列出当前账本支付账户。 |
| GET | `/api/metadata/{kind}/` | yes | required | transitional | `metadata.List` | 元数据列表，kind 为 categories/tags/accounts，支持 include_archived，返回 `sort_order`、`usage_count` 和当前账本 active rule 的 `rule_reference_count`；分类/标签可返回 `system_key`。 |
| POST | `/api/metadata/{kind}/` | yes | required | transitional | `metadata.Create` | 创建分类、标签或账户，仅 owner。 |
| POST | `/api/metadata/{kind}/reorder` | yes | required | transitional | `metadata.Reorder` | 调整分类、标签或账户排序，仅 owner。 |
| PATCH | `/api/metadata/{kind}/{id}` | yes | required | transitional | `metadata.Update` | 更新分类、标签或账户，仅 owner。 |
| POST | `/api/metadata/{kind}/{id}/archive` | yes | required | transitional | `metadata.Archive` | 归档分类、标签或账户，仅 owner；归档 expense_other/income_other 必须提交同账本、active、同类型且无 system_key 的 replacement_category_id，转移与归档同事务。 |
| POST | `/api/metadata/{kind}/{id}/restore` | yes | required | transitional | `metadata.Restore` | 恢复归档分类、标签或账户，仅 owner。 |
| GET | `/api/metadata/default-profile` | yes | required | transitional | `metadata.GetDefaultProfile` | 读取 `basic_cn_v1` 或 `empty` 定义及当前账本解析结果；只读。 |
| POST | `/api/metadata/default-profile/preview` | yes | required | transitional | `metadata.PreviewDefaultProfile` | 预览创建、已存在和同名冲突，不写元数据或绑定 `system_key`。 |
| POST | `/api/metadata/default-profile/apply` | yes | required | transitional | `metadata.ApplyDefaultProfile` | active Owner 显式解决冲突后原子应用；幂等更新 profile version 并写审计。 |

说明：旧 `/api/categories`、`/api/accounts` 是选择器兼容接口；新增 `/api/metadata/{kind}` 是 Task35 管理基础接口。

## 6. Transactions

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/transaction-defaults` | yes | required | transitional | `transaction.HandleGetTransactionDefault` | 读取当前用户在当前账本下的快捷记账默认值，自动剔除已归档分类、账户和标签。 |
| GET | `/api/transactions/` | yes | required | transitional | `transaction.HandleList` | 账单列表，支持筛选。 |
| POST | `/api/transactions/` | yes | required | transitional | `transaction.HandleCreate` | 创建普通收入/支出。 |
| POST | `/api/transactions/batch-tag` | yes | required | transitional | `transaction.HandleBatchTag` | 批量打标签。 |
| GET | `/api/transactions/{id}` | yes | required | transitional | `transaction.HandleGetByID` | 获取账单详情，必须遵守可见性。 |
| PATCH | `/api/transactions/{id}` | yes | required | transitional | `transaction.HandleUpdate` | 更新账单，需校验编辑权限和审计。 |
| DELETE | `/api/transactions/{id}` | yes | required | transitional | `transaction.HandleDelete` | 软删除账单，需审计。 |

## 7. Import

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/transactions/import/parse` | yes | required | transitional | `transaction.HandleParseCSV` | 解析 CSV 文件。 |
| POST | `/api/transactions/import/analyze` | yes | required | transitional | `transaction.HandleAnalyzeImport` | 预览、匹配规则和重复检测。 |
| POST | `/api/transactions/import/commit` | yes | required | transitional | `transaction.HandleCommitImport` | 提交导入批次。 |
| POST | `/api/imports/preview` | yes | required | stable | `importer.HandlePreview` | Owner 上传 CSV/受支持 XLSX 并生成预览批次，不写正式账单；Task53 模式非 off 时持久化分类解释和 summary。 |
| GET | `/api/imports/{batchID}` | yes | required | stable | `importer.HandleGetBatch` | Owner 读取导入批次、行级预览、classification 快照和服务端重算 summary。 |
| PATCH | `/api/imports/{batchID}/rows/{rowID}` | yes | required | stable | `importer.HandleUpdateRow` | v1.2 Owner 调整导入行状态、目标类型、分类、账户、标签和可见性。 |
| POST | `/api/imports/{batchID}/reclassify` | yes | required | stable | `importer.HandleReclassify` | Owner 对 ready/未过期批次重算 eligible 非 manual/bulk 行；默认 dry-run，执行写脱敏审计但不创建 transaction。 |
| POST | `/api/imports/{batchID}/rows/bulk-adjust` | yes | required | stable | `importer.HandleBulkAdjust` | Owner 对 ready/未过期批次按持久化建议或完整显式值批量调整；部分成功返回行级结果，单事务写一条脱敏审计，不创建 transaction/learned rule。 |
| POST | `/api/imports/{batchID}/rows/{rowID}/learn` | yes | required | stable | `importer.HandleLearnMerchant` | Owner 从已另行保存的 manual/bulk 行读取最终分类与完整标签，按账本/来源范围/规范化商户 UUIDv5 幂等创建、更新或恢复 learned rule；不学习账户、可见性或账单原文。 |
| POST | `/api/imports/{batchID}/commit` | yes | required | stable | `importer.HandleCommit` | v1.2 Owner 提交 ready 批次，事务写入正式账单和导入去重映射。 |
| POST | `/api/imports/{batchID}/discard` | yes | required | stable | `importer.HandleDiscardBatch` | Owner 显式放弃 ready 批次；收敛为 expired，保留行/hash，不创建 transaction。 |
| POST | `/api/import-rules/` | yes | required | stable | `importer.HandleCreateRule` | Owner 创建 `origin=manual` 规则，可设置来源范围与 auto/suggest，confidence 由服务端固定 high。 |
| GET | `/api/import-rules/` | yes | required | stable | `importer.HandleListRules` | Owner 列出当前账本 manual/learned 规则及 origin/source/apply/confidence、stale 引用和 committed/imported 命中指标，支持 `status=active/archived/all`。 |
| PATCH | `/api/import-rules/{ruleID}` | yes | required | stable | `importer.HandleUpdateRule` | Owner 更新规则；learned 的来源、merchant_equals 和规范化 pattern 不可修改。 |
| POST | `/api/import-rules/{ruleID}/archive` | yes | required | stable | `importer.HandleArchiveRule` | v1.2 Owner 归档导入规则。 |
| POST | `/api/import-rules/{ruleID}/restore` | yes | required | stable | `importer.HandleRestoreRule` | Owner 恢复导入规则；任何 stale rule 均拒绝恢复，learned rule 与同范围 active manual 商户精确规则冲突时同样拒绝。 |
| DELETE | `/api/import-rules/{ruleID}` | yes | required | transitional | `importer.HandleArchiveRule` | Owner 兼容旧删除入口，实际执行归档。 |

## 8. Templates 与周期账单

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/transaction-templates/` | yes | required | transitional | `transaction.HandleCreateTemplate` | 创建交易模板。 |
| GET | `/api/transaction-templates/` | yes | required | transitional | `transaction.HandleListTemplates` | 模板列表，默认仅未归档；`include_archived=true` 返回管理列表。 |
| GET | `/api/transaction-templates/{id}` | yes | required | transitional | `transaction.HandleGetTemplate` | 模板详情。 |
| PUT | `/api/transaction-templates/{id}` | yes | required | transitional | `transaction.HandleUpdateTemplate` | 更新模板。 |
| POST | `/api/transaction-templates/{id}/archive` | yes | required | transitional | `transaction.HandleArchiveTemplate` | 归档模板，不再出现在快捷填入中。 |
| POST | `/api/transaction-templates/{id}/restore` | yes | required | transitional | `transaction.HandleRestoreTemplate` | 恢复已归档模板。 |
| DELETE | `/api/transaction-templates/{id}` | yes | required | deprecated | `transaction.HandleDeleteTemplate` | 历史兼容入口，当前等同软归档。 |
| POST | `/api/recurring-rules/` | yes | required | transitional | `transaction.HandleCreateRecurringRule` | 创建周期规则。 |
| GET | `/api/recurring-rules/` | yes | required | transitional | `transaction.HandleListRecurringRules` | 周期规则列表。 |
| DELETE | `/api/recurring-rules/{id}` | yes | required | transitional | `transaction.HandleDeleteRecurringRule` | 删除周期规则。 |
| GET | `/api/recurring-reminders/` | yes | required | transitional | `transaction.HandleListRecurringReminders` | 待确认周期提醒。 |
| POST | `/api/recurring-reminders/{id}/confirm` | yes | required | transitional | `transaction.HandleConfirmReminder` | 确认提醒并生成账单。 |
| POST | `/api/recurring-reminders/{id}/skip` | yes | required | transitional | `transaction.HandleIgnoreReminder` | 跳过本期待确认提醒，不生成真实账单。 |
| POST | `/api/recurring-reminders/{id}/ignore` | yes | required | deprecated | `transaction.HandleIgnoreReminder` | 历史兼容入口，当前等同跳过本期。 |

## 9. Shared Expenses

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/shared-expenses/` | yes | required | transitional | `transaction.HandleCreateSharedExpense` | 创建共同支出并写入 splits。 |
| GET | `/api/shared-expenses/{id}` | yes | required | transitional | `transaction.HandleGetSharedExpenseByID` | 共同支出详情。 |
| PATCH | `/api/shared-expenses/{id}` | yes | required | transitional | `transaction.HandleUpdateSharedExpense` | 更新共同支出。 |

## 10. Settlements

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/settlements/balance` | yes | required | transitional | `settlement.HandleGetBalance` | 获取共同支出轧差，`user_balances` 返回 `paid_cents`、`share_cents`、`raw_net_cents`、`settlement_net_cents`、`final_net_cents` 和兼容字段 `net_cents`。 |
| GET | `/api/settlements/` | yes | required | transitional | `settlement.HandleList` | 结算记录列表。 |
| POST | `/api/settlements/` | yes | required | transitional | `settlement.HandleCreate` | 创建结算记录和对应 transaction。 |

## 11. Safety、Backup 与 Export

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/admin/diagnostics` | yes | none | stable | `safety.HandleDiagnostics` | 独立实例管理员读取脱敏诊断；忽略账本 header，写 `system_diagnostics` instance audit。 |
| POST | `/api/admin/backup` | yes | none | stable | `safety.HandleManualBackup` | 独立实例管理员创建整库物理备份；响应为完整 BackupInfo，文件与 `manual_database_backup` audit 同成败。 |
| POST | `/api/admin/restore` | yes | none | stable | `safety.HandleRestoreBackup` | 独立实例管理员创建前置备份并返回停机恢复指引，不在 HTTP 请求中替换运行数据库。 |
| GET | `/api/admin/backups` | yes | none | stable | `safety.HandleGetBackups` | 返回受管理目录内的安全相对 key 并写 `list_database_backups` instance audit。 |
| GET | `/api/admin/backups/{filename}` | yes | none | stable | `safety.HandleDownloadBackup` | 下载 basename key；校验规范路径、`.db` 扩展名和目录边界并写 instance audit。 |
| GET | `/api/admin/backups/*` | yes | none | stable | `safety.HandleDownloadBackup` | 正式支持列表返回的嵌套相对 key，例如 `manual/*.db`；不消费账本 header。 |
| GET | `/api/export/transactions.csv` | yes | required | stable | `safety.HandleExportCSV` | 导出当前角色可见的当前账本 CSV；历史成员名称仅通过可见账本对象引用解析。 |
| GET | `/api/export/full.json` | yes | required | stable | `safety.HandleExportJSON` | 导出带 manifest 的只读账本数据包；Owner 原始导入行同时受导入权限和交易可见性过滤，Editor 的批次/原始行/规则段为空但保留可见交易来源引用；不含全局用户、app_settings、实例管理员或其他账本数据，且不能替代 SQLite 物理备份。 |

## 12. Reports 与 Dashboard

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| GET | `/api/reports/monthly-summary` | yes | required | transitional | `reports.HandleGetMonthlySummary` | 月度汇总。 |
| GET | `/api/reports/category-summary` | yes | required | transitional | `reports.HandleGetCategorySummary` | 分类汇总。 |
| GET | `/api/reports/tag-summary` | yes | required | transitional | `reports.HandleGetTagSummary` | 标签汇总。 |
| GET | `/api/reports/member-summary` | yes | required | transitional | `reports.HandleGetMemberSummary` | 成员汇总。 |
| GET | `/api/dashboard` | yes | required | transitional | `dashboard.HandleGetDashboard` | 首页聚合数据。 |

## 13. Attachments

| Method | Path | Auth | Ledger | Stability | Handler | 说明 |
|---|---|---:|---|---|---|---|
| POST | `/api/attachments` | yes | required | transitional | `transaction.HandleUploadAttachment` | 上传附件，返回历史兼容路径 `/uploads/{filename}`。 |
| GET | `/api/attachments/{filename}` | yes | required | stable | `transaction.HandleGetAttachment` | 受保护附件读取，根据关联账单可见性校验。 |
| GET | `/uploads/*` | no | none | disabled | router guard | 裸静态附件路径已关闭，必须通过 `/api/attachments/{filename}` 访问。 |

## 14. 后续治理清单

1. `docs/api/openapi.yaml` 已在 Task50.6 提升为 v1.3.0-rc 正式契约；ledger path、显式 header、实例运维、v1.2 导入和 Task50 导出均需与本清单同步。
2. `docs/api/API规范.md` 继续作为错误包络、金额和命名约束；新增接口不得绕过。
3. Task50.3C 已完成生产账本路由 `required` 收口；后续新增账本 API 不得重新引入 optional/fallback。
4. 统一 handler 使用 `response.WriteError`，避免手写 `response.Error` 和 `http.Error` 返回不一致。
5. Task39 已关闭裸 `/uploads/*`，后续前端展示附件时应使用 `/api/attachments/{filename}`，不要直接请求历史兼容路径。
6. OpenAPI 当前保留与真实 chi router 一致的部分尾斜杠路径；Redocly 校验需跳过 `no-path-trailing-slash` 风格规则，其他结构错误必须为 0。
