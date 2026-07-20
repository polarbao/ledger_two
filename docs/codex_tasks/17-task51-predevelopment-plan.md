# Task51 多人分摊开发前准备计划

状态：Task51P.1 内部准备完整并进入非约束性证据执行；Task50/Task53 WSL2 技术门禁已满足，但真实证据 Gate 未满足
创建日期：2026-07-16
前置任务：Task50.1-Task50.6 与 Task53 WSL2/NAS staging 验收已完成；production 保持独立发布线
禁止事项：不得实现 Task51 代码、解除两人约束或冻结 migration/版本号

调度说明：Task53.1-Task53U 与 Task53.5 WSL2 验收已完成，结论 `pass_with_suggest_only`；冻结任务树没有 Task53.6。Task51P.1 的匿名证据模板、登记、3/5 人假设原型、访谈执行手册和决策模板已经完整，可进入真实证据执行；有效真实目标小组/完整回放仍为 0/0，P2-P6、代码和冻结性产物继续等待 `continue/narrow` 决策。

## 1. Goal

把 Task51 从路线级“多人分摊体验增强”转化为可验证的产品、架构、算法、API、数据、UI 和发布准备包。只有全部 Gate 完成并通过评审后，才允许生成 Task51 详细开发任务。

## 2. Scope

1. 证明 3+ 成员场景的真实优先级，并允许得出延后结论。
2. 冻结成员生命周期、角色、历史可见性和隐私边界。
3. 冻结多人 split/settlement 的整数分算法、解释、API 和数据迁移方案。
4. 形成移动端可用的 UI/Figma handoff、匿名 Fixture 和完整发布门禁。
5. 保持旧双人账本、现有结算记录和 Task50 schema 21 行为向前兼容。

## 3. Non-goals

1. 不默认实现公开邀请、直接通知共同支付、支付状态追踪或催款。
2. 不扩展企业多租户、审批、银行同步、OCR 或原生 App。
3. 在 P1 未形成 `continue/narrow` 前不修改 schema、OpenAPI 正式契约或生产代码。
4. 不以 Figma 画面替代金额守恒、权限、migration 和回滚评审。

## 4. Entry gate

Task51P 正式启动前必须满足：

1. Task50.3 生命周期/成员/实例 API 已整体关闭。
2. Task50.4 active-ledger 状态机、Task50.5 管理 UI 和 Task50.6 全模块/升级验收已完成。
3. schema 21、目标应用提交、Fixture 32 和独立 v1.3 staging 证据可追踪。
4. v1.2 production/NAS 未完成事项与 Task51 development 数据目录物理隔离。
5. 用户重新确认 Task51 进入正式准备，不把本草案自动视为版本承诺。

当前允许按冻结 runbook 执行 P1 访谈和匿名原型回放；不得把“准备完整”写成“证据 Gate 完成”。

已建立工作区：`docs/project_analysis/task51_p1/`，并完成进入执行阶段评审 `docs/project_analysis/2026-07-20-task51-p1-entry-review.md`。当前有效真实小组证据数为 0，P1 状态保持进行中。

## 5. Gates

### Task51P.1：场景与证据

事实源：`docs/prd/33-task51-scenario-evidence-and-scope-questions.md`

步骤：

1. 收集匿名访谈/工作流证据和替代方案。
2. 覆盖长期家庭/合租、旅行/装修和偶发聚会，不只验证单一假设。
3. 使用 3 人与 5 人 Fixture/Figma 原型回放核心流程。
4. 输出 `continue`、`narrow` 或 `defer`，并说明通知功能在无实现情况下是否影响价值。

完成标准：证据来源、隐私同意、场景频率和决策理由可审阅；材料不足必须 defer。

当前进度：

1. 匿名证据登记表和单条记录模板已完成。
2. 3 人、5 人假设原型回放 Fixture 已完成，不计入真实证据。
3. 真实访谈、工作流回放和“不做通知仍愿意使用”的证据待收集。
4. 招募配额、匿名同意、35-45 分钟访谈/回放脚本和停止条件已形成执行手册。
5. Gate 计数、证据质量审查及 `continue/narrow/defer` 决策模板已形成；当前计数仍为 0，不改变 P1 状态。
6. P.1 内部准备审计已通过；后续不再新增平行模板，以真实招募、访谈、回放和匿名登记为唯一推进方式。

### Task51P.2：PRD 与权限/历史矩阵

前置：P1 结论为 continue 或 narrow。

必须冻结：

1. 目标用户、场景、成员上限和非目标。
2. Owner/Editor/Viewer 的成员、账单、split、settlement、导入、导出权限。
3. 加入前历史、离开后历史、再次加入、成员移除和账本归档语义。
4. private/partner_readable/shared 在 3+ 成员下的命名与可见性；若 partner_readable 心智不再成立，必须给兼容迁移而不是静默改义。
5. 验收指标与失败/空/权限/冲突状态。

