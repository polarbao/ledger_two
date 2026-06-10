# Demo 后续 AI 编码任务拆分与提示词模板

## 1. 使用原则

当前 LedgerTwo 已完成 Demo 版本，并能进行本地前后端联调。后续不要让 Codex 一次性实现整个系统，应继续采用“小步任务 + 明确边界 + 测试验收”的方式推进。

每个任务必须要求 Codex：

1. 先阅读本任务相关文档。
2. 输出实现计划，说明会改哪些文件。
3. 只实现当前任务，不扩展任务外功能。
4. 运行测试或构建命令。
5. 输出变更摘要、风险和下一步建议。

通用约束：

- 金额使用整数分，禁止 float。
- 后端是结算和统计的唯一可信来源。
- 结算只生成 settlement 记录，不修改历史账单。
- 删除必须 soft delete。
- private 账单不能被对方通过 API 获取。
- 真实数据库、备份、上传文件、`.env` 不得提交到 Git。
- 优先保证数据安全，再做体验增强。

## 2. 阶段路线

| 阶段 | 目标 | 说明 |
|---|---|---|
| v0.2 | 稳定真实可用 | 错误码、安全、审计日志、备份、统计口径、测试、NAS 部署 |
| v0.3 | 提升录入效率 | 快捷记账、复制一笔、模板、周期账单 |
| v0.4 | 数据导入与附件 | CSV 导入、导入预览、去重、规则、附件小票 |
| v0.5 | 家庭账本扩展 | 多账本、多成员、角色权限、多人分摊和结算 |
| v0.6 | 跨端体验增强 | PWA、移动端优化、缓存、离线只读/草稿 |
| v1.0 | 长期稳定发布 | CI、迁移、发布、备份恢复、运维文档 |

## 3. 推荐任务顺序

## v0.2：稳定真实可用版

### Task 01：Demo 现状审计与回归基线

```text
你是 LedgerTwo 项目的高级全栈工程师。
当前项目已完成 Demo 并可以本地前后端联调。请先阅读：
- docs/README.md
- docs/prd/00-product-roadmap.md
- docs/tech/09-test-quality.md
- docs/reviews/prd-tech-review-2026-06-11.md

任务目标：
1. 检查当前代码结构、已有 API、前端页面、测试和 Docker 配置。
2. 输出当前 Demo 已实现功能清单。
3. 输出缺失项、风险项和 v0.2 优先级。
4. 不要修改业务代码，除非发现明显 README 或文档路径错误。

验收标准：
- 给出 backend/frontend/database/deploy 四部分现状说明。
- 给出可运行命令，例如 go test、pnpm build、docker compose。
- 明确下一步应执行 Task 02。

完成后按统一格式输出：完成内容、修改文件、验证命令、未完成/风险、下一步建议。
```

### Task 02：统一错误码与 API 错误响应

```text
请先阅读：
- docs/tech/10-error-codes.md
- docs/tech/02-backend-modules.md
- docs/tech/04-database-api.md
- docs/ui/09-empty-error-loading-states.md

任务目标：
1. 在后端实现统一 API 错误结构。
2. 定义 APIError/AppError 类型和错误码常量。
3. 统一 handler 的错误响应，不再各自拼接错误 JSON。
4. 前端 api/client.ts 统一识别 success=false 和 error.code。
5. 映射未登录、无权限、校验失败、资源不存在、内部错误。

禁止事项：
- 不要改变已有 API 路由语义。
- 不要新增业务功能。
- 不要把内部错误堆栈返回给前端。

测试要求：
- 后端增加错误响应单元测试或集成测试。
- 前端至少覆盖 API client 错误解析逻辑。
- 运行 go test ./... 和前端构建/测试命令。

验收标准：
- 所有失败响应符合 {success:false,error:{code,message,details}}。
- 未登录返回 UNAUTHORIZED。
- 表单校验返回 VALIDATION_ERROR 或具体业务错误码。
```

### Task 03：认证、安全与权限加固

