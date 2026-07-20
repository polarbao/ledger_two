# Task53.2 Built-in v1 匿名关键词误命中评审

状态：Task53.2 已实现并通过正负例回归；后续新增词条仍需重新评审<br>
日期：2026-07-17<br>
范围：只为没有高优先级用户/学习规则的 eligible 导入行产生 medium 建议

## 1. Frozen minimal set

| Rule key | Direction/type | Match input | Anonymous terms | Category key | Tag keys | Confidence |
|---|---|---|---|---|---|---|
| `builtin_takeout_v1` | expense/expense | merchant or title contains | `外卖` | `expense_food` | `tag_takeout` | medium |
| `builtin_salary_v1` | income/income | title contains | `工资` | `income_salary` | none | medium |
| `builtin_refund_v1` | refund/income | title contains | `退款`、`退回` | `income_refund` | none | medium |

首版仅冻结上述三个匿名、通用词条，以覆盖 CT-R03、CT-R05、CT-R06。品牌名、用户真实商户、订单描述和支付渠道名称不得进入内置表。

## 2. Guardrails

1. built-in 永远不写 `selected_*`，只返回建议；即使唯一命中也不能升级为 high。
2. 必须同时满足方向与目标交易类型，支出词条不得用于收入，收入词条不得用于支出。
3. 用户规则或学习规则已给出有效候选时，built-in 只保留诊断信息，不形成第二个可提交选择。
4. 目标 `system_key` 缺失、归档或跨账本时整条候选失效；不得用名称猜测或自动创建元数据。
5. 多个 built-in 给出不同分类时返回 conflict，不按声明顺序偷偷取第一条。
6. 只做 NFKC、Unicode 小写、首尾 trim 和连续空白折叠；首版不删除数字、品牌后缀或任意标点片段。
7. 默认 profile 为 `empty` 或既有账本未应用基础包时，允许自然降级到 fallback/unresolved。

## 3. Negative cases

| Input | Context | Expected |
|---|---|---|
| `退款失败` | out/expense | 不命中 refund |
| `工资卡还款` | out/expense | 不命中 salary |
| `外卖平台退款` | in/income | 可命中 refund，不命中 takeout |
| `Apple Store` / `Apple Music` | 任意 | 不做噪声裁剪，不合并主体 |
| 空 merchant + 普通 title | out/expense | 无候选，进入 fallback |

## 4. Task53.2 test gate

1. CT-R01 至 CT-R12、Normalization、Tag limit 和上述负例先写 table-driven failing tests。
2. 同一输入循环 100 次输出必须一致；禁止依赖 map 遍历顺序。
3. `IMPORT_CLASSIFICATION_MODE` 默认 `off`；Task53.2 不接入 preview/commit。
4. 新增词条必须先追加匿名正例、负例、目标 system key 和误命中理由，再修改代码。

结论：Task53.2 的 built-in v1 已按冻结范围落地；不需要扩展竞品研究或读取真实账单。后续新增词条必须先补匿名正负 Fixture。
