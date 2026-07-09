# Task41-Task49 Detailed DEV Plan

状态：建议任务规格  
适用阶段：Foundation before v1.1 完成后

## 1. 使用方式

本文细化 Task41-Task49。执行任一任务前，必须读取：

1. `docs/00_DOCUMENT_INDEX.md`
2. `docs/prd/24-short-mid-module-breakdown.md`
3. 对应模块 PRD
4. `docs/tech/18-short-mid-architecture-slices.md`
5. `docs/ui/14-v1.1-v1.2-module-flows.md`
6. Task47-Task49 额外读取 `docs/prd/29-prd-v1.2-module-business-service-breakdown.md` 与 `docs/tech/20-v1.2-import-implementation-contract.md`

## 2. Task41：快捷记账默认值

依赖：

- Task31-Task40 完成。
- 分类、标签、账户查询 API 稳定。

状态：已完成；Task41.1 服务端用户记账默认值、读取接口、前端默认值消费和账户选择已完成，Task41.2 金额输入聚焦与移动端键盘提示代码完成，Task41.3 保存并继续记字段保留规则已补前端纯函数测试。本机 WSL2 移动端验收已覆盖记账抽屉打开、金额输入聚焦画面、普通支出保存并继续、共同支出提交和 375px 无横向溢出；验收证据见 `docs/project_analysis/v1.1-ui-submit-acceptance-2026-07-07/`。

开发范围：

- 后端保存和读取用户记账默认值。（已完成）
- 前端记账表单应用默认值。（已完成）
- 保存并继续记。（已完成基础逻辑、字段保留规则测试和本机 WSL2 UI 实际提交验收）
- 移动端金额输入优化。（代码完成并通过浏览器截图验收）

完成标准：

- 普通支出只填金额即可保存。
- 共同支出默认两人 equal split。
- 保存并继续记不会保留上一笔金额。
- 默认值不使用已归档元数据。

验证：

- 后端 service test。
- 前端表单测试。
- 手工验证 375px 记账路径。（本机 WSL2 Chrome CDP 已完成普通支出、共同支出和保存并继续）

## 3. Task42：复制与模板统一

依赖：

- Task41 完成。
- 元数据归档规则明确。

状态：已完成；Task42.1 模板创建/更新的角色、分类、账户、付款人和分摊方式校验已完成；Task42.2 模板软归档、恢复和管理列表已完成；Task42.3 复制账单 preview 已完成；Task42.4 从已有账单创建模板入口已完成；Task42.5 模板生成账单的服务端元数据重校验已完成；Task42.6 模板管理列表编辑入口与编辑弹窗已完成；Task42.7 本机 WSL2 真实 UI 验收已覆盖复制一笔、存为模板、模板填入、模板生成账单和 375px 无横向溢出。

开发范围：

- 账单复制 preview。（已完成）
- 从账单创建模板。（已完成入口）
- 模板列表、编辑、归档。（创建/更新输入校验、软归档、恢复和前端编辑入口已完成）
- 从模板生成账单。（服务端创建账单时已重新校验付款人、分类、账户和写权限）

完成标准：

- 复制不修改原账单。
- 模板不进入统计和结算。
- 模板生成账单时重新校验权限和元数据。

验证：

- 复制 service test。
- 模板实例化 service test。
- 前端复制和模板生成流程测试。（本机 WSL2 Chrome CDP 已完成）

## 4. Task43：周期账单待确认

依赖：

- Task42 的模板 payload 设计稳定，或明确独立 payload。

状态：已完成核心收口；既有代码已具备周期规则 CRUD、pending reminder 生成、确认生成真实账单和历史 ignore 入口；已补充 `/skip` 语义兼容入口，并在周期规则页展示待确认项、确认入账和跳过本期操作。本机 WSL2 API 验收已覆盖跳过本期不生成账单、确认后生成真实账单，以及空 pending 提醒列表返回 `[]` 而非 `null`。

开发范围：

- 周期规则 CRUD。
- pending instance 生成。
- 确认生成真实账单。
- 跳过本期。

完成标准：

- pending 不进入统计和结算。
- 确认后才生成 transaction。
- 删除规则不影响已确认账单。

验证：

- rule next_run 计算测试。
- confirm/skip service test。
- Dashboard 待确认入口手工验证。
- 本机 API 验收记录：`docs/project_analysis/v1.1-settings-safety-acceptance-2026-07-07/`

