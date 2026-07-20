# Task53U Fresh Light UI 实现与视觉门禁记录

日期：2026-07-20<br>
结论：源码实现和自动化契约完成；Task53U 尚未关闭，等待隔离 schema 22 运行时的 375/390/430/1440 浏览器视觉验收

## 1. 已完成实现

| 范围 | 实现结果 |
|---|---|
| 分类摘要与筛选 | auto/suggested/fallback/manual/bulk/conflict/unresolved 摘要、筛选与双层行状态并存 |
| 行级解释 | 最终分类/账户/标签、来源、置信度、可读原因常驻展示 |
| 行编辑与学习 | 标签 `N/8`、记住商户、来源范围；行保存和学习成功/失败分别反馈 |
| 批量与重分类 | 接受建议、应用相同值、相同商户、可选长期学习、reclassify dry-run/确认 |
| 规则管理 | 精确商户、来源、auto/suggest、手工/学习分组、stale、提交命中次数和时间 |
| 默认元数据 | 新账本默认基础包/显式空白账本；既有账本 profile preview/conflict/apply |
| 兜底保护 | 归档 `expense_other/income_other` 时强制选择同类型非系统替代分类 |
| 正式契约 | 前端 DTO/API client 与正式 OpenAPI 补齐 default-profile 和 `metadata_profile` |

所有动作继续遵守“预览调整不等于提交账单”。接受建议、批量调整、学习规则和重新分类均不调用 commit。

## 2. 自动化证据

已执行：

```text
frontend npm run lint
frontend npm test -- --run
frontend npm run build
docs/api/openapi.yaml PyYAML parse + local $ref resolution
git diff --check
```

新增 `Task53UI.contract.test.ts`，覆盖持久化分类状态、显式批量操作、学习双结果、规则健康、profile 和兜底替代入口。API/model/既有页面契约测试同步扩展。

## 3. 未关闭的视觉门禁

当前本机可发现的运行时事实：

| 地址 | channel | schema | 结论 |
|---|---|---:|---|
| `http://127.0.0.1:5173` -> development API | development | 21 | 不具备 Task53 DTO/接口，不能作为 Task53U 验收 |
| `http://127.0.0.1:38091` | staging | 21 | Task50 环境，禁止混用 |
| 计划 `http://127.0.0.1:38092` | Task53 staging | 22 | 尚未启动，本轮未部署 |

根据 Task53.5 冻结门禁，38092 必须使用独立 runtime root/数据库，并在用户确认后执行。因此本轮没有把旧 schema 21 环境伪装成 UI 通过，也没有更新 WSL/NAS。

待补视觉矩阵：

| Frame/流程 | 375 | 390 | 430 | 1440 |
|---|---|---|---|---|
| 自动选择/建议/冲突预览 | pending | pending | pending | pending |
| 行编辑、8 标签、学习失败 | pending | pending | pending | pending |
| 批量接受/相同值/相同商户 | pending | pending | pending | pending |
| 规则分组与 stale 引用 | pending | pending | pending | pending |
| 新账本 profile / 既有账本 preview | pending | pending | pending | pending |
| 兜底分类替代 | pending | pending | pending | pending |
| Fresh Light / Dark Glass | pending | pending | pending | pending |

浏览器验收还必须覆盖长商户名、8 标签、键盘焦点、`aria-live` 和无横向滚动。线上 Figma 状态继续保持 `not_verified`；本记录不声称已同步线上文件。

## 4. 下一入口

1. 用户确认后按 `deploy/v13/docker-compose.task53-staging.yml` 启动隔离 38092/schema 22。
2. 运行 Task53.5 的 migration/fixture/flag 验证，并生成上述视口截图。
3. 视觉矩阵通过后关闭 Task53U，再完成 Task53.5 发布级收口。
4. Task53 关闭后只返回 Task51P.1 证据评审；当前真实证据仍为 0，不进入 Task51P.2 或代码。
