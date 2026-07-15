# UI-FL-09 分析工作台验收记录

日期：2026-07-15

实现提交：`9d8066f`

验收环境：本机 WSL2 staging，`http://localhost:38088`

## 1. 验收结论

UI-FL-09 已通过代码、自动化和运行时门禁，波次 D 可以关闭并进入 UI-FL-10：

1. 分析页按趋势、分类、成员、标签四个问题组织，不堆叠无业务结论的装饰图表。
2. 趋势复用既有 monthly-summary 接口组合近六个月数据，没有新增 API 或在前端重算服务端权威金额。
3. 分类、标签、成员支付和月份均可钻取到流水页现有 URL 筛选，账期与账本上下文保持不变。
4. 成员页明确展示实际支付、消费承担、垫付净额、已登记结算和最终未结，不使用“记账人排行”。
5. 页面明确 settlement 只调整最终未结，不进入消费统计；标签页明确多标签全额命中后总和不等于账期总支出。
6. 页面说明统计基于当前用户可见账单，继续遵守 private/partner_readable/shared 可见性，不展示 UUID。
7. 375px 与 1440px 均满足 `scrollWidth = innerWidth`，无横向滚动；移动端没有宽表，摘要、图表、排行和成员卡片可继续纵向浏览。

## 2. 证据清单

| 视口 | 趋势 | 分类 | 成员 | 标签 |
|---|---|---|---|---|
| 1440px | `trend-1440.png` | `category-1440.png` | `member-1440.png` | `tag-1440.png` |
| 375px | `trend-375.png` | `category-375.png` | `member-375.png` | `tag-375.png` |

运行时断言：

- 趋势：6 个账期柱组、5 个摘要指标，当前账期支出/收入/个人/共同/未结口径齐全。
- 分类：当前 QA 数据 11 行且每行有可读进度；有 ID 的分类可钻取。
- 成员：2 个成员卡片，支付、承担、垫付和未结概念齐全。
- 标签：当前 QA 数据 9 行，多标签口径提示存在。
- 分类钻取：`/transactions?month=2026-07&page=1&category_id=...`。
- 成员钻取：`/transactions?month=2026-07&page=1&payer_user_id=...`。
- 标签钻取：`/transactions?month=2026-07&page=1&tag=...`。

验收记录只保留参数结构，不在文档中固化本机用户或分类 UUID。

## 3. 数据安全

本次运行时验收只执行 GET 报表、登录和页面导航，没有触发任何 mutation。验收后：

```text
PRAGMA quick_check = ok
users = 2
ledgers = 2
transactions = 40
settlements = 2
import_batches = 34
```

数据库核心计数与 UI-FL-08 结束时一致。没有上传账单、生成结算、修改账单、执行 migration 或访问 NAS。

## 4. 部署回读

- 镜像标签：`ledger-two:1.2.0-rc-ui-fl-09`
- 镜像 ID：`sha256:e121b617c76a35a2939ae7ca09ba63829ac9cc6b2aa45c1b24e12a1dd8c16144`
- 前端 JS：`/assets/index-CHjxi6IV.js`
- 前端 CSS：`/assets/index-AYwveltP.css`
- Health：`staging / schema 19 / XLSX enabled / db ok / version 1.2.0-rc`

本次只更新本机 WSL2 staging 静态资源和本地候选镜像，NAS 未更新。

## 5. 自动化门禁

```text
frontend npm run lint       PASS
frontend npm test -- --run  PASS (27 files / 99 tests)
frontend npm run build      PASS
backend go test ./...       PASS (本会话早前执行，UI-FL-09 未改后端)
```

Vite 主包约 677.80 kB，仍有大于 500 kB 的既有分包告警。该告警进入 UI-FL-10 性能审阅，但不以临时隐藏阈值代替实际判断。

## 6. 变更边界

本任务没有修改后端报表算法、金额、可见性、结算、API DTO、OpenAPI、migration 或第三方依赖。Task50 仍只进行 P.4/P.5 准备，不因分析页完成而提前编码。