```text
请先阅读：
- docs/tech/11-security-auth.md
- docs/prd/01-ledger-member.md
- docs/prd/02-transaction.md
- docs/tech/10-error-codes.md

任务目标：
1. 检查并加固登录、退出、me、Session Cookie。
2. 确保 Cookie 使用 HttpOnly、SameSite，生产环境支持 Secure。
3. 所有业务 API 默认需要登录。
4. private/partner_readable/shared 权限在后端严格判断。
5. 不可见资源优先返回 NOT_FOUND，避免暴露资源存在性。

禁止事项：
- 不要实现 OAuth。
- 不要实现复杂 Token 体系。
- 不要开放公开注册。

测试要求：
- 未登录访问业务 API 返回 401。
- private 账单对方无法查询。
- partner_readable 对方可见但不能编辑。
- shared 账双方可见。

验收标准：
- 权限判断在后端完成，不依赖前端隐藏。
- 敏感信息不写入日志。
- .env、数据库、备份、上传文件不会被提交。
```

### Task 04：审计日志完整化

```text
请先阅读：
- docs/tech/11-security-auth.md
- docs/prd/02-transaction.md
- docs/prd/03-shared-split-settlement.md
- docs/ui/11-data-safety-confirmations.md

任务目标：
1. 完善 audit_logs 写入逻辑。
2. 修改金额、修改分摊、删除账单、生成结算、导出数据、手动备份必须写审计日志。
3. 审计日志记录 actor_user_id、action、entity_type、entity_id、before、after、created_at。
4. 不在 audit_logs 中保存密码、Session、密钥。

禁止事项：
- 不要做复杂审计日志 UI，最多保留后端查询接口或内部能力。
- 不要记录敏感明文。

测试要求：
- 修改账单写日志。
- 删除账单写日志。
- 创建结算写日志。
- 导出/备份操作如果已实现，也写日志。

验收标准：
- 高风险操作均可追踪。
- 失败操作不应伪造成功审计。
```

### Task 05：备份、导出与数据安全

```text
请先阅读：
- docs/prd/06-import-export.md
- docs/tech/06-import-export-backup.md
- docs/tech/08-nas-deployment.md
- docs/ui/11-data-safety-confirmations.md

任务目标：
1. 实现手动 SQLite 备份。
2. 实现备份列表查询。
3. 实现 CSV 导出和 JSON 全量导出。
4. JSON 导出不得包含明文密码、Session token、SESSION_SECRET。
5. 备份失败时返回明确错误码。

推荐 API：
- POST /api/admin/backup
- GET /api/admin/backups
- GET /api/export/transactions.csv
- GET /api/export/full.json

禁止事项：
- v0.2 不做 UI 恢复备份。
- 不要在写入中直接复制 SQLite 文件，优先使用 VACUUM INTO 或安全备份方式。

测试要求：
- 备份文件能生成。
- CSV 字段正确。
- JSON 不包含敏感字段。
- 备份目录不可写时返回 BACKUP_FAILED 或 BACKUP_PATH_INVALID。

验收标准：
- 真实账务数据可以导出和备份。
- NAS 部署时 data/backups/uploads 路径清晰。
```

### Task 06：结算算法统一与回归测试

```text
请先阅读：
- docs/tech/05-settlement-algorithm.md
- docs/prd/03-shared-split-settlement.md
- docs/tech/12-statistics-caliber.md
- docs/tech/09-test-quality.md

任务目标：
1. 检查当前结算算法是否统一使用推荐公式：
   raw_net = paid_amount - share_amount
   settlement_net = received_settlement - paid_settlement
   final_net = raw_net - settlement_net
2. 修正任何方向相反或命名混乱的实现。
3. 补齐结算核心单元测试和集成测试。
4. 确认 settlement 不修改历史 shared_expense。

测试场景：
- A 支付 200，两人平摊，B 欠 A 100。
- B 支付 80，两人平摊，合并后 B 欠 A 60。
- B 向 A 结算 60 后双方结清。
- 100.01 元奇数分平摊，多出的 1 分由付款人承担。
- 删除共同支出后净额重新计算。

验收标准：
- 结算方向、金额、正负号稳定。
- 前端不自行计算最终结算金额。
```

### Task 07：统计口径落地与报表回归

