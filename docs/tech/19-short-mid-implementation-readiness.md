# 技术：短中期实施就绪评审

状态：持续更新
适用阶段：Foundation 收尾与开始 Task41-Task49 前

## 1. 结论

当前短中期产品模块已经足够进入分阶段开发，但截至 2026-07-06 仍不能直接冻结 Foundation 并全面进入 v1.1 业务功能。正确顺序是：

1. 先完成 Foundation before v1.1 的最后收口：Task40 系统诊断与审计规范。
2. 再做 v1.1 分类、标签、账户管理体验收口。
3. 再进入快捷记账、模板、周期账单、结算解释和移动端优化。
4. 最后进入 v1.2 导入能力。

原因：

- Task41 快捷记账依赖稳定的分类、标签、账户和权限。
- Task42 模板依赖账单复制、元数据校验和权限边界。
- Task47-Task49 导入依赖分类、标签、账户和审计能力。
- 多账本/RBAC 与附件权限已经有回归覆盖，但系统诊断和审计规范未完成前，NAS 长期运维能力仍不足。

当前判断：

| 项目 | 状态 |
|---|---|
| Foundation Task31-Task39 | 基本完成 |
| Foundation Task40 | 未完成 |
| NAS 内测部署 | 可进行 |
| v1.1 开发冻结 | 暂不建议 |

## 2. 文档充分性判断

### 2.1 已足够

以下文档已经足够支撑产品、UI 和高层技术评审：

- `docs/prd/24-short-mid-module-breakdown.md`
- `docs/prd/25-prd-v1.1-module-specs.md`
- `docs/prd/26-prd-v1.2-import-module-specs.md`
- `docs/prd/27-acceptance-case-matrix.md`
- `docs/prd/28-transaction-caliber-supplement.md`
- `docs/tech/18-short-mid-architecture-slices.md`
- `docs/ui/14-v1.1-v1.2-module-flows.md`
- `docs/codex_tasks/09-task41-49-detailed-plan.md`
- `docs/codex_tasks/10-task33-40-detailed-plan.md`

它们已经明确：

- 模块范围。
- 非目标。
- 依赖顺序。
- 数据边界。
- UI 流程。
- 测试方向。
- 冻结验收样例。
- 退款、报销、转账和账户口径。

### 2.2 仍需在实施中补齐

进入具体任务时，还需要随代码一起补齐：

| 缺口 | 处理方式 |
|---|---|
| 精确 migration 字段、索引、唯一约束 | 每个涉及数据模型的任务新增 migration 设计说明 |
| API request/response DTO | 每个任务更新 OpenAPI 或 API contract |
| service/repository 接口 | 在任务实现中随代码明确，不提前写死 |
| 测试 fixture | 跟随后端测试和前端测试落地 |
| feature flag 或入口关闭策略 | 涉及 v1.1/v1.2 新入口时补充 |
| 迁移回滚方案 | 每个 migration 任务给出修正策略，不修改历史 migration |

## 3. 推荐执行顺序

### 3.1 Foundation

| 顺序 | 任务 | 原因 |
|---:|---|---|
| 1 | Task32 配置与部署安全 | 低业务耦合，先消除生产安全风险 |
| 2 | Task33 LedgerContext 与 RBAC | 所有后续写入 API 的安全前提 |
| 3 | Task34 API 契约与 OpenAPI | 后续新增 API 的合同基线 |
| 4 | Task35 分类、标签、支付账户管理基础 | v1.1 高频记账和 v1.2 导入规则的共同基础 |
| 5 | Task36 前端 LedgerProvider 与 Query Key | 多账本和权限 UI 的前端基础 |
| 6 | Task37 设置页信息架构重组 | 承载元数据、导入、模板、诊断入口 |
| 7 | Task38 迁移、测试与质量门禁 | 为后续功能提供回归保护 |
| 8 | Task39 附件访问控制 | 修复 private 账单附件泄露风险 |
| 9 | Task40 审计与系统诊断中心 | 补齐运维与审计可见性 |

Task31 已通过当前文档收口完成，不再单独进入代码实现。

### 3.2 v1.1

| 顺序 | 任务 | 原因 |
|---:|---|---|
| 1 | Task44 分类、标签、账户管理体验 | 先让长期元数据稳定 |
| 2 | Task41 快捷记账默认值 | 依赖元数据和权限 |
| 3 | Task42 复制与模板统一 | 依赖账单与元数据校验 |
| 4 | Task43 周期账单待确认 | 依赖模板 payload 或等价账单快照 |
| 5 | Task45 结算页可解释性与复制文案 | 独立提升信任，但必须沿用后端计算 |
| 6 | Task46 移动端高频路径优化 | 页面结构稳定后收口 |

### 3.3 v1.2

| 顺序 | 任务 | 原因 |
|---:|---|---|
| 1 | Task47 CSV 导入预览 | 先保证不写库的可控导入 |
| 2 | Task48 导入去重与事务落库 | 在 preview 稳定后再开放写入 |
| 3 | Task49 导入规则 | 规则必须基于稳定预览和元数据 |

## 4. 每个任务的实施前检查

开始任何任务前必须确认：

1. 是否涉及 migration。
2. 是否涉及权限变化。
3. 是否涉及金额计算。
4. 是否涉及多账本隔离。
5. 是否涉及 private / partner_readable / shared 可见性。
6. 是否需要 OpenAPI 或 API contract 更新。
7. 是否需要前端 query key 增加 ledgerId。
8. 是否需要审计日志。
9. 是否需要移动端验收。

只要任一项为“是”，任务验收必须包含对应测试或手工验证说明。

## 5. 当前可直接开始的原子任务

建议从 Task32 开始：

- 任务边界清楚。
- 不依赖 v1.1 新业务。
- 可通过配置测试验证。
- 风险集中在启动配置和部署文档，容易回滚。

Task32 的最小原子切片：

1. 读取当前 config、`.env.example`、Docker Compose 和部署文档。
2. 对齐环境变量命名。
3. 增加 production 强密钥校验。
4. 增加配置测试。
5. 更新文档。

## 6. 不建议现在开始的任务

暂不建议直接开始：

- Task41：依赖 Task35 和 Task33。
- Task42：依赖 Task41 和元数据归档。
- Task47：依赖 v1.1 与导入权限。
- Task50+：属于长期规划，不进入当前执行。
