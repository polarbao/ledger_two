# Task33-Task40 Detailed DEV Plan

状态：短期 Foundation 已完成；Task33-Task40 可作为 v1.1 前冻结基线
适用阶段：Foundation before v1.1  
目的：细化剩余 Foundation 任务，确保 v1.1 业务开发前基础框架可冻结。

## 0. 当前完成度快照

更新时间：2026-07-06

| 任务 | 当前状态 | 冻结判断 |
|---|---|---|
| Task33 LedgerContext 与 RBAC | 基本完成 | 已具备 v1.1 前置能力，仍需避免新业务继续 fallback |
| Task34 API 契约与 OpenAPI | 基本完成 | 新增 API 必须同步更新契约 |
| Task35 分类/标签/账户管理基础 | 基本完成 | 排序等增强可进入 v1.1 Task44 |
| Task36 前端 LedgerProvider 与 Query Key | 基本完成 | 后续新增 query 必须包含 ledgerId |
| Task37 设置页信息架构重组 | 完成 | 设置页已覆盖诊断区 |
| Task38 迁移、测试与质量门禁 | 完成 | migration、R01/R02、CI compose gate 已补 |
| Task39 附件访问控制 | 完成 | R03 已覆盖，裸 `/uploads/*` 已关闭 |
| Task40 审计与系统诊断中心 | 完成 | 诊断接口、设置页面板、脱敏与权限验收已补 |

结论：Foundation before v1.1 已具备冻结条件；后续开发应进入中期 v1.1 任务，推荐从 Task44 分类、标签、账户管理体验收口开始。

## 1. 使用方式

执行任一任务前必须读取：

1. `docs/00_DOCUMENT_INDEX.md`
2. `docs/prd/11-foundation-framework-before-v1.1.md`
3. 对应 Tech/UI 文档
4. 本文对应任务章节
5. 如涉及业务口径，读取 `docs/prd/27-acceptance-case-matrix.md` 与 `docs/prd/28-transaction-caliber-supplement.md`

## 2. Task33：LedgerContext 与 RBAC

已完成切片：

- Role、Operation、LedgerContext、RolePolicy 基础类型。
- owner/editor/viewer 权限矩阵单元测试。

剩余切片：

### Task33.1：Membership Resolver

范围：

- 新增根据 `user_id + ledger_id` 查询成员角色的 resolver。
- 明确非成员返回 403/404 的转换位置。
- 不接入业务 API。

验收：

- owner/editor/viewer 可解析。
- 非成员不可解析。
- 非法 role 返回错误或拒绝。

### Task33.2：LedgerContext 中间件/服务入口

范围：

- 从 `X-Ledger-Id` 或 URL ledger id 解析明确账本。
- 兼容未携带 ledger 的只读用户接口。
- 对业务写接口保留过渡兼容，但标记 deprecated。

验收：

- 明确 ledger 的请求可得到 LedgerContext。
- 非成员请求失败。
- 不再在新业务 API 中新增 fallback。

### Task33.3：业务模块逐步接入

顺序：

1. ledger members。
2. transaction write。
3. settlement write。
4. export/backup audit context。
5. reports/dashboard read。

验收：

- viewer 无法写入。
- 非成员不能通过 `X-Ledger-Id` 访问数据。
- private 可见性不被破坏。

## 3. Task34：API 契约与 OpenAPI

切片：

### Task34.1：API 现状盘点

范围：

- 从 router 生成当前 API 清单。
- 标记已稳定、过渡、待废弃路径。

验收：

- `docs/api/API_INVENTORY.md` 或等价文档存在。
- 每个路径有方法、认证要求、ledger 要求。

### Task34.2：OpenAPI 草案

范围：

- 新增 `docs/api/openapi.yaml`。
- 覆盖 auth、ledger、transaction、settlement、metadata、safety 核心路径。

验收：

- 文档包含统一成功/错误响应。
- 金额字段统一 `amount_cents`。
- 时间字段统一 ISO8601。

### Task34.3：错误码和分页规范

范围：

- 整理错误码。
- 明确分页、筛选、排序字段。

验收：

- 前端 API client 可按文档处理错误。
- 新 API 必须先更新契约。

## 4. Task35：分类、标签、支付账户管理基础

切片：

### Task35.1：后端元数据归档规则

范围：

- category/tag/account 支持归档、恢复。
- 已使用项不可物理删除。
- 名称同账本内唯一。

验收：

- 归档后新增选择器不返回。
- 历史账单仍可显示。
- viewer 不可管理。

### Task35.2：设置页管理入口

范围：

- 设置页增加分类、标签、账户管理区。
- 支持新增、编辑、排序、归档、恢复。