```text
请先阅读：
- docs/tech/12-statistics-caliber.md
- docs/prd/05-analytics-report.md
- docs/tech/04-database-api.md
- docs/ui/06-analytics.md

任务目标：
1. 按统计口径统一 Dashboard 和 reports API。
2. settlement 不进入消费统计。
3. deleted 不进入任何默认统计。
4. expense/shared_expense 进入支出统计。
5. income 只进入收入统计。
6. 成员统计必须区分 paid_amount、share_amount、raw_net、settlement_paid、settlement_received、final_net。

禁止事项：
- 不要让前端自行聚合最终金额。
- 不要把 transfer、settlement 当作消费。

测试要求：
- Dashboard 与 reports 的月度总额一致。
- 删除账单后统计更新。
- shared_expense 统计正确。
- 标签多选统计按文档口径实现。

验收标准：
- 统计 API 口径一致。
- UI 文案不会把付款金额误称为消费承担金额。
```

### Task 08：前端状态、错误态与高风险确认统一

```text
请先阅读：
- docs/ui/09-empty-error-loading-states.md
- docs/ui/10-design-system.md
- docs/ui/11-data-safety-confirmations.md
- docs/tech/10-error-codes.md

任务目标：
1. 抽象 EmptyState、ErrorState、LoadingSpinner、SkeletonCard、SkeletonTable、PageState。
2. 前端统一展示 loading、empty、error、unauthorized、forbidden、offline 状态。
3. 删除账单、删除共同支出、生成结算、导出、备份必须有确认弹窗。
4. 危险按钮使用统一 danger 样式。

禁止事项：
- 不要重做整体 UI。
- 不要修改后端业务逻辑。

验收标准：
- 所有主页面无空白加载。
- 无数据时有空状态。
- API 失败有重试入口。
- 高风险操作必须二次确认。
- 移动端确认弹窗不横向溢出。
```

### Task 09：Docker Compose 与 NAS 部署验证

```text
请先阅读：
- docs/tech/08-nas-deployment.md
- docs/tech/06-import-export-backup.md
- docs/06_NAS_DEPLOYMENT.md，如果存在

任务目标：
1. 完善 Dockerfile 和 docker-compose.yml。
2. 明确挂载 data、backups、uploads、logs。
3. 增加 healthcheck 或 /api/healthz 检查说明。
4. 确保 docker compose up -d --build 后可访问前端和 API。
5. 更新 .env.example，不提交真实 .env。

测试要求：
- 本地 docker compose up -d --build 成功。
- 容器重启后 SQLite 数据不丢。
- 手动备份能写入 backups。
- /api/healthz 正常。

验收标准：
- NAS 部署路径清晰。
- 数据不会写入容器临时层。
```

### Task 10：CI、测试与质量门禁

```text
请先阅读：
- docs/tech/09-test-quality.md
- docs/tech/10-error-codes.md
- docs/tech/12-statistics-caliber.md

任务目标：
1. 增加或完善测试命令。
2. 增加 GitHub Actions，至少包含 backend test、frontend build/lint、docker build，可按项目实际情况裁剪。
3. 增加核心业务测试：权限、结算、统计、soft delete。
4. 更新 README 或开发文档中的验证命令。

禁止事项：
- 不要为了让 CI 过而删除核心测试。
- 不要提交真实数据库和密钥。

验收标准：
- go test ./... 通过。
- 前端构建通过。
- 核心结算和权限测试稳定。
```

## v0.3：录入体验增强版

### Task 11：复制一笔

```text
请先阅读：
- docs/prd/02-transaction.md
- docs/ui/03-transactions.md
- docs/ui/04-transaction-form.md
- docs/tech/03-frontend-modules.md

任务目标：
1. 在账单详情中增加“复制一笔”。
2. 复制时带入类型、金额、分类、账户、付款人、参与人、分摊方式、标签、备注。
3. 默认 occurred_at 使用当前时间。
4. 用户确认后创建新账单，不修改原账单。

验收标准：
- 普通账单可复制。
- 共同支出可复制且 split 正确。
- 复制后刷新流水和 Dashboard。
```

### Task 12：快捷记账默认值

```text
请先阅读：
- docs/prd/02-transaction.md
- docs/ui/04-transaction-form.md
- docs/ui/10-design-system.md

任务目标：
1. 记一笔表单支持最近分类、最近账户、最近标签。
2. 支持默认付款人。
3. 支持保存并继续记。
4. 移动端优化金额输入体验。

禁止事项：
- 不要做复杂 AI 自动分类。
- 不要改动结算算法。

验收标准：
- 高频记账步骤减少。
- 保存并继续记不会清空必要默认项。
```

### Task 13：账单模板

