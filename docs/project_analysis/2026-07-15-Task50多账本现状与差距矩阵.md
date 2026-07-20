# Task50 多账本正式化现状与差距矩阵

状态：Task50P.1 已完成

盘点日期：2026-07-15

盘点范围：当前 `main` 代码、migration 001-019、OpenAPI/API Inventory、前端账本状态、RBAC 与隔离回归
边界：本文只记录事实和差距，不授权 Task50 代码或 migration 变更

## 1. 结论

当前项目已经具备“多账本技术地基”，但尚未形成“多账本产品闭环”。创建账本、成员关系、显式 `X-Ledger-Id`、账本级查询键和主要数据表的 `ledger_id` 已存在；生命周期、成员不变量、显式上下文强制、当前账本失效回退、全模块隔离证明和可恢复发布方案仍不完整。

Task50 不能从新增页面开始，应按“冻结产品规则 -> 统一上下文与生命周期 Guard -> 事务化成员不变量 -> 前端失效回退 -> 全模块隔离验收”的顺序实施。现有入口只能作为兼容基线，不能视为正式完成。

## 2. 状态定义

| 状态 | 定义 |
|---|---|
| 已实现 | 已有代码、数据模型和测试共同证明，可作为 Task50 基线保留 |
| 部分实现 | 存在入口或局部隔离，但缺少生命周期、不变量、统一契约或完整测试 |
| 缺失 | 当前没有可用实现，Task50 必须新增 |
| 历史兼容 | 为单账本/Demo 兼容保留，Task50 必须迁移或明确退场 |

## 3. 能力矩阵

