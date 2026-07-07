# 技术：短中期模块架构切片

状态：建议采纳  
适用阶段：Foundation before v1.1 之后

## 1. 目标

本文把 v1.1 和 v1.2 产品模块映射为后端、前端、数据库和测试切片，避免后续实现时把业务逻辑散落在 handler 或页面组件中。

## 2. 后端模块增量

建议目录：

```text
internal/preference
internal/template
internal/recurring
internal/importer
internal/importer/parser
internal/importer/dedupe
internal/importer/rule
```

既有模块增强：

- `internal/transaction`：复制、模板生成真实账单、导入落库。
- `internal/category`：归档、排序、恢复。
- `internal/tag`：归档、自动补全数据源。
- `internal/account`：归档、排序。
- `internal/settlement`：解释性 DTO、复制文案数据源。
- `internal/audit`：周期确认、导入提交、结算、归档操作。

## 3. 数据模型建议

### 3.1 v1.1

建议新增或扩展：

```text
user_preferences
- id
- ledger_id
- user_id
- key
- value_json
- updated_at

transaction_templates
- id
- ledger_id
- owner_user_id
- name
- payload_json
- status
- created_at
- updated_at

recurring_rules
- id
- ledger_id
- owner_user_id
- name
- frequency
- interval
- next_run_at
- payload_json
- status
- created_at
- updated_at

recurring_instances
- id
- ledger_id
- rule_id
- due_at
- status
- generated_transaction_id
- created_at
- updated_at
```

说明：

- `payload_json` 存放生成账单所需快照，但生成真实账单时必须重新校验权限和元数据状态。
- 周期实例只表示待确认状态，不进入统计。
- 已使用的分类、标签、账户只能归档，不物理删除。

### 3.2 v1.2

建议新增：

```text
import_batches
- id
- ledger_id
- source_type
- filename
- status
- total_rows
- imported_rows
- skipped_rows
- failed_rows
- created_by_user_id
- created_at
- committed_at

import_rows
- id
- batch_id
- row_number
- raw_json
- normalized_json
- import_hash
- duplicate_status
- row_status
- error_message
- generated_transaction_id

import_rules
- id
- ledger_id
- name
- match_type
- pattern
- priority
- result_json
- status
- created_at
- updated_at
```

说明：v1.2 导入模块的最终字段、索引、状态机、DTO、API 迁移和回滚策略以 `docs/tech/20-v1.2-import-implementation-contract.md` 为准。本文保留高层切片，不替代 Task47-Task49 的实施契约。

## 4. API 切片

### 4.1 v1.1 API

建议端点：

```text
GET    /api/preferences/transaction-defaults
PUT    /api/preferences/transaction-defaults

POST   /api/transactions/{id}/copy-preview
POST   /api/transaction-templates
GET    /api/transaction-templates
POST   /api/transaction-templates/{id}/instantiate
PATCH  /api/transaction-templates/{id}
POST   /api/transaction-templates/{id}/archive

POST   /api/recurring-rules
GET    /api/recurring-rules
GET    /api/recurring-instances/pending
POST   /api/recurring-instances/{id}/confirm
POST   /api/recurring-instances/{id}/skip

GET    /api/settlements/explanation
GET    /api/settlements/copy-text
```

### 4.2 v1.2 API

建议端点：

```text
POST   /api/imports/preview
GET    /api/imports/{batch_id}
PATCH  /api/imports/{batch_id}/rows/{row_id}
POST   /api/imports/{batch_id}/commit

GET    /api/import-rules
POST   /api/import-rules
PATCH  /api/import-rules/{id}
POST   /api/import-rules/{id}/archive
```

说明：当前代码中仍存在早期 `/api/transactions/import/*` 接口。v1.2 新开发以 `/api/imports/*` 为目标契约，旧接口只作为 transitional 兼容入口，不应继续承载新增业务逻辑。

## 5. 服务层边界

必须放在 service 层：

- 复制账单字段筛选和权限校验。
- 模板实例化校验。
- 周期实例确认入账。
- 分类/标签/账户归档约束。
- 结算解释数据计算。
- CSV 解析后的标准化。
- import_hash 生成和去重判断。
- 导入批次事务提交。

不得放在 handler 或前端：

- 金额分摊计算。
- 结算净额计算。
- 是否可见、可编辑、可导入的权限判断。
- 重复导入最终判断。

## 6. 前端模块切片

建议新增：

```text
frontend/src/features/preferences
frontend/src/features/templates
frontend/src/features/recurring
frontend/src/features/imports
```

既有增强：

- `features/transactions`：快捷表单、复制入口、保存并继续记。
- `features/settings`：分类、标签、账户、模板、周期账单、导入规则。
- `features/settlement`：解释面板、复制文案。

Query key 必须包含 ledger id，避免跨账本缓存污染。

## 7. 测试切片

后端必须覆盖：

- 复制账单不修改原账单。
- 模板不进入统计和结算。
- 周期实例确认后才生成账单。
- 归档元数据不影响历史账单展示。
- 结算解释字段与净额一致。
- 重复 CSV 不重复导入。
- 导入事务失败回滚。

前端必须覆盖：

- 快捷表单默认值。
- 保存并继续记字段保留规则。
- 归档项不出现在新建选择器。
- 导入预览错误提示。
- 移动端关键页面无横向滚动。

## 8. 回滚策略

- v1.1 新表可通过 feature flag 隐藏入口，不删除历史数据。
- 周期账单若有问题，关闭 pending 生成，不影响已确认账单。
- 导入若有问题，关闭 commit 入口，保留 preview 只读能力。
- 不允许通过修改历史 migration 回滚，必须新增 migration 修正。