```text
请先阅读：
- docs/prd/08-budget-reminder.md
- docs/ui/04-transaction-form.md
- docs/tech/04-database-api.md

任务目标：
1. 支持创建账单模板。
2. 模板包含标题、类型、金额可选、分类、账户、付款人、参与人、分摊方式、标签、备注。
3. 用户可从模板快速生成账单。
4. 模板不等于真实账单，不进入统计和结算。

验收标准：
- 可以从模板生成普通支出。
- 可以从模板生成共同支出。
- 删除模板不影响历史账单。
```

### Task 14：周期账单提醒

```text
请先阅读：
- docs/prd/08-budget-reminder.md
- docs/ui/07-settings.md
- docs/ui/09-empty-error-loading-states.md

任务目标：
1. 支持周期账单规则：每周、每月、每年。
2. 到期后生成待确认提醒。
3. 用户确认后才生成真实账单。
4. 初期不自动扣账，不自动创建真实账单。

禁止事项：
- 不要实现复杂通知推送。
- 不要实现后台自动扣款。

验收标准：
- 到期提醒可见。
- 确认后生成账单。
- 取消提醒不影响历史账单。
```

### Task 15：高级筛选与批量标签

```text
请先阅读：
- docs/prd/02-transaction.md
- docs/prd/04-category-tag-account.md
- docs/ui/03-transactions.md
- docs/ui/11-data-safety-confirmations.md

任务目标：
1. 流水页支持金额区间、成员、类型、分类、标签、可见性、关键词筛选。
2. 支持清空筛选。
3. 后续可选：批量打标签。
4. 批量操作必须有确认或撤销策略。

禁止事项：
- 不要先做批量删除。
- 不要改变统计口径。

验收标准：
- 筛选参数可反映在 URL 或前端状态中。
- 移动端筛选使用底部 Sheet。
```

## v0.4：导入、导出、附件版

### Task 16：CSV 导入基础与预览

```text
请先阅读：
- docs/prd/06-import-export.md
- docs/tech/06-import-export-backup.md
- docs/ui/11-data-safety-confirmations.md

任务目标：
1. 实现 CSV 上传和解析。
2. 先展示导入预览，不直接写入数据库。
3. 支持字段映射：时间、金额、标题、商户、分类、账户、备注。
4. 支持导入前取消。

禁止事项：
- 不要一上传就写入正式账单。
- 不要做银行自动同步。

验收标准：
- CSV 可解析为预览列表。
- 格式错误有明确错误态。
- 预览不会影响统计。
```

### Task 17：导入去重与确认写入

```text
请先阅读：
- docs/tech/06-import-export-backup.md
- docs/prd/06-import-export.md
- docs/ui/11-data-safety-confirmations.md

任务目标：
1. 实现 import_batches、import_items 或等价结构。
2. 生成 import_hash 防止重复导入。
3. 导入确认前展示待导入、重复跳过、需要人工确认数量。
4. 确认后批量写入 transactions。

测试要求：
- 同一文件重复导入不会生成重复账单。
- 导入失败可回滚当前批次。

验收标准：
- 导入过程可追踪。
- 用户确认前不写入正式数据。
```

### Task 18：导入分类规则

```text
请先阅读：
- docs/prd/06-import-export.md
- docs/tech/06-import-export-backup.md
- docs/prd/04-category-tag-account.md

任务目标：
1. 支持基于商户/描述关键词的分类规则。
2. 规则可设置分类、标签、账户。
3. 导入预览中展示规则命中结果。
4. 用户确认后写入。

禁止事项：
- 不要做复杂机器学习分类。
- 不要自动覆盖用户手动修改的预览字段。

验收标准：
- 星巴克 -> 餐饮/咖啡 这类规则可命中。
- 用户可以在预览中调整分类。
```

### Task 19：附件与小票上传

```text
请先阅读：
- docs/prd/07-attachment-receipt.md
- docs/tech/06-import-export-backup.md
- docs/tech/08-nas-deployment.md
- docs/ui/03-transactions.md

任务目标：
1. 支持账单上传图片附件。
2. 附件存储到 uploads 目录，不存入数据库 BLOB。
3. 账单详情页可查看附件。
4. 删除账单时附件进入不可见或软删除状态。

禁止事项：
- v0.4 不做 OCR。
- 不要将附件提交到 Git。

验收标准：
- 附件可上传、查看、删除。
- NAS 部署 uploads 目录挂载正确。
- 备份说明包含 uploads。
```

