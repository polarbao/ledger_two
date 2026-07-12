# Task49X 真实账单字段与移动端 UI 验收

日期：2026-07-12
范围：本机 WSL staging；不访问 NAS，不调用 import commit

## 1. 验收结论

真实微信 XLSX 的 276 行已完成源文件与 schema 19 preview 数据逐行核对：金额、交易时间和交易订单号均为 0 差异，方向规则也为 0 差异。导入工作台在 375px/390px 下未发现页面横向滚动或按钮文字截断，XLSX 解析摘要和 generic XLSX 拒绝状态可读。

本轮仍未提交任何真实账单。`transactions=40`，数据库 `PRAGMA quick_check=ok`。

## 2. 账务字段核对

| 项目 | 源 XLSX | preview 数据 | 结果 |
|---|---:|---:|---|
| 数据行 | 276 | 276 | 一致 |
| 金额绝对值合计 | 4,301,943 分 | 4,301,943 分 | 一致 |
| 金额逐行差异 | - | 0 | 通过 |
| 交易时间逐行差异 | - | 0 | 通过 |
| 交易订单号逐行差异 | - | 0 | 通过 |
| 方向逐行差异 | - | 0 | 通过 |

方向分层：

| 标准方向 | 行数 | 金额合计/分 | 目标行为 |
|---|---:|---:|---|
| expense | 175 | 1,181,804 | 候选支出 |
| income | 22 | 152,586 | 候选收入 |
| refund | 23 | 260,363 | 候选收入，保留退款语义 |
| transfer | 56 | 2,707,190 | 默认 skipped，不创建正式收支 |

分层抽样覆盖首两行、唯一 suspicious 行和末行，即物理行 19、20、214、294。四行的时间、金额与长订单号均与原文件一致；第 214 行保持 `suspicious/pending`，未被自动确认。

## 3. 移动端证据

截图：

1. `screenshots/01-source-file-format-390.png`：390px 来源与格式入口。
2. `screenshots/02-xlsx-preview-summary-390.png`：390px XLSX 工作表、表头、识别行和状态统计。
3. `screenshots/03-xlsx-preview-summary-375.png`：375px 同一预览摘要。
4. `screenshots/04-generic-xlsx-error-390.png`：通用模板选择 XLSX 后的前端拒绝状态。

`metrics.json` 记录：

- 390px：documentScrollWidth=390，horizontalOverflow=false。
- 375px：documentScrollWidth=375，horizontalOverflow=false。
- 两个宽度均无 clippedButtons。
- generic XLSX 错误为“通用模板当前仅支持 CSV 文件”。

视觉检查未发现文字越界。首次截图发现 `scrollIntoView` 会把预览标题滚到 sticky 顶栏下方；本轮为预览面板增加 86px `scroll-margin-top`，避免程序化聚焦或锚点跳转遮住标题。固定底部导航保持既有层级，账单行仍可继续向下滚动。

## 4. 本轮修正

1. 空预览状态仍写着“上传 CSV 后”，与已经支持 CSV/XLSX 的事实不一致；改为“上传账单文件后”，并增加服务端静态渲染测试。
2. 预览面板缺少 sticky 顶栏滚动避让；增加 86px `scroll-margin-top`，并使用浏览器位置断言复核。

## 5. 自动化副作用与数据边界

Chrome CDP 文件控件验收触发了 5 个额外本机 ready preview 批次：

```text
4874f3a3-d65c-4989-9f21-17a585e4fd06
7cdc4785-a885-4491-837f-548ee9a13d4b
f624669b-352f-428e-bb4e-4b89cccdf149
01c7c85e-60f0-42c0-abfa-df24b20d8ef6
3898b025-be20-44e9-9bcd-fbe62f0f0374
```

五个批次均为 276 行、`imported_rows=0`，未改变 transactions。为避免未经确认删除本机验收记录，本轮保留这些 preview 批次；它们不是 production 数据。

## 6. Figma 与剩余门禁

账户主文件仍为 `LedgerTwo v1.2 UI System - polar`。本轮连接到的 Figma 账号 handle 为 `zy j`，团队席位为 View；读取主文件时返回“没有 edit access”，因此未对 Figma 文件执行写入，也未伪造 node id 或同步结果。

剩余门禁：

1. 将当前 Figma 账号升级为可编辑席位，或由文件 Owner 对该账号授予编辑权限。
2. 获得真实支付宝 XLSX 样本并完成 Task49X.4 正式冻结。
3. 在明确的 NAS staging 维护窗口执行 schema 19 备份、迁移和重启验收。
4. 任何 production commit 前重新预览，并由用户逐批确认状态数量。
