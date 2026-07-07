# LedgerTwo v1.1 UI 提交验收记录

日期：2026-07-07

## 1. 验收环境

- 本机 WSL2 Docker：`http://localhost:38088`
- 健康检查：`GET /api/healthz`
- 版本口径：`version=1.1.0-rc`、`db=ok`、`schema_version=12`
- 账号：本地 QA 账号 `userA`
- 视口：移动端 375px

## 2. 覆盖路径

1. 打开记账抽屉。
2. 普通支出输入金额 31.23 元和标题，通过“保存并继续”提交。
3. 验证普通支出成功写入交易列表。
4. 验证保存成功后抽屉保持打开，金额和标题清空。
5. 切换为共同支出，输入金额 66.00 元和标题，通过“确认记账”提交。
6. 验证共同支出成功写入交易列表。
7. 验证共同支出默认 equal split，两个参与人各承担 3300 分。
8. 验证结算余额随共同支出更新，`final_net_cents` 为正负 7700 分。
9. 验证提交后 375px 无横向溢出。

## 3. 验收证据

- `submit-metrics.json`：交易查询、共同支出 split、结算余额和页面溢出指标。
- `continue-diagnostic.json`：保存并继续后的抽屉、表单、金额和标题 DOM 诊断。
- `screenshots/01-drawer-open.png`：记账抽屉打开。
- `screenshots/02-ordinary-before-continue.png`：普通支出提交前。
- `screenshots/03-after-save-continue.png`：保存并继续后。
- `screenshots/04-shared-before-submit.png`：共同支出提交前。
- `screenshots/05-transactions-after-submit.png`：提交后流水页。
- `screenshots/diagnostic-after-continue.png`：保存并继续后诊断截图。
- `card-category-check.json`：重建容器后流水卡片分类展示检查。
- `screenshots/06-transactions-card-category-after-fix.png`：移动端流水卡片分类展示修复后截图。

## 4. 结论

普通支出、保存并继续、共同支出提交、equal split 和结算余额联动已在本机 WSL2 完成真实 UI 提交闭环。

移动端流水卡片重检结果：前 8 张交易卡片 `uuidInCards=false`、`uuidInBody=false`、`scrollWidth=375`、`innerWidth=375`，不再暴露分类 UUID，375px 无横向溢出。

说明：`submit-metrics.json` 中 `continueAmountCleared=false` 来自脚本选择器未抓到金额输入框，原始 `afterContinue.amount` 为 `null`；该项以 `continue-diagnostic.json` 的 `drawerCount=1`、`formCount=1`、`amount=""`、`title=""` 和截图 `diagnostic-after-continue.png` 作为最终验收依据。

## 5. 本轮修复

1. 修复 `TransactionFormDrawer` 成功回调读取旧 `submitAction` 导致“保存并继续”后误关闭抽屉的问题。
2. 修复移动端流水卡片展示 `category_id` 的问题，改为展示分类名、`未分类` 或 `已设分类`。
3. 补充流水卡片标题和分类字段的移动端省略保护，降低长文本撑破布局的风险。
