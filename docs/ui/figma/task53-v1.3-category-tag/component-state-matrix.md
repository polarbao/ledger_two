# Task53 分类标签组件状态矩阵

状态：Task53P.5 设计要求基线

| Component | Required variants | Forbidden behavior | Owner |
|---|---|---|---|
| ClassificationSummary | auto/suggested/fallback/manual/bulk/conflict/loading | 把 fallback 计入成功识别、只用颜色表达 | Task53U |
| ClassificationStatusChip | source/confidence/status、light/dark | 显示“AI 已完成”、隐藏可读标签 | Task53U |
| ClassificationExplanation | rule/learned/builtin/fallback/conflict | hover-only 原因、暴露数据库 ID | Task53U |
| ImportPreviewRows | desktop-table/mobile-card/long-merchant/eight-tags | 卡片套卡片、长文本撑破布局 | Task53U |
| ImportRowEditor | idle/invalid/submitting/row-saved-learn-failed | 学习失败回滚行保存、超过 8 标签静默截断 | Task53U |
| BulkClassificationBar | accept-suggestions/apply-values/partial-success | 接受建议后自动 commit、批量隐式学习 | Task53U |
| RememberMerchantControl | hidden/available/checked/conflict/error | 默认勾选、空商户显示、Viewer 可用 | Task53U |
| ReclassifySurface | dry-run/confirm/protected-manual/stale | 默认直接执行、覆盖 manual/bulk | Task53U |
| ImportRuleManager | manual/learned/auto/suggest/stale/archived | 隐藏学习规则、不可撤销清除 | Task53U |
| MetadataProfileSelector | basic/empty/loading/error | 既有账本静默应用、名称冲突自动绑定 system_key | Task53U |
| MetadataProfilePreview | create/reuse/conflict/skip/no-op | 未解决 conflict 仍提交 | Task53U |
| FallbackCategoryReplacement | selecting/type-mismatch/submitting/error | 直接归档兜底、改写历史交易分类 | Task53U |

## Copy baseline

| Scenario | Title | Primary action |
|---|---|---|
| 建议 | 待接受分类建议 | 接受所选建议 |
| 兜底 | 使用其他分类 | 逐条检查 |
| 冲突 | 多条规则给出不同分类 | 选择分类 |
| 批量相同值 | 将分类和标签应用到所选账单 | 应用到 N 条 |
| 同商户 | 将设置应用到相同商户 | 仅本批次应用 |
| 显式学习 | 记住此商户 | 保存并记住 |
| 学习失败 | 本行已保存，规则未创建 | 重试创建规则 |
| 基础包 | 补充基础分类与标签 | 预览基础包 |
| 兜底替代 | 先选择新的其他分类 | 替换并归档 |
| 重分类 | 检查当前规则会产生的变化 | 重新分类可处理行 |

所有危险或长期动作必须写明对象和影响范围，不使用单独“确定”“继续”。