完成标准：独立 Task51 PRD 与角色 x 对象 x 生命周期矩阵评审通过。

### Task51P.3：领域架构与 ADR

必须产出：

1. 多人成员/可见性模型 ADR，比较保留现有 visibility、引入 audience 关系表等至少两个方案。
2. split 与 settlement 服务边界、依赖方向和历史不变性。
3. schema 21 两人 trigger 的向前解除策略；禁止修改已应用 migration。
4. 并发 version、唯一 Owner、成员移交/离开和跨账本隔离策略。

完成标准：ADR 明确选项、取舍、兼容、回滚和被拒绝方案。

### Task51P.4：算法、API、Migration 与 Fixture

必须冻结：

1. equal/amount/ratio/shares 的整数分输入、余数分配、稳定排序和总额守恒。
2. 结算建议算法的确定性、复杂度、解释字段和“建议不等于已支付”。
3. OpenAPI、错误码、ETag/version、批量编辑和幂等边界。
4. 新 migration 草案、19/21 -> 新 schema 升级、异常数据预检、备份和向前修复。
5. 2/3/5/上限成员匿名 Fixture，覆盖负数禁止、余数、成员更替、历史可见性和结算守恒。

完成标准：算法属性测试、OpenAPI 引用检查和 migration review 清单可直接指导实现，但 migration 尚不应用于共享环境。

### Task51P.5：UI/UX 与 Figma handoff

必须覆盖：

1. 375/390/430/1440 下的成员选择、分摊编辑、总额校验、错误定位和结算解释。
2. 3、5、上限成员的信息密度；不使用卡片套卡片，不让成员名称/金额溢出。
3. 键盘、读屏、焦点返回、44x44 触控目标和动态错误播报。
4. Fresh Light 默认与 Dark Glass 回退；优先复用现有原语和 Task50 状态机。
5. 本地 Figma Frame Manifest、需求/生成/审阅文件边界和线上同步真实状态。

完成标准：required Frame、交互注释、组件归属、可访问性矩阵和代码验收范围齐全。

### Task51P.6：详细开发与发布准入

必须产出：

1. 按数据不变量 -> service/algorithm -> API -> frontend state -> UI -> 发布切片的详细 Task。
2. 每个 Task 的文件所有权、TDD 用例、提交边界、验证命令和回滚。
3. 独立 Task51 development/staging 目录、端口、数据库、密钥和镜像标签。
4. 数据数量/金额/split/settlement 守恒、旧双人兼容、跨账本泄漏和浏览器验收门禁。

完成标准：形成独立 readiness 记录，明确“可以开发”或“继续阻断”，不得以文档数量代替逻辑闭环。

## 6. Dependency order

```text
Task50.4 -> Task50.5 -> Task50.6
                                              |
                                              v
Task51P.1 -> P2 -> P3 -> P4 -> P5 -> P6 -> Task51 implementation plan
```

P1 的匿名问题模板可提前准备；P2-P6 不并行冻结。P3/P4 可在 P2 范围稳定后协同评审，但最终签字必须串行，避免算法或 migration 反向创造产品规则。

## 7. Risks

| 风险 | 控制 | 阻断条件 |
|---|---|---|
| 为偶发聚会扩大核心模型 | P1 允许 defer，长期关系场景优先 | 证据仅来自单次活动 |
| 破坏双人隐私语义 | P2 历史矩阵 + P3 ADR + 兼容 Fixture | private/partner_readable 新含义未冻结 |
| float/余数导致金额不守恒 | int64 cents、确定性余数和属性测试 | 任一 split 总和不等于交易金额 |
| 最小转账不可解释或不稳定 | 冻结排序、解释字段和反例 Fixture | 同输入产生不同建议 |
| 修改已应用 migration | 只新增向前 migration | 直接编辑 020/021 |
| Task51 与 Task53/UI 共改文件 | P1 决策后登记文件所有权 | 与当前 Task53 实现并行修改 |
| 通知功能偷渡主线 | 始终保持 Task52 调研边界 | 出现通知渠道/支付状态代码 |

## 8. Validation

准备阶段至少执行：

1. PRD、ADR、Tech、OpenAPI、Fixture、UI 和 Task 的双向链接检查。
2. OpenAPI YAML 解析、schema 引用和错误码清单检查。
3. Fixture 数据匿名性、金额/split/settlement 守恒静态检查。
4. migration 只读 review 和历史 migration 未修改检查。
5. Figma manifest required Frame、文件角色和 online sync 状态检查。
6. `git diff --check` 与敏感数据审计。

## 9. Rollback

Task51P 文档在正式冻结前均可回退或标记 defer，不影响 schema 21 和 Task50 行为。任何试验 Fixture、原型或 development 数据必须可丢弃；不得把未冻结 Task51 镜像部署到 WSL staging、NAS 或 production。