## v0.5：家庭账本版

### Task 20：default ledger 迁移

```text
请先阅读：
- docs/prd/01-ledger-member.md
- docs/tech/04-database-api.md
- docs/reviews/prd-tech-review-2026-06-11.md

任务目标：
1. 引入 ledgers 和 ledger_members 表。
2. 为当前 Demo 数据创建 default ledger。
3. transactions、categories、tags、accounts 等按需要增加 ledger_id。
4. 保证迁移后当前双人账本行为不变。

禁止事项：
- 不要一次性开放多账本 UI。
- 不要破坏当前 Demo 数据。

测试要求：
- 旧数据迁移后可查询。
- 当前用户只能访问自己所属 ledger。

验收标准：
- 为 v0.5 多账本打基础。
- 单账本模式仍可用。
```

### Task 21：多账本与成员角色

```text
请先阅读：
- docs/prd/01-ledger-member.md
- docs/tech/11-security-auth.md

任务目标：
1. 支持创建账本。
2. 支持账本成员和角色：owner/editor/viewer。
3. 所有业务 API 校验 ledger membership。
4. 设置页增加成员管理入口。

禁止事项：
- 不要做公开邀请链接，除非已有明确需求。
- 不要实现复杂企业权限。

验收标准：
- A 账本数据不会出现在 B 账本。
- viewer 不能新增或编辑账单。
- owner 可管理成员。
```

### Task 22：多人分摊方式

```text
请先阅读：
- docs/prd/03-shared-split-settlement.md
- docs/tech/05-settlement-algorithm.md

任务目标：
1. 将共同支出从双人扩展到 N 人。
2. 支持 equal、amount、ratio、shares。
3. 校验分摊金额合计、比例合计和成员有效性。
4. UI 明确展示每个成员承担金额。

测试要求：
- 三人平均分摊。
- 按金额分摊合计等于总额。
- 按比例分摊合计等于 100%。
- 奇数分处理稳定。

验收标准：
- 多人分摊结果准确。
- 结算算法可消费多人 split 数据。
```

### Task 23：多人结算与最小转账建议

```text
请先阅读：
- docs/prd/03-shared-split-settlement.md
- docs/tech/05-settlement-algorithm.md

任务目标：
1. 基于多人 final_net 生成转账建议。
2. 净额为正的人应收，净额为负的人应付。
3. 通过正负队列生成较少转账次数。
4. UI 展示建议转账列表。

禁止事项：
- 不要修改历史账单。
- 不要把转账建议直接当成已结算。

验收标准：
- 三人及以上可生成正确结算建议。
- 用户确认后才生成 settlement 记录。
```

## v0.6：跨端与移动体验版

### Task 24：PWA 基础能力

```text
请先阅读：
- docs/prd/09-cross-platform.md
- docs/tech/07-cross-platform-tech.md
- docs/ui/08-mobile-pwa.md

任务目标：
1. 增加 manifest、icons、基础 service worker。
2. 支持添加到手机桌面。
3. 静态资源可缓存。
4. 离线时展示明确提示。

禁止事项：
- 不要实现离线正式记账。
- 不要实现 Push 通知。

验收标准：
- iPhone Safari / Android Chrome 可添加到桌面。
- 离线不会误导用户保存成功。
```

### Task 25：移动端响应式优化

```text
请先阅读：
- docs/ui/01-layout-navigation.md
- docs/ui/08-mobile-pwa.md
- docs/ui/10-design-system.md

任务目标：
1. 优化 375px 宽度下的 Dashboard、流水、记账、结算、设置页面。
2. 流水页使用 TransactionCard。
3. 筛选使用底部 Sheet。
4. 高风险确认适配移动端。

验收标准：
- 主要页面无横向滚动。
- 手机端可以完成记账、筛选、结算。
- 移动端按钮不误触。
```

### Task 26：本地缓存与离线只读

```text
请先阅读：
- docs/tech/07-cross-platform-tech.md
- docs/ui/09-empty-error-loading-states.md

任务目标：
1. 使用 TanStack Query 缓存最近数据。
2. 可选使用 IndexedDB 缓存分类、标签、账户、最近流水。
3. 离线时允许查看最近缓存数据。
4. 离线提交正式账单必须禁用。

验收标准：
- 离线时页面有明确状态。
- 不会产生假保存。
- 恢复网络后可刷新数据。
```

