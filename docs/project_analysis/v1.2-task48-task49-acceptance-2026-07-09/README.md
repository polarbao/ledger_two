# v1.2 Task48 / Task49 收口验收记录

日期：2026-07-09

## 1. 环境

- 本机 WSL2 Docker：`http://localhost:38088`
- Schema：`18`
- 浏览器：Chrome DevTools Protocol
- 视口：`375px`、`390px`
- 账号：本地 QA Owner 账号

## 2. Task48 验收

### 自动化

后端 `internal/importer` 已覆盖：

1. I02：重复文件第二次预览标记 duplicate，提交后新增 0 条、跳过 4 条。
2. I03：未确认 suspicious 行阻止提交；确认后批次恢复 ready 并可提交。
3. I04：invalid 行导致批次进入 failed，transactions 和 import refs 均无半批写入。
4. failed 批次将问题行跳过后恢复 ready，可重新提交。
5. editor 不可读取导入批次。

前端 Vitest 已覆盖：

1. suspicious 和 invalid 未处理时阻断提交。
2. suspicious 确认、invalid 跳过后的提交统计。
3. API 错误详情中的 `row_number` 转为可行动提示。

### 浏览器

- `task48-commit-modal-375.png`：375px 提交确认弹窗，疑似已确认后显示导入 4、跳过 0、阻断 0。
- `task48-commit-result-375.png`：实际提交成功，显示导入 4、跳过 0、失败 0。
- `task48-failure-feedback-390.png`：模拟 commit API 事务失败响应，页面显示“第 2 行”及“未写入正式账单”。
- 三个场景均无横向溢出和 React Router 错误页。

提交失败截图使用 CDP 拦截 commit API 返回 409，仅用于验证前端失败反馈；后端真实回滚由 importer 自动化测试覆盖。

## 3. Task48 结论

Task48 与 Task48U 验收完成，可以标记为已完成。提交确认弹窗曾受页面入场动画 transform 影响而定位到长页面中部，已改用 Portal 挂载到 `document.body`，恢复真正的视口居中。

## 4. Task49 验收

### 自动化

1. active 规则生成 `suggested_*`、规则 ID 和解释文案。
2. archived 规则不参与预览命中。
3. active 规则引用已归档元数据时整条规则不应用，避免静默推荐不可用值。
4. 用户手工 `selected_*` 字段保持优先，规则建议仅保留为解释信息。
5. editor/viewer 对规则读取、创建、更新、归档、恢复和兼容删除入口均返回 403。
6. editor/viewer 不可读取导入批次。

### 浏览器

- `task49-rule-manager-390.png`：390px 规则表单、状态筛选、多标签和规则卡片。
- `task49-rule-hit-390.png`：预览行展示命中原因及推荐分类、账户、标签名称。
- `task49-archived-no-hit-390.png`：规则归档后重新预览，同一行不再显示规则建议。
- 三个场景均无横向溢出和 React Router 错误页。

## 5. Task49 结论

Task49 与 Task49U 的功能和验收项已完成，可以进入 v1.2 冻结前 OpenAPI、API inventory、迁移和综合质量门禁检查。
