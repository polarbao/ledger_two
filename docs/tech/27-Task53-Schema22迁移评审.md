# Migration Review：Task53 schema 22 分类与标签智能化

状态：Task53.1 migration 022 已实现并通过临时数据库、WSL2 与 NAS 独立 staging 评审；production 未部署<br>
创建日期：2026-07-16<br>
目标升级：schema 21 -> schema 22<br>
禁止环境：当前 WSL staging、NAS staging、NAS production 和任何真实账单数据库

## 1. Goal

评审 Task53 默认元数据、规则来源/应用模式和预览解释所需的最小数据库增量，确保历史规则行为不变、账务数量金额守恒、应用可向后回退。

## 2. Options

### Option A：只复用现有 JSON 字段

做法：把 origin、apply_mode、confidence 和解释全部塞入 `result_json`/`normalized_json`。

优点：字段少，不新增规则列。

问题：

1. 无法可靠索引和筛选学习规则。
2. 历史 JSON 兼容逻辑复杂。
3. 规则列表、归档和统计需要全表解析 JSON。
4. system_key/idempotency 无法解决。

结论：不采用。

### Option B：新增结构化列，解释保留 JSON

做法：高频筛选/约束使用列，低频解释详情使用 JSON。

优点：

1. origin/status/priority 可索引。
2. 兼容当前 repository 和 SQLite。
3. 解释结构可后续增加 reason 参数。
4. 迁移默认值可保持历史行为。

风险：SQLite `ALTER TABLE` 增加列较多，需要严格副本演练。

结论：采用。

### Option C：新建 classification_rules/classification_events

优点：模型纯净，可完全脱离历史 import_rules。

问题：重复 CRUD、审计和 UI；迁移历史规则成本高；首期形成两套规则引擎。

结论：不采用。

## 3. Proposed migration

实现文件：

```text
backend/migrations/022_add_category_tag_intelligence.sql
```

结构变更以 `docs/tech/26-v1.3-Task53分类标签智能化实施契约.md` 第 9 节为准。SQL 实现阶段必须额外考虑：

1. SQLite 对新增 `CHECK` 的兼容性；若不能无损增加，校验放 service 并用触发器只保护关键枚举。
2. partial unique index 在当前 SQLite 版本的支持情况。
3. 新列默认值不得触发旧行重写或长时间锁表风险。
4. `classification_reason_json` 和 `matched_rule_ids_json` 必须是合法 JSON 文本，但不在 SQL 中使用复杂 JSON CHECK 阻塞旧环境。

## 4. Preflight

升级前必须满足：

1. 当前 schema 精确为 21。
2. `PRAGMA quick_check` 返回 `ok`。
3. 不存在孤立 category/tag/import_rule/import_item。
4. `import_rules.result_json` 全部可解析；不可解析时停止升级并报告规则 ID。
5. `import_items` 的 selected/suggested tag JSON 全部可解析。
6. 每账本 category `(type,name)` 和 tag `name` 继续满足唯一约束。
7. 没有任何既有 `system_key` 字段，因为 migration 尚未应用。

禁止自动修复真实数据。异常只能通过升级前报告和单独向前修复处理。

## 5. Data mapping

历史行映射：

| Table | Field | Historical value |
|---|---|---|
| ledgers | metadata_profile_version | 0 |
| categories/tags | system_key | null |
| import_rules | origin | manual |
| import_rules | source_type | null，代表所有支持来源 |
| import_rules | apply_mode | suggest |
| import_rules | confidence | high |
| import_items | classification_status | unresolved |
| import_items | classification_confidence | none |
| import_items | classification_source | null |
| import_items | reason/rule JSON | `{}` / `[]` |

不得根据名称把历史“餐饮”“其他”自动绑定为 system_key；绑定只能由用户在默认包预览中显式确认复用，且复用仍不夺取用户对象的 system_key。

## 6. Invariants

迁移前后必须一致：

```text
ledger_count
user_count
ledger_member_count
transaction_count_by_ledger
transaction_amount_sum_cents_by_ledger
transaction_split_count_and_sum
settlement_count_and_sum
category_count_by_ledger
tag_count_by_ledger
transaction_tag_count
import_rule_count_by_ledger
import_batch/item/ref_count
import_hash_set
audit_log_count
```

新列只增加描述能力，不修改任何 transaction、split、settlement、import hash 或历史 metadata 值。

## 7. Failure injection

测试副本至少覆盖：

1. 添加列后、创建索引前失败。
2. 创建第一个 unique index 后失败。
3. 历史 result_json 不合法。
4. schema 非 21。
5. quick_check 非 ok。
6. 升级完成但应用启动自检失败。

所有失败都必须证明：原数据库可从一致性备份恢复；不得依赖 goose down 删除生产列。

## 8. Rollback

1. 迁移执行前创建应用/数据库成对备份。
2. migration 022 一旦进入共享环境，不通过 down 删除字段。
3. 应用可回退到忽略新列的旧版本，但旧版本不能创建 Task53 规则。
4. 行为回滚优先使用 `IMPORT_CLASSIFICATION_MODE=suggest|off`。
5. 若数据库结构异常，从迁移前一致性备份恢复，不手工 DROP COLUMN/INDEX 修补生产库。

## 9. Environment order

```text
temporary migration test DB
-> isolated Task53 development DB
-> isolated Task53 staging clone
-> WSL staging maintenance window
-> NAS staging
-> NAS production after explicit approval
```

Task53.1 已只在临时 migration test DB 执行。Task50 schema 21 已进入本机独立 38091 WSL staging，但该数据库仍是 Task53 禁止目标；Task53.5 必须另建隔离 staging，NAS 继续没有部署准入。

## 10. Review gate

Task53.1 实施复核：

1. 当前 `go-sqlite3` SQLite 3.53.2 已实际验证 partial unique index 生效并拒绝同账本重复 `system_key`。
2. migration 使用单个 Goose statement transaction，只新增列和索引；故障注入证明失败后 schema/version 保持 21。
3. schema 21 匿名 Fixture 已覆盖账务金额、split、settlement、metadata、导入数量和 hash 集合守恒。
4. 新 schema 对旧查询保持只增量列兼容；历史应用二进制读取演练保留到 Task53.5 独立 staging，不作为现有 WSL/NAS 升级授权。
5. OpenAPI、router、初始化和新账本 DTO 已同步；导入分类 DTO 仍由 Task53.2-Task53.4 逐步实现。