## 5. Task44：分类、标签、账户管理体验

状态：已完成核心收口；Task44.1 元数据排序能力完成，Task44.2 归档确认与历史引用信息完成，Task44.3 管理列表筛选与移动端密度代码收口完成。本机 WSL2 移动端验收已覆盖分类管理页 375px 无横向溢出；本机 WSL2 API 验收已覆盖分类、标签、账户归档恢复和 editor 拒绝创建。

依赖：

- Foundation RBAC 完成。

开发范围：

- 分类、标签、账户新增、编辑、排序、归档、恢复。
- 设置页二级管理。
- 表单选择器过滤归档项。

当前切片：

- Task44.1：追加 `sort_order` 基础，分类/标签/账户列表支持上下移动排序。（已完成）
- Task44.2：列表返回 `usage_count`，归档确认区分未使用和已被历史账单引用，历史账单仍显示原名称。（已完成）
- Task44.3：增加搜索、状态筛选和筛选状态下的稳定排序，降低移动端长列表管理成本。（代码完成，待浏览器截图验收）

完成标准：

- 已使用项不可物理删除。
- 历史账单显示归档项。
- viewer 无管理入口且后端拒绝。

验证：

- 后端权限和归档测试。
- 前端设置页测试。
- 历史账单展示手工验证。

## 6. Task45：结算页可解释性与复制文案

依赖：

- 结算服务测试基线稳定。

状态：已完成；Task45.1 结算余额 DTO 已补充 raw_net、settlement_net、final_net 解释字段；Task45.2 结算页已展示 paid/share/raw_net/settlement/final_net 并提供复制结算文案入口；Task45.3 已增加影响结算的共同支出明细入口，跳转到流水页共同支出筛选视图；Task45.4 本机 WSL2 真实 UI 验收已覆盖复制文案兜底、登记结算、结算历史和 375px 无横向溢出。

开发范围：

- settlement explanation DTO。
- 共同支出影响明细。
- 复制结算文案。

完成标准：

- paid/share/raw_net/settlement/final_net 与后端计算一致。
- 复制文案不改变结算状态。
- settlement 不进入消费统计。

验证：

- 结算 service test。
- 前端展示测试。
- 手工验证复制文案。（本机 WSL2 Chrome CDP 已完成，含剪贴板失败兜底）

## 7. Task46：移动端高频路径优化

依赖：

- Task41、Task44、Task45 页面结构稳定。

状态：已启动；Task46.1 流水页移动端筛选已收口为单一底部 Sheet，桌面筛选面板不再参与移动端布局；Task46.2 流水详情、删除确认和批量打标签操作区已支持移动端换行与全宽按钮；Task46.3 周期账单待确认卡片和删除确认操作区已完成移动端纵向收口；Task46.4 设置页备份/导出确认弹窗操作区已支持移动端换行与全宽按钮；Task46.5 记账抽屉主操作区和模板弹窗操作区已接入移动端全宽按钮；Task46.6 分类/标签/账户管理列表项已完成移动端操作按钮收口；Task46.7 Dashboard 周期待确认入口已统一跳过本期语义并完成移动端按钮收口；Task46.8 本机 WSL2 Chrome CDP 截图验收完成，覆盖 375px/390px/430px，横向溢出 0，React Router 错误页 0。

开发范围：

- Dashboard、流水、记账、结算、设置移动端收口。
- 流水筛选 bottom sheet。
- 表单分组和危险操作确认。

完成标准：

- 375px 无横向滚动。
- 手机端可完成记账、筛选、结算、复制文案。
- 危险操作不误触。

验证：

- Playwright 或手工截图验证 375px/390px/430px。
- 前端 build。

## 8. Task47：CSV 导入预览

状态：已完成；`internal/importer` 已接入 `/api/imports/preview`、`GET /api/imports/{batchID}`、`PATCH /api/imports/{batchID}/rows/{rowID}`，微信/支付宝/通用 CSV fixture、批次持久化、行级调整、owner 权限和前端导入预览工作台已落地。Task47U 移动端验收证据见 `docs/project_analysis/v1.2-task47u-import-workbench-2026-07-08/`。

依赖：