| 能力 | 状态 | 当前证据 | 差距与风险 | 建议切片 |
|---|---|---|---|---|
| 账本实体 | 部分实现 | `migrations/001_init.sql`、`ledger/model.go` 仅有 id/name/time | 无 status、archived_at、archived_by、版本/并发字段 | 50.1 |
| 创建账本 | 部分实现 | `ledger.Service.CreateLedger` 与 repository 在同一事务创建 owner | 仅校验非空；未 trim、限长、审计、重复策略和初始化失败契约 | 50.1/50.3 |
| 列出账本 | 部分实现 | 按成员关系列出，`created_at ASC` | 无 active/archived 筛选、分页、最近使用、状态与失效原因 | 50.3/50.4 |
| 重命名账本 | 缺失 | RolePolicy 有 `rename_ledger`，无 repository/API/UI | 权限声明与实际能力断裂 | 50.3/50.5 |
| 归档/恢复 | 缺失 | 无字段、API、统一只读 Guard | 归档后仍可通过任一业务 API 写入；无恢复闭环 | 50.1-50.5 |
| 永久删除账本 | 缺失且不应新增 | 当前无 API | 若直接删除会触发大量级联和审计/备份风险；PRD 冻结为不支持 | 非目标 |
| 成员关系 | 已实现 | migration 007 创建 `(ledger_id,user_id)` 主键；owner/editor/viewer | 表级唯一关系可保留 | 基线 |
| 旧数据 owner 补值 | 部分实现 | migration 007 全用户回填 editor；009 每账本补首个 owner | 历史回填把所有用户加入所有账本，只适合旧 Demo；Task50 升级前要审计异常成员数 | 50.1 |
| 查看成员 | 已实现 | 所有成员均可查看；UI-FL-07 已验收 | 需补归档态和成员离开后的 403/404 契约 | 50.3 |
| 添加成员 | 部分实现 | Owner 可按已存在 username 直接添加 editor/viewer | 无双人上限、并发保护、历史可见性提示、审计；当前可无限添加 | 50.3 |
| 修改角色 | 部分实现 | Owner 可把非本人目标改为 editor/viewer | 不检查目标存在/RowsAffected；另一 owner 可被降级，缺少最后 owner 不变量和原子移交 | 50.3 |
| 移除成员 | 部分实现 | Owner 可移除其他成员 | 不检查目标存在/RowsAffected；无历史数据归属说明、审计、当前账本失效通知 | 50.3/50.4 |
| 自行离开 | 缺失 | 通用移除明确禁止移除自己 | Editor/Viewer 无离开能力；Owner 移交后离开无闭环 | 50.3 |
| Owner 移交 | 缺失 | 添加/改角色接口不接受 owner | 无原子“目标升 Owner + 原 Owner 降 Editor”及并发冲突处理 | 50.3 |
| 角色策略中心 | 部分实现 | `ledger/rbac.go` 定义 RolePolicy | 部分 handler/service 直接比较角色，策略未覆盖导入、归档等新操作；运行行为与表格存在漂移 | 50.2/50.3 |
| 导出权限 | 契约冲突 | RolePolicy 未授予 Editor；`safety_test.go` 和 UI-FL-07 验收允许 Owner/Editor 导出其可见数据 | 文档/策略与可执行回归不一致；Task50 PRD 冻结为保留 Owner/Editor，并在技术评审统一 | 50P.2/50P.3 |
| 导入权限 | 已实现但 UI 入口漂移 | importer service/路由验收为 Owner-only；ImportPage 自身阻止非 Owner | SettingsPage 仍向 Editor 展示导入入口，进入后才被拦截 | UI-FL-08 |
| LedgerContext 解析 | 部分实现 | 显式 header 时验证成员并注入 user/ledger/role | header 缺失时 middleware 直接放行，无法作为统一隔离边界 | 50.2 |
| 缺省账本 fallback | 历史兼容 | dashboard/transaction/settlement/reports/safety/shared repo 与 auth `/me` 多处 `LIMIT 1` | 多账本下选择不确定，可能把无 header 请求静默落到错误账本 | 50.2 |
| API header 契约 | 历史兼容 | API Inventory 明示多数接口 optional；OpenAPI 大量使用 `LedgerIdHeaderOptional` | Task50 正式业务接口必须显式账本，不得依赖第一个成员关系 | 50P.4/50.2 |
| 跨账本对象访问 | 部分实现 | Router RBAC、transaction/settlement/report/safety/importer 有隔离测试 | 测试分散；缺少统一 A 账本对象 ID 在 B 账本读/写/附件/导入/导出的矩阵 | 50P.4/50.6 |
| 前端 active ledger | 部分实现 | Zustand persist 保存 id/role；API client 有 id 时加 header | role 可陈旧；无状态、最近使用时间、失效原因或服务端偏好 | 50.4 |
| 默认选择与失效回退 | 历史兼容 | AppShell 在当前 id 不存在时选择列表第一项 | 受 `created_at ASC` 影响；归档、移除、无账本时没有确定性产品状态 | 50.4 |
| Query key 隔离 | 已实现 | 大部分业务 query key 显式包含 ledger id，并有 `no-active-ledger` 哨兵 | `ledgers.all` 是合理全局键；仍需逐路由确认 mutation 失效范围，切换目前全量 invalidate | 50.4/50.6 |
| 账本切换缓存 | 部分实现 | AppShell 切换后全量 `invalidateQueries()` | 可避免大部分陈旧读取，但成本高且无法证明请求竞态不会回写旧账本 UI | 50.4 |
| 无账本状态 | 缺失 | selector 可禁用，页面仍可能发无 header 请求 | 会触发历史 fallback 或错误循环；需要明确 no-ledger shell 和创建入口 | 50.4/50.5 |
| 交易/结算/报表 | 部分实现 | 主表有 `ledger_id`，主要 service 优先读取 LedgerContext | 仍保留 `LIMIT 1` fallback；归档态没有统一写阻断 | 50.2/50.6 |
| 分摊与标签关联表 | 部分实现 | 通过 transaction 父记录间接隔离 | 子查询必须始终 join/验证父交易；需跨账本伪造 ID 回归 | 50.6 |
| 元数据/模板/周期规则 | 部分实现 | 主表均有 ledger_id，已有归档和部分 RBAC | 账本归档后所有 mutation 必须统一拒绝；周期确认不得越过归档 Guard | 50.2/50.6 |
| 导入批次/规则/去重引用 | 部分实现 | import_batches/rules/transaction_import_refs 均带 ledger_id，Task49/49X 已有事务与去重测试 | 待确认 ready 批次与账本归档的关系；归档后 preview/update/commit 均须拒绝 | 50.2/50.6 |
| 附件 | 部分实现 | 通过可见交易校验下载，私密附件裸路径已被回归阻断 | 需增加跨账本同名文件、归档只读下载、成员替换后的历史可见性测试 | 50.6 |
| CSV/JSON 导出 | 部分实现 | 交易/元数据/结算/审计按当前 ledger 与可见性过滤 | JSON 仍导出全局 users 和 app_settings；需定义并修正“当前账本数据包”边界 | 50P.3/50.6 |
| 物理备份/恢复 | 已实现但存在正式化阻断 | SQLite `VACUUM INTO` 备份整个数据库；当前只校验所选账本 Owner | 任一账本 Owner 都可能取得其他账本数据，正式多账本下构成越权；必须改为独立实例管理员能力，不能声称单账本备份 | 50P.2/50P.3 |
| 审计 | 部分实现 | 金额、删除、导入、导出、备份等已有 audit_logs | 创建/重命名/归档/恢复/成员添加/角色移交/离开缺少统一事件 | 50.3 |
| 数据库隔离 | 已实现 | 开发、staging、production 物理隔离文档与 v1.2 发布门禁已建立 | Task50 migration 只能在独立 development/staging 副本验证，不能混用 v1.2 生产库 | 50P.6/50.6 |
| Migration 回滚 | 缺失 | 尚无 Task50 migration | 需冻结新增编号、补值、索引、升级前检查、应用回退兼容窗口；不得修改历史 migration | 50P.3/50.1 |
| OpenAPI/Fixture | 部分实现 | 基础 ledger/member API 已列入 OpenAPI；现有隔离 fixture 分散在测试 | 缺 lifecycle、transfer/leave、409 并发、archived/no-ledger 错误和完整 fixture | 50P.4 |
| Fresh Light UI | 部分实现 | UI-FL-02 selector、UI-FL-07 settings/member 已完成 | Task50 新状态归 Task50，不得回写 UI-FL 既有业务；需 375/390/430/1440 全状态 Frame | 50P.5/50.5 |

