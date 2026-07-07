# LedgerTwo v1.1 复制与模板 UI 验收记录

日期：2026-07-07

## 1. 验收环境

- 本机 WSL2 Docker：`http://localhost:38088`
- 健康检查：`GET /api/healthz`
- 版本口径：`version=1.1.0-rc`、`db=ok`、`schema_version=12`
- 账号：本地 QA 账号 `userA`
- 视口：移动端 375px

## 2. 覆盖路径

1. 通过 API 创建一笔普通支出作为源账单。
2. 在流水页搜索源账单并打开详情。
3. 点击“复制一笔”，打开复制账单抽屉。
4. 验证复制来源提示、金额和标题回填。
5. 修改标题并通过“保存为新账单”提交。
6. 回查复制账单成功写入，源账单仍只有 1 条且未被修改。
7. 再次打开源账单详情，点击“存为模板”。
8. 保存模板并回查模板列表。
9. 从 Dashboard 打开“记一笔”，选择刚保存的模板填入表单。
10. 修改标题并通过“确认记账”提交。
11. 回查模板生成账单成功写入，模板名称本身没有作为正式流水进入交易列表。
12. 验证 375px 全流程无横向溢出。

## 3. 验收证据

- `copy-template-metrics.json`：源账单、复制账单、模板、模板生成账单和移动端宽度指标。
- `screenshots/01-source-detail.png`：源账单详情与复制/模板入口。
- `screenshots/02-copy-drawer.png`：复制一笔抽屉。
- `screenshots/03-template-save-modal.png`：另存为模板弹窗。
- `screenshots/04-template-applied.png`：模板填入后的记账表单。
- `screenshots/05-templated-transaction-list.png`：模板生成账单后的流水列表。

## 4. 结论

复制与模板模块 v1.1 核心路径已完成本机 WSL2 真实 UI 验收：

- 复制一笔不会修改原账单。
- 复制账单成功写入正式流水。
- 源账单可保存为模板。
- 模板可在记账抽屉一键填入。
- 模板生成账单成功写入正式流水。
- 模板本身不作为正式流水参与交易列表。
- 375px 移动端全流程 `scrollWidth=375`、`innerWidth=375`，无横向溢出。

## 5. 本轮修复

1. 修复移动端流水卡片 header 收缩规则，避免长标题挤压金额区域。