- v1.1 完成。
- 分类、标签、账户管理稳定。
- `docs/prd/29-prd-v1.2-module-business-service-breakdown.md` 已明确 ImportBatch、ImportRow、Parser、Normalizer、Batch、Row、Dedupe、Rule、Commit 服务边界。
- `docs/tech/20-v1.2-import-implementation-contract.md` 已确认 API 迁移策略、状态机和 DTO。

开发范围：

- 新建或整理 `internal/importer`，旧 `/api/transactions/import/*` 仅作为 transitional 兼容入口。
- 实现 `/api/imports/preview`，上传并生成 `ready` 预览批次。
- 微信/支付宝/通用 CSV parser 与 normalizer。
- 字段映射、金额转整数分、时间标准化。
- 行级状态和错误展示。

完成标准：

- 预览不写 transactions。
- 金额转整数分。
- 错误提示包含行号和原因。
- invalid 行不得进入可提交状态。
- owner 以外角色默认不可预览导入。

验证：

- parser fixture test。
- 上传预览 API test。
- 前端预览页测试。
- 375px 预览卡片或核心字段视图无横向滚动。

## 9. Task48：导入去重与事务落库

状态：进行中；Task48 已定案采用独立 `transaction_import_refs` 表保存正式账单与导入行的唯一映射，不直接修改 `transactions` 主表。后端 `/api/imports/{batchID}/commit` 基线已实现，可在单事务内写入正式账单、导入映射、批次状态和审计日志；后续继续补 Task48U 提交确认/结果反馈、suspicious 手工确认 UI 和更完整的事务失败验收。

依赖：

- Task47 完成。
- import_hash 存储方案定案：采用独立 `transaction_import_refs` 映射表，同账本内 `import_hash` 唯一。
- Task47 的 batch/row service 和预览工作台已冻结。

开发范围：

- import_hash。
- duplicate/suspicious/invalid 状态。
- commit 事务落库。
- import batch 结果页。
- audit log 写入。

完成标准：

- 同一文件重复导入不重复。
- 任一必需行失败整批回滚。
- 审计日志记录导入提交。
- duplicate 默认跳过。
- suspicious 必须用户明确确认导入或跳过。
- 转账类默认 skipped 或 unknown，不自动写入正式账单。

验证：

- dedupe service test。
- transaction rollback test。
- 重复导入手工验证。
- Case I02/I03/I04 全覆盖。

## 10. v1.2 UI/UX 并行切片

v1.2 的 UI/UX 不作为“最后美化”处理，而是随 Task47-Task49 并行交付。编号采用 `Task47U/48U/49U`，避免打乱既有后端导入任务编号。

### 10.1 Task47U：导入入口与预览工作台

依赖：

- Task47 后端 preview API 可用，或 mock DTO 与 OpenAPI 草案稳定。
- `docs/ui/figma/v1.1-v1.2-ui-draft-spec.md`、`docs/ui/figma/component-library.md` 已确认。

开发范围：

- 设置页导入入口状态：owner 可进入，editor/viewer 显示无权限说明。
- Import Entry：来源选择、上传区、格式错误提示。
- Preview Workbench：批次统计、行状态汇总、移动端行卡片。
- invalid/duplicate/suspicious/new 四类状态视觉和文案。

完成标准：

- 375px 移动端不使用宽表格，无横向滚动。
- 用户能在预览页理解“尚未写入正式账单”。
- invalid 行展示行号、错误码和可行动说明。
- commit 按钮在 Task47 阶段隐藏或 disabled，并标注“预览阶段暂不可提交”。

验证：

- 前端 build。
- 375px/390px 导入预览截图或 CDP 指标。
- owner/editor/viewer 入口状态验收。

### 10.2 Task48U：提交确认与结果反馈

状态：基础已落地；导入工作台已接入 commit API、提交确认弹窗、导入/跳过/疑似/错误数量二次确认、成功结果反馈和 suspicious 行“确认导入”操作。后续还需补 375px/390px 实际截图验收和事务失败 UI 手工记录。

依赖：

- Task48 commit API 可用。
- duplicate/suspicious/invalid 状态语义冻结。

开发范围：

- Commit Confirm Modal。
- 导入、跳过、疑似、错误数量二次确认。
- 导入结果页或结果区。
- duplicate 默认跳过、suspicious 必须人工确认的提示。

完成标准：

- 提交前明确展示本次会写入多少正式账单。
- suspicious 未确认时不能提交。
- 提交成功后可回看 imported/skipped/failed。
- 失败时不暗示已写入半批数据。