验收：

- 移动端无横向滚动。
- 高风险操作二次确认。

### Task35.3：测试与验收样例

必须覆盖：

- `docs/prd/27-acceptance-case-matrix.md` 的 M01、M02、M03。

## 5. Task36：前端 LedgerProvider 与 Query Key

切片：

### Task36.1：queryKeys 工厂

范围：

- 新增统一 query key 工厂。
- ledger scoped query 必须包含 ledgerId。

验收：

- Dashboard、Transactions、Settings 关键 query 不跨账本污染。

### Task36.2：LedgerProvider

范围：

- 管理 active ledger。
- 切换账本时刷新相关 query。
- 不使用 `window.location.reload()`。

验收：

- 切换账本后页面数据正确变化。

### Task36.3：PermissionGate

范围：

- 基于当前角色控制按钮和入口显示。

验收：

- viewer 不显示新增、编辑、删除、导出、备份入口。
- 后端仍必须拒绝越权请求。

## 6. Task37：设置页信息架构重组

切片：

### Task37.1：设置页分区

分区：

- 账号与登录。
- 账本与成员。
- 分类、标签、支付账户。
- 模板与周期账单。
- 导入导出。
- 备份恢复。
- 系统诊断。

验收：

- 现有入口不丢失。
- 移动端可用。

### Task37.2：角色态显示

范围：

- owner/editor/viewer 看到不同入口状态。
- 不可用入口显示原因或隐藏，按风险决定。

验收：

- viewer 看不到危险操作。

## 7. Task38：迁移、测试与质量门禁

切片：

### Task38.1：迁移回归测试

范围：

- 测试空库迁移到最新版本。
- 测试关键表和索引存在。

验收：

- 不修改历史 migration。
- 新 migration 只能追加。

### Task38.2：权限矩阵测试

范围：

- owner/editor/viewer 操作矩阵。
- 非成员访问。

验收：

- 对应 `docs/prd/27-acceptance-case-matrix.md` 的 R01、R02。

### Task38.3：CI 门禁

范围：

- 后端测试。
- 前端 lint/test/build。
- Docker build 或 dry-run。

验收：

- CI 文档和实际 workflow 一致。

## 8. Task39：附件访问控制

切片：

### Task39.1：附件访问 API

范围：

- 新增受保护下载/预览接口。
- 根据关联账单可见性判断权限。

验收：

- private 附件非授权成员不可访问。

### Task39.2：静态路径策略

范围：

- 禁止或限制裸 `/uploads` 绕过。
- 保留历史附件路径兼容。

验收：

- 旧数据不丢。
- 新访问路径受控。

### Task39.3：附件权限测试

必须覆盖：

- `docs/prd/27-acceptance-case-matrix.md` 的 R03。

## 9. Task40：审计与系统诊断中心

切片：

### Task40.1：诊断接口

范围：

- 返回环境、数据库、schema、备份目录、上传目录、日志目录状态。
- 不返回 secret 和绝对敏感路径。
- `GET /api/admin/diagnostics` 仅 Owner 可访问，必须携带有效账本上下文。

验收：

- 可帮助定位部署问题。
- 不泄露 `JWT_SECRET`、token、password_hash。
- viewer 请求返回 403。

### Task40.2：设置页诊断面板

范围：

- 展示数据库、备份、上传、日志状态。
- 展示当前 APP_ENV 和 Cookie 策略。
- 展示最近备份、外部访问地址是否配置和诊断生成时间。

验收：

- 非技术用户能知道系统是否健康。
- 刷新诊断不暴露 DSN、真实目录或密钥。

### Task40.3：审计日志规范

范围：

- 统一高风险操作审计字段。
- 可选只读查询接口。
- 当前基础诊断返回 `audit_action_count`，用于确认高风险操作审计是否有数据写入。

验收：

- 修改金额、删除、结算、备份、恢复、导入提交、归档写审计。
- 只读审计明细查询不进入短期冻结范围，后续如进入运维后台再单独设计权限和脱敏。

## 10. 冻结标准

Foundation before v1.1 冻结前必须满足：

- Task33 至少完成 LedgerContext 解析和 viewer 写入拒绝。
- Task34 至少完成核心 API inventory 和 OpenAPI 草案。
- Task35 完成元数据归档恢复。
- Task36 完成 ledger scoped query key。
- Task37 完成设置页分区。
- Task38 能证明核心回归测试可运行。
- Task39 关闭 private 附件裸访问风险。
- Task40 提供基础诊断能力。

当前状态：以上冻结标准已满足，短期计划标记完成。中期 v1.1 开发从 Task44 开始。
