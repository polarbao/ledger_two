# Task53 分类、标签与自动归类 Fixture 规格

状态：Task53 匿名 Fixture 基线；Task53.1 默认 profile/migration 用例已落地，不包含真实账单<br>
创建日期：2026-07-16<br>
适用：Task53 classifier、migration、API、前端状态和浏览器验收

## 1. Rules

1. Fixture 全部匿名构造，不使用真实用户、商户订单号、账单文件、金额明细或备注。
2. 金额使用整数分，时间使用固定 ISO8601。
3. 每个 Fixture 显式给出原始字段、规范化字段、候选、最终结果和解释。
4. 同一输入多次运行结果必须稳定。
5. 账本 A 的规则、分类和标签不得影响账本 B。

## 2. Metadata fixture

### Ledger A

| ID | kind | system_key | name | state |
|---|---|---|---|---|
| `cat_a_food` | category/expense | `expense_food` | 餐饮 | active |
| `cat_a_transport` | category/expense | `expense_transport` | 交通 | active |
| `cat_a_other_expense` | category/expense | `expense_other` | 其他支出 | active |
| `cat_a_salary` | category/income | `income_salary` | 工资 | active |
| `cat_a_refund` | category/income | `income_refund` | 退款 | active |
| `cat_a_other_income` | category/income | `income_other` | 其他收入 | active |
| `tag_a_takeout` | tag | `tag_takeout` | 外卖 | active |
| `tag_a_commute` | tag | `tag_commute` | 通勤 | active |
| `tag_a_archived` | tag | null | 旧标签 | archived |

Ledger B 使用不同 ID 和同名对象，验证账本隔离。

## 3. Rule fixture

| ID | origin | match | priority | mode | result |
|---|---|---|---:|---|---|
| `rule_a_didi_manual` | manual | merchant_contains=滴滴 | 100 | auto | 交通 + 通勤 |
| `rule_a_coffee_learned` | learned | merchant_equals=星河咖啡 | 500 | auto | 餐饮 |
| `rule_a_food_broad` | manual | description_contains=餐 | 800 | suggest | 餐饮 |
| `rule_a_archived_tag` | manual | merchant_contains=旧店 | 100 | auto | 旧标签 |
| `rule_a_conflict_1` | manual | merchant_equals=冲突商户 | 10 | auto | 餐饮 |
| `rule_a_conflict_2` | manual | merchant_equals=冲突商户 | 10 | auto | 交通 |

## 4. Row fixture

| ID | merchant/title | expected status | expected category/tags | reason |
|---|---|---|---|---|
| `CT-R01` | 滴滴出行 | auto_selected | 交通 + 通勤 | 用户规则 |
| `CT-R02` | 星河咖啡 | auto_selected | 餐饮 | 学习规则精确匹配 |
| `CT-R03` | 美团外卖订单 | suggested | 餐饮 + 外卖建议 | built-in 只建议 |
| `CT-R04` | 未知商户甲 | fallback | 其他支出 | 无可靠候选 |
| `CT-R05` | 工资发放 | suggested | 工资建议 | income built-in |
| `CT-R06` | 原路退款 | suggested | 退款建议 | refund direction |
| `CT-R07` | 冲突商户 | conflict | 无自动分类 | 同级规则冲突 |
| `CT-R08` | 旧店 | fallback | 其他支出 | 规则引用 archived tag，整体失效 |
| `CT-R09` | merchant empty | fallback | 其他支出 | 不允许学习 |
| `CT-R10` | duplicate row | skipped | 不分类 | duplicate 不执行 classifier |
| `CT-R11` | invalid amount | unresolved | 不分类 | invalid 不执行 classifier |
| `CT-R12` | same as R01 in ledger B | fallback | B 的其他支出 | A 规则不可泄漏 |

## 5. Normalization fixture

以下值应规范化为相同 key：

```text
" 星河咖啡 "
"星河咖啡"
"星河咖啡　"
```

ASCII 示例应大小写不敏感：

```text
"STAR CAFE"
"star cafe"
```

不得把以下不同主体错误合并：

```text
"Apple Store"
"Apple Music"
```

## 6. Tag limit fixture

1. 两条高置信规则分别给出 5 个不同标签，总计 10 个。
2. resolver 返回 `TAG_LIMIT_EXCEEDED`/conflict，不截断前 8 个。
3. 用户手工调整到 8 个后可保存。
4. 重复标签去重后按规则顺序稳定展示。

## 7. Default profile fixture

| Case | Existing state | Expected preview/apply |
|---|---|---|
| `CT-P01` | 空账本 | 创建完整 basic_cn_v1 |
| `CT-P02` | 已有同名同类型“餐饮” | 提示复用/新建/跳过，不自动绑定 system_key |
| `CT-P03` | 已有收入“餐饮” | 不冲突，仍可创建支出“餐饮” |
| `CT-P04` | 已有 tag“旅行” | 提示复用或跳过 |
| `CT-P05` | 已应用 v1 | 再次 apply 为 no-op |
| `CT-P06` | 中途注入失败 | 事务回滚，version 不更新 |
| `CT-P07` | 选择 empty | 分类/标签数量均为 0 |

## 8. Learning fixture

1. R03 手工选择餐饮 + 外卖并勾选记住。
2. 创建 `origin=learned`、`merchant_equals`、当前 ledger/source 范围的规则。
3. 再次 preview 相同规范化商户时变为 auto_selected。
4. 不勾选记住时只更新当前行。
5. 存在显式用户规则冲突时，本行保存成功、learn 返回冲突。
6. 归档 learned rule 后恢复为 built-in suggestion/fallback。

## 9. API snapshots

expected JSON：

```text
docs/fixtures/category-tag/expected/auto-selected.json
docs/fixtures/category-tag/expected/suggested.json
docs/fixtures/category-tag/expected/conflict.json
docs/fixtures/category-tag/expected/profile-preview.json
```

4 个 JSON 已生成并纳入 OpenAPI/Tech 静态校验。`profile-preview.json` 已与 `basic_cn_v1` 的 27 个冻结 system key、图标和颜色对齐；其余分类 expected 将从 Task53.2 起作为 pure classifier 与 API 快照依据。DTO 变化必须联合更新 OpenAPI 和 expected。

## 10. Acceptance matrix

| ID | Layer | Expected |
|---|---|---|
| CT-CL-001 | pure | 排序和冲突确定性 |
| CT-CL-002 | pure | 标签并集与上限 |
| CT-API-001 | API | high 写 selected，suggest 写 suggested |
| CT-API-002 | API | manual/bulk 不被 reclassify 覆盖 |
| CT-API-003 | API | learn 只作用当前账本 |
| CT-API-004 | API | accept_suggestions 可逐行接受不同建议，且不创建学习规则 |
| CT-DB-001 | migration | 21 -> 22 数量与金额守恒 |
| CT-DB-002 | migration | 历史规则保持 suggest |
| CT-UI-001 | UI | 状态筛选和原因可访问 |
| CT-UI-002 | UI | 批量接受不触发 commit |
| CT-UI-003 | UI | 375px 无横向滚动 |
| CT-ROLL-001 | rollback | graded -> suggest 行为降级 |
