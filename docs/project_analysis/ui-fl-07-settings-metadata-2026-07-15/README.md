# UI-FL-07 设置与元数据收口验收

日期：2026-07-15  
状态：通过  
范围：设置页、账本成员、分类/标签/支付账户、导入导出入口、备份恢复和系统诊断；本机 WSL2 staging；未访问 NAS

## 1. 结论

UI-FL-07 已完成开发与收口。设置页从 Dark Glass 长页面重组为 Fresh Light 六分区工作台；元数据页统一搜索、活跃/归档筛选、排序、引用数量、颜色标识和生命周期操作。高风险操作均复用共享确认组件，并明确“操作会改变什么、不会改变什么”。

实现提交：

```text
912523c  feat(settings): 完成 UI-FL-07 设置与元数据可信流程
```

## 2. 权限与业务边界

- 所有账本成员可查看成员名单；只有 Owner 显示添加成员、角色调整和移除入口，后端继续作为最终权限边界。
- Owner 可管理分类、标签和支付账户；Editor/Viewer 保持只读。归档只影响新账单选择器，历史引用继续显示原名称。
- Owner/Editor 保留既有“导出当前角色可见数据”能力；Viewer 不显示导出动作。备份、恢复准备和诊断仍为 Owner-only。
- 导入继续支持微信 CSV/XLSX、支付宝 CSV；preview 不写正式账单。
- 恢复按钮更名为“准备恢复”：服务端只创建前置安全备份并返回停机人工替换指引，不在线覆盖数据库。
- 未新增 API、DTO、依赖或 migration，未实现 Task50 的账本归档、恢复和无账本状态。

## 3. 自动化门禁

```text
npm run lint
结果：通过

npm run test
结果：22 个测试文件、82 个测试通过

npm run build
结果：通过；保留既有主包大于 500 kB 告警

CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go test ./...
结果：全部通过
```

新增前端契约测试覆盖共享 Button/ConfirmDialog/SegmentedControl、角色入口、元数据归档历史语义、无内联暗色样式和 1024/768/430px 响应式边界。Go 全量回归继续覆盖元数据归档/恢复、RBAC、备份、诊断和导出隐私。

## 4. 本机部署

部署前在线备份：

```text
backups/predeploy/ui-fl-07-20260715-151643/ledger.db
quick_check=ok
users|ledgers|transactions|settlements=2|2|40|2
```

运行结果：

```text
URL=http://localhost:38088
image=ledger-two:1.2.0-rc-ui-fl-07
deployment_channel=staging
schema_version=19
import_xlsx_enabled=true
db=ok
```

本次只向 UI-FL-06 固定镜像注入新的 production 前端静态资源，后端二进制和 schema 未改变。该镜像用于本机 staging 验收，不替代 NAS 标准发布构建。

## 5. 浏览器验收

真实 `userA` QA 账号仅执行登录、读取、打开归档确认并取消，没有创建账本、调整成员、导出、备份或修改元数据。

```text
设置页 1440px：innerWidth=1440，scrollWidth=1440
设置页 375px：innerWidth=375，scrollWidth=375
元数据 1440px：innerWidth=1440，scrollWidth=1440
元数据 375px：innerWidth=375，scrollWidth=375
theme=fresh-light
成员名单=2
六个设置分区=visible
元数据列表=18，活跃/归档筛选、引用数量和颜色标识=visible
归档确认=明确历史引用保留，取消后未写入
```

验收后主数据库：

```text
quick_check=ok
users=2
ledgers=2
transactions=40
settlements=2
categories active|archived=13|5
```

## 6. 视觉证据

- `settings-fresh-light-1440.png`：桌面六分区设置导航、成员列表和 Owner 管理入口。
- `settings-fresh-light-375.png`：移动端账号摘要、两列设置分组和首个业务分区，无横向溢出。
- `metadata-fresh-light-1440.png`：桌面编辑器、状态筛选、排序、颜色和历史引用列表。
- `metadata-fresh-light-375.png`：移动端单列编辑器、稳定控件尺寸和底部导航边界。

## 7. 后续边界

1. UI-FL-07 门禁关闭，波次 C 完成；下一任务为 UI-FL-08 导入工作台。
2. UI-FL-08 只能迁移 Task47U/48U/49U/49XU 已冻结交互，不得修改 parser、hash、批次状态、commit 事务或规则优先级。
3. Task50 继续执行 P.1/P.2 文档准备；正式代码仍等待 UI-FL-10 和开发准入评审完成。
