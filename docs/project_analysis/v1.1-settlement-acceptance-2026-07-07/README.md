# LedgerTwo v1.1 结算页 UI/UX 验收记录

日期：2026-07-07

## 1. 验收环境

- 本机 WSL2 Docker：`http://localhost:38088`
- 健康检查：`GET /api/healthz`
- 版本口径：`version=1.1.0-rc`、`db=ok`、`schema_version=12`
- 账号：本地 QA 账号 `userA`
- 视口：移动端 375px

## 2. 覆盖路径

1. 若当前无待结算金额，先创建一笔 88.88 元共同支出作为验收前置数据。
2. 打开结算中心。
3. 验证 paid/share/raw_net/settlement/final_net 展示。
4. 点击“复制文案”，强制模拟剪贴板失败。
5. 验证页面展示只读完整结算文案，用户可手动长按复制。
6. 点击“登记结算”，打开确认弹窗。
7. 填写结算备注并确认登记。
8. 验证结算后余额归零、历史记录新增、备注展示。
9. 验证 375px 全流程无横向溢出。

## 3. 验收证据

- `settlement-metrics.json`：结算前后余额、复制兜底、历史记录、备注和移动端宽度指标。
- `screenshots/01-settlement-before.png`：结算前待结算状态。
- `screenshots/02-after-copy-fallback.png`：剪贴板失败后的手动复制兜底。
- `screenshots/03-confirm-modal.png`：登记结算确认弹窗。
- `screenshots/04-after-settlement.png`：登记后账目结清和历史记录。

## 4. 结论

结算页 v1.1 核心路径已完成本机 WSL2 真实 UI 验收：

- 复制文案在剪贴板失败时有完整手动复制兜底。
- 登记结算成功生成 settlement 记录。
- 结算后 `final_net_cents` 归零，`suggested_transfers` 为空。
- 当期结算历史展示最新记录和备注。
- 375px 移动端全流程 `scrollWidth=375`、`innerWidth=375`，无横向溢出。

## 5. 本轮修复

1. 复制文案失败时不再只提示错误，而是展示完整只读结算文案。
2. 手动复制区域增加移动端可读样式、换行和可聚焦选中行为。