验证：

- commit modal 前端测试或 CDP 验收。
- 事务失败 UI 状态验收。
- 375px 无横向滚动。

### 10.3 Task49U：规则管理与推荐解释

状态：主体已落地；导入工作台已加入规则管理面板，支持创建、编辑、归档、恢复、active/archived/all 筛选和多标签推荐选择，预览行已展示规则命中解释与推荐分类/账户/标签名称；规则列表已补充归档或不可用元数据提示。375px/390px 截图验收仍需继续收口。

依赖：

- Task49 规则 API 可用。
- 分类、标签、账户归档规则稳定。

开发范围：

- Import Rule Manager。
- 规则命中来源说明。
- 规则归档/恢复状态。
- 规则引用归档元数据时的阻断或替换提示。

完成标准：

- 规则只展示建议，不自动提交账单。
- 用户手工调整优先于规则。
- archived 规则不再命中。
- 规则命中解释在预览行可见。

验证：

- 规则命中前端状态测试。
- 归档规则和归档元数据回归验收。
- Figma handoff checklist 同步更新，并补充 `docs/ui/figma/v1.2-task49-import-rule-manager-handoff.md`。

## 11. Task49：导入规则

状态：进行中；Task49 规则扩展 migration 已顺延为 `017_extend_import_rules_for_v12.sql`，在保留旧 `keyword/category_id/account_id/tag_names` 兼容字段的基础上新增 `match_type/pattern/result_json/status/priority/archived_at`。后端 `/api/import-rules` 已切到 `internal/importer`，支持 create/list/update/archive/restore，旧 `DELETE` 兼容入口改为归档；预览阶段已接入 active 规则建议，命中结果只写 `suggested_*` 与解释字段，不覆盖用户 selected 字段。前端规则管理面板和命中解释基础已落地；剩余收口项见 `docs/project_analysis/2026-07-09-v1.2-task49-readiness-and-closure.md`。

依赖：

- Task48 完成。
- 分类、标签、账户归档规则稳定。
- ImportRule 业务对象、权限、规则优先级和归档语义已按 v1.2 细分文档冻结。

开发范围：

- 导入规则 CRUD、归档、恢复。
- 预览页规则命中。
- 用户手工调整优先。
- 引用已归档元数据时阻止静默应用。

完成标准：

- 规则只推荐，不自动提交。
- 禁用规则后不再命中。
- 手工修改不被规则覆盖。
- 规则命中记录可在预览行看到，方便解释分类来源。

验证：

- rule matching test。
- 前端预览页调整测试。
- 归档规则和归档元数据回归测试。

## 12. 禁止混入

- Task41-Task46 不做 CSV 导入。
- Task47-Task49 不做 OCR。
- Task41-Task49 不做直接通知共同支付。
- 不新增银行同步。
- 不绕过 service 层业务规则。

## 13. v1.1 冻结收口项

以下事项不属于单一业务 Task，但必须在 v1.1 冻结前完成或形成明确验收记录：

- 健康检查版本口径：`GET /api/healthz` 返回 `version=1.1.0-rc`、`db` 和 `schema_version`，用于区分 NAS 是否运行最新候选版本。（代码、OpenAPI 和 NAS 部署验证已完成）
- 前端质量门禁：`corepack pnpm test` 与 `corepack pnpm build` 已通过；主 JS chunk 大小警告进入后续性能优化，不阻断 v1.1 内测。
- 移动端验收记录：本机 WSL2 已补齐 375px/390px/430px 关键入口截图和 scrollWidth 指标，详见 `docs/project_analysis/v1.1-local-acceptance-2026-07-07/`；普通支出保存并继续、共同支出提交、equal split 和结算余额联动已补 UI 实际提交闭环，详见 `docs/project_analysis/v1.1-ui-submit-acceptance-2026-07-07/`。
- 设置与数据安全验收记录：本机 WSL2 API 已覆盖分类/标签/账户归档恢复、模板管理、周期确认/跳过、附件受控读取、手动备份、系统诊断和非 owner 拒绝高风险入口，详见 `docs/project_analysis/v1.1-settings-safety-acceptance-2026-07-07/`。
- NAS 内测验收记录：完成登录、记账、附件、手动备份、系统诊断的浏览器内验证。
