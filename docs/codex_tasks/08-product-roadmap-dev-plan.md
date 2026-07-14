# Product Roadmap DEV Task Plan

状态：当前任务入口，持续维护<br>
适用阶段：Foundation before v1.1 之后

> 执行更新（2026-07-14）：Task41-Task49 已完成。因微信、支付宝实际交付 XLSX，v1.2 RC 通过 Task49X 重新打开输入格式范围；Task49X 外部门禁完成后再恢复冻结。Fresh Light 波次 A（UI-FL-01/02）与 UI-FL-03/04 已完成，UI-FL-05 流水工作台为下一项；v1.3 业务任务仍需重新评审。

## 1. 使用说明

本文档把新的产品规划转成后续可执行 DEV 任务。执行前必须先完成或确认不阻塞：

- `docs/codex_tasks/05-foundation-task-plan.md` Task31-Task40。
- `docs/prd/20-product-retrospective-and-positioning.md`。
- `docs/prd/21-roadmap-short-mid-long.md`。
- `docs/prd/22-prd-v1.1-trust-and-daily-use.md`。
- `docs/prd/23-feature-priority-and-deferral-decisions.md`。
- `docs/prd/24-short-mid-module-breakdown.md`。
- `docs/prd/25-prd-v1.1-module-specs.md`。
- `docs/prd/26-prd-v1.2-import-module-specs.md`。
- `docs/prd/29-prd-v1.2-module-business-service-breakdown.md`。
- `docs/tech/18-short-mid-architecture-slices.md`。
- `docs/tech/20-v1.2-import-implementation-contract.md`。
- `docs/ui/14-v1.1-v1.2-module-flows.md`。
- `docs/ui/figma/ledger-two-fresh-light-implementation-spec-2026-07-13.md`（涉及 Fresh Light 或后续 UI 变更时）。
- `docs/codex_tasks/09-task41-49-detailed-plan.md`。
- `docs/codex_tasks/13-fresh-light-ui-interaction-plan.md`（涉及 UI-FL 或业务 Task UI 协同时）。

## 2. v1.1 DEV 任务

### Task41：快捷记账默认值

目标：

1. 记录最近分类、账户、标签、付款人。
2. 新增普通支出时自动带出默认值。
3. 支持“保存并继续记”。
4. 移动端金额输入优化。

非目标：

- 不做 AI 自动分类。
- 不做银行/OCR。

验收：

- 普通支出 10 秒内完成。
- 保存并继续记保留合理默认值。
- 离线时只保存草稿。

### Task42：账单复制与模板统一

目标：

1. 普通账单和共同支出都支持复制。
2. 模板可从已有账单创建。
3. 模板生成账单时重新校验分类、账户、成员和分摊。
4. 模板不进入统计和结算。

验收：

- 复制不修改原账单。
- 模板删除不影响历史账单。

### Task43：周期账单待确认

目标：

1. 支持每周、每月、每年规则。
2. 到期后生成待确认提醒。
3. 用户确认后才生成真实账单。
4. 取消提醒不影响历史账单。

非目标：

- 不自动扣账。
- 不做推送通知。

### Task44：分类、标签、账户管理体验

目标：

1. 设置页增加分类、标签、账户二级管理页。
2. 支持新增、编辑、排序、归档、恢复。
3. 已归档项不出现在新增账单默认选择器。
4. 历史账单保留并展示归档项。

验收：

- owner 可管理。
- viewer 不可管理。
- 已使用项不可物理删除。

### Task45：结算页可解释性与复制文案

目标：

1. 结算页展示 paid/share/raw_net/settlement/final_net。
2. 增加影响结算的共同支出明细入口。
3. 增加“复制结算文案”。
4. 不做自动通知共同支付。

验收：

- 文案包含月份、付款人、收款人、金额。
- 复制文案不改变结算状态。

### Task46：移动端高频路径优化

目标：

1. 375px 宽度下 Dashboard、流水、记账、结算、设置无横向滚动。
2. 流水筛选使用底部 Sheet。
3. 记账表单分组，避免过长。
4. 高风险确认适配移动端。

验收：

- 手机端可完成记账、筛选、结算。
- 危险按钮不误触。

## 3. v1.2 DEV 任务

### Task47：CSV 导入预览

目标：

1. 支持微信/支付宝 CSV 基础解析。
2. 上传后只生成预览，不写正式账单。
3. 支持字段映射。
4. 格式错误有明确错误。

### Task48：导入去重与事务落库

目标：

1. 生成 import_hash。
2. 展示待导入、重复跳过、需人工确认数量。
3. 用户确认后事务落库。
4. 写入审计日志。

### Task49：导入规则

目标：

1. 商户/描述关键词匹配分类、标签、账户。
2. 预览页展示命中规则。
3. 用户调整优先于规则。

### Task49X：微信/支付宝 XLSX 导入专项

目标：

1. 在现有 preview/commit 管线前增加安全 XLSX reader。
2. 支持非首行表头、工作表选择、金额/日期/订单号精度保护。
3. CSV/XLSX 相同流水生成一致 import_hash。
4. 前端支持格式选择、解析摘要和可行动错误。
5. staging 真实文件只预览验收，用户确认后才允许 production 提交。

详细计划：`docs/codex_tasks/12-v1.2-xlsx-import-special-plan.md`。

## 4. v1.2 收口后 Fresh Light UI/UX 专项

状态：计划已冻结；波次 A（UI-FL-01/02）与 UI-FL-03/04 已完成，UI-FL-05 为下一项。

执行入口：`docs/codex_tasks/13-fresh-light-ui-interaction-plan.md`。

任务：

1. UI-FL-01/02：Token、基础组件、AppShell 和全局导航。
2. UI-FL-03/04/05：Dashboard、记账抽屉和流水高频路径。
3. UI-FL-06/07：结算、设置与元数据可信交互。
4. UI-FL-08/09：导入工作台和分析钻取。
5. UI-FL-10：375/390/430/1440、可访问性和真实业务回归。

协调规则：

- `Task47U/48U/49U/49XU` 保留为 v1.2 导入业务与历史验收事实，UI-FL-08 复用而不覆盖它们。
- 后续业务 Task 触及 UI 时，必须反向登记 UI-FL 编号、共享组件归属、API 不变声明和截图范围。
- 共享组件契约未冻结或存在同文件所有权冲突时不得并行；优先完成归属任务，再推进页面任务。
- UI 专项不新增 API、migration 或业务状态；确需变更时另开 PRD/DEV 评审。

## 5. v1.3+ DEV 任务

### Task50：多账本正式化

目标：

1. 账本创建、切换、归档。
2. 成员角色稳定。
3. 数据隔离测试完善。

### Task51：多人分摊体验增强

目标：

1. 多人 equal/amount/ratio/shares UI 优化。
2. 转账建议解释。
3. 旅行/聚会账本场景评估。

### Task52：通知共同支付调研

目标：

1. 只做产品调研，不实现。
2. 评估聚会/旅行场景频率。
3. 明确通知渠道、状态机和隐私成本。

准入条件：

- 多人账本稳定。
- 用户真实反馈明确。
- 有低成本原型或复制文案数据验证。

## 6. 执行规则

1. 每个任务单独分支、单独提交。
2. 每个任务必须先读对应 PRD/Tech/UI。
3. 不允许把 v1.2/v1.3 功能混入 v1.1。
4. 不允许绕过金额整数分和后端结算可信源。
5. 完成后必须说明验证命令和未验证原因。
6. UI 任务必须遵守 Task13 的双向登记、共享组件归属和四个同步点，不得把设计稿直接当作业务契约。