## 4. 关键矛盾与冻结方向

### 4.1 “已有多账本”不等于“可正式发布多账本”

当前创建、切换和成员入口可以演示，但没有 lifecycle 和最后 Owner 不变量。若直接开放，会出现归档后仍写入、成员无限扩张、当前账本静默切错和 Owner 丢失等不可接受状态。

### 4.2 `X-Ledger-Id` 必须从可选兼容变为业务接口强制

Task50 正式化后，受账本约束的 handler 不再允许通过用户第一个成员关系决定账本。登录、健康检查、账本列表/创建等全局端点可不带 header；交易、结算、报表、导入、元数据、附件、导出和账本内运维端点必须显式携带并解析 LedgerContext。

### 4.3 导出与备份不是同一个产品能力

- CSV/JSON 是当前角色可见数据导出，PRD 保留 Owner/Editor 权限。
- SQLite 备份/恢复是整库运维动作，必须由独立于账本角色的实例管理员执行；任一账本 Owner 均可执行的现状不能进入正式多账本版本。
- JSON 中全局 users/app_settings 与“当前账本数据包”文案冲突，Task50 技术评审必须收窄或明确独立全局导出类型。

### 4.4 Task50 继续保持双人账本

现有代码虽然能添加第三名，但分摊、`partner_readable` 和结算仍以双人场景为核心。Task50 冻结每个账本最多两名活跃成员；多人分摊、第三名成员和邀请通知留给 Task51 及后续独立评审。

## 5. Task50 实施顺序建议

1. **50.1 数据模型**：新增 lifecycle migration、升级前数据审计和兼容读取。
2. **50.2 统一 Guard**：收紧显式 LedgerContext，建立 archived read-only 和对象归属校验。
3. **50.3 生命周期/成员 API**：事务化 rename/archive/restore/transfer/leave 与最后 Owner 不变量。
4. **50.4 前端状态**：确定性 active ledger 回退、无账本状态、定向 cache 失效和竞态保护。
5. **50.5 Fresh Light UI**：账本管理、历史可见性警告、归档/恢复和角色状态。
6. **50.6 全模块验收**：导入、附件、周期规则、模板、结算、报表、导出、备份、migration 和回滚。

## 6. Task50P.1 完成判定

- 已覆盖数据模型、API/service、RBAC、上下文、前端状态、缓存、全业务模块、数据安全、测试和部署。
- 已明确“已实现/部分实现/缺失/历史兼容”，没有把入口当成闭环。
- 已登记导出权限、导入 UI 入口、JSON 全局数据和 header fallback 四项高优先级契约漂移。
- 下一步进入 Task50P.2 产品范围冻结；仍禁止 Task50 代码和 migration。
