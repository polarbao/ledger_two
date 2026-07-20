# Task50.5 多账本管理与归档只读验收

状态：已完成代码、自动化与本地 development 浏览器验收<br>
日期：2026-07-16<br>
目标版本：v1.3 / schema 21<br>
前置任务：Task50.1-Task50.4 已完成

## 1. 验收结论

Task50.5 已按 PRD 31、Tech 25、UI 16、Fixture 32 和本地 Task50 Figma handoff 完成。当前已具备：

1. `/settings/ledgers` 活跃/已归档列表、创建、切换和统一管理入口。
2. `/settings/ledgers/:ledgerId` 账本详情、重命名、归档、恢复、成员角色、Owner 移交、移除和离开流程。
3. ready 导入批次阻断归档时的显式处理入口，以及导入工作台“放弃预览”闭环。
4. 首页、流水、分析和结算的临时归档账本只读上下文；归档账本不会写入 active-ledger 或最近使用偏好。
5. Owner/Editor/Viewer、实例管理员、无活跃账本、离线、冲突和访问失效状态。
6. 375/390/430/1440、Fresh Light/Dark Glass、无横向溢出和 Dialog 焦点返回。

本任务没有新增 migration、依赖或后端业务路由，没有部署 WSL staging 或 NAS。

## 2. 实现范围

### 2.1 管理与成员

- 新增账本管理页、账本详情页、生命周期操作面板和归档只读 Banner。
- 创建成功后通过 Task50.4 的切换状态机进入新账本，不使用全局无差别缓存失效。
- 生命周期和成员写操作继续使用服务端 `If-Match`、稳定错误码和权限拒绝；前端隐藏或禁用仅用于降低误操作。
- Editor/Viewer 可离开；Owner 必须先移交所有权；成员变化不改写历史账单、分摊、结算或审计。
- Owner/Editor 可导出其可见账本数据；实例级备份和诊断仍仅对实例管理员展示。

### 2.2 归档只读上下文

- 归档历史通过 `archived_ledger_id` 显式进入，不替换持久化 active ledger。
- API client 在无 path 级 ledger ID 时优先使用临时归档上下文，并保持查询键按有效 ledger ID 隔离。
- 归档页隐藏记账、草稿、导入和周期规则等写入口，保留首页、流水、分析、结算和允许角色的数据导出。
- 返回活跃账本或恢复账本后清理临时归档上下文。

### 2.3 验收中关闭的兼容缺陷

1. 空账本 Dashboard 的集合字段原先可能序列化为 `null`，会使旧持久化缓存触发前端 `.length` 异常。后端现统一输出空数组，前端同时兼容清洗旧缓存。
2. 流水页旧持久化分类缓存可能为 `null`，会在 `.reduce` 时中断渲染。流水页现先归一化分类集合，再传给筛选和展示组件。
3. 从归档历史直接跳转到不带 `archived_ledger_id` 的活跃路由时，旧临时上下文可能在一个渲染周期内残留。AppShell 现先阻止活跃业务 Outlet 和写组件挂载，清理后再以 active ledger 发起请求。
4. 共享 Dialog 的 `onClose` 变化和表单 `autoFocus` 曾覆盖返回焦点。`useModalSurface` 现记住弹层外触发元素、忽略弹层内焦点，并在 Escape 关闭后恢复触发按钮。

## 3. 浏览器证据

证据目录：`docs/project_analysis/evidence/task50-5/`

| 证据 | 范围 |
|---|---|
| `browser-acceptance.json` | URL、主题、视口、横向溢出、归档只读、严重日志和焦点返回断言 |
| `desktop-active-fresh.png` | 1440 Fresh Light 活跃账本管理 |
| `desktop-archived-fresh.png` | 1440 Fresh Light 已归档列表 |
| `desktop-detail-fresh.png` | 1440 Fresh Light 账本详情与成员 |
| `mobile-375-active-fresh.png` | 375 活跃账本管理 |
| `mobile-390-archived-fresh.png` | 390 已归档账本管理 |
| `mobile-430-detail-fresh.png` | 430 账本详情 |
| `mobile-active-fresh.png` | 390 活跃账本管理补充截图 |
| `desktop-active-dark.png` | 1440 Dark Glass 回退主题 |
| `desktop-archived-view-fresh.png` | 1440 归档账本首页只读上下文 |

浏览器断言结果：

- 375/390/430/1440 的 `scrollWidth` 均等于视口宽度。
- 所有记录页面均无 Router Error、无严重 Console/Runtime 异常。
- 归档首页存在只读 Banner，不存在 FAB 或快捷记账写入口。
- 创建账本 Dialog 可由 Escape 关闭，并把焦点返回创建按钮。
- 从归档历史直接进入无归档参数的活跃流水页后，所有账本内请求均使用 active ledger header，不携带旧归档账本 ID。

## 4. 自动化验证

```text
frontend: npm run test
frontend: npm run lint
frontend: npm run build
backend:  go test ./...
backend:  go vet ./...
backend:  go build -o <temporary-path> ./cmd/server
```

最终验收应以本提交前的最新命令输出为准。Vite 仍报告既有主 bundle 大于 500 kB 的非阻断警告，本任务没有扩大为前端分包重构。

## 5. 环境与数据边界

1. 浏览器验收使用本仓库可丢弃的 native development schema 21 数据库。
2. 验收数据仅包含匿名开发账号和匿名账本，不写入文档、提交信息或生产数据。
3. WSL staging 继续保持 v1.2 schema 19；NAS 未执行 migration 020/021。
4. Task50.5 回滚只需回退本任务代码并清理 development 浏览器状态，不需要回退数据库 schema。

## 6. 后续准备结论

下一任务为 Task50.6。其 PRD、技术契约、Fixture、升级和回滚框架已经具备，不需要新增平行准备文档；尚未完成的是只能在候选版本产生后生成的真实运行证据：

1. 全模块 A/B 隔离和历史可见性矩阵。
2. 匿名 schema 19 副本到 schema 21 的升级、守恒和恢复演练。
3. 独立 v1.3 staging、固定 RC 镜像、health/schema/channel 校验。
4. 备份恢复、旧镜像阻断、成对回滚和发布说明收口。

Task51P.1 仍只允许非约束性证据收集；有效真实小组证据不足，P2-P6 不得冻结。Task53P.1-P.6 准备包已经完成，但实现顺序继续等待 Task50.6 后统一评审。