### Task 27：离线草稿

```text
请先阅读：
- docs/prd/09-cross-platform.md
- docs/tech/07-cross-platform-tech.md

任务目标：
1. 离线时允许保存账单草稿。
2. 草稿不进入正式统计和结算。
3. 网络恢复后用户手动提交。
4. 提交前重新校验金额、分类、参与人和分摊方式。

禁止事项：
- 不要自动静默同步正式账单。
- 不要实现复杂冲突合并。

验收标准：
- 离线草稿不丢失。
- 草稿提交后才进入正式数据。
```

## v1.0：长期稳定发布版

### Task 28：数据库迁移与版本发布规范

```text
请先阅读：
- docs/tech/04-database-api.md
- docs/tech/08-nas-deployment.md
- docs/tech/09-test-quality.md

任务目标：
1. 整理 migration 规范。
2. 增加数据库 schema version。
3. 发布前自动检查迁移状态。
4. 编写升级前备份说明。

验收标准：
- 新版本可从旧版本安全升级。
- 升级前有备份提示。
- 迁移失败不会静默破坏数据。
```

### Task 29：安全恢复备份 UI

```text
请先阅读：
- docs/tech/06-import-export-backup.md
- docs/ui/11-data-safety-confirmations.md

任务目标：
1. 实现备份恢复 UI。
2. 恢复前展示备份元信息。
3. 要求输入确认文本。
4. 恢复前自动备份当前数据库。
5. 恢复失败可给出明确错误。

禁止事项：
- 不要一键无确认恢复。
- 不要跳过恢复前自动备份。

验收标准：
- 恢复流程可控。
- 用户明确知道会覆盖当前数据。
```

### Task 30：发布与运维文档

```text
请先阅读：
- docs/tech/08-nas-deployment.md
- README.md
- docs/README.md

任务目标：
1. 整理 v1.0 发布说明。
2. 补齐 NAS 部署、升级、回滚、备份、恢复、日志查看文档。
3. 增加常见问题排查。
4. 增加版本变更记录。

验收标准：
- 新用户能按文档部署。
- 老用户能按文档升级。
- 数据备份和恢复路径清晰。
```

## 4. 每次开始任务的通用提示词

```text
你是 LedgerTwo 项目的高级全栈工程师。
当前项目已完成 Demo，并进入后续模块化开发阶段。

请先阅读：
- docs/README.md
- docs/prd/00-product-roadmap.md
- docs/18_POST_DEMO_AI_CODING_TASKS.md
- 本任务指定的 PRD / UI / Tech 文档

执行规则：
1. 只实现当前任务，不要实现后续任务。
2. 先输出实现计划和预计修改文件。
3. 严格遵守金额 int64 cents。
4. 后端是结算和统计的唯一可信来源。
5. 删除必须 soft delete。
6. private 账单不能泄露。
7. 高风险操作必须写审计日志或二次确认。
8. 完成后运行测试和构建。
9. 输出完成内容、修改文件、验证命令、未完成/风险、下一步建议。
```

## 5. 后端任务补充约束

```text
后端要求：
- Go 1.22+
- SQLite
- REST JSON
- 金额用 int64 分
- handler 只处理 HTTP
- service 处理业务规则
- repository 处理数据库访问
- 统一错误响应结构
- 错误码使用 docs/tech/10-error-codes.md
- 权限规则必须测试
- 结算和统计不得在前端实现最终计算
```

## 6. 前端任务补充约束

```text
前端要求：
- React + TypeScript + Vite
- TanStack Query 管服务端状态
- Zustand 只管 UI 状态
- React Hook Form + Zod 做表单
- 金额输入元，API 提交分
- 遵守 docs/ui/10-design-system.md
- 所有页面覆盖 loading / empty / error 状态
- 危险操作遵守 docs/ui/11-data-safety-confirmations.md
- 移动端优先，桌面端增强
```

## 7. AI 输出验收格式

每次 AI 完成任务，必须输出：

```text
完成内容：
- ...

修改文件：
- ...

验证命令：
- ...

未完成/风险：
- ...

下一步建议：
- ...
```

## 8. Git 提交建议

每个任务完成后单独提交。

提交信息建议：

```text
feat: implement backup export
fix: unify settlement calculation
test: add permission regression tests
docs: update deployment guide
```

不要把多个阶段任务混在一个提交中。
