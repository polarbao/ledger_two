# UI-FL-06 结算解释与确认收口验收

日期：2026-07-15  
状态：通过  
范围：本机 WSL2 staging、真实 QA 账号只读路径、隔离数据库副本写路径；未访问 NAS

## 1. 结论

UI-FL-06 已完成开发与收口。结算页现在提供真实的“全部未结 / 仅本月”范围切换，按服务端整数分结果解释 `paid/share/raw_net/settlement/final_net`，并使用共享确认框生成结算记录。复制结算文案不改变状态，浏览器拒绝 Clipboard API 时会展示可手动选择的完整文本。

实现提交：

```text
daccda3  feat(settlement): 完成 UI-FL-06 结算解释与确认流程
```

## 2. 契约补齐

既有后端文档已经声明 `GET /api/settlements/balance?month=YYYY-MM`，但实现此前忽略月份。为避免范围控件成为假切换，本任务补齐可选 `month` 查询：

- 不传 `month`：保持全部账期的向后兼容结果。
- 传合法月份：共同支出、分摊和结算记录使用同一月份范围。
- 非法月份：返回 400，不进入聚合查询。
- 无 migration、schema 或依赖变化。

## 3. 自动化门禁

```text
CC=C:\Qt\Qt6.9.x\Tools\mingw1310_64\bin\gcc.exe go test ./...
结果：全部通过

npm run lint
结果：通过

npm run test
结果：20 个测试文件、78 个测试通过

npm run build
结果：通过；保留既有主包大于 500 kB 告警
```

新增测试覆盖月份聚合与非法月份、范围缓存键、结算文案、Clipboard API/旧复制路径失败、共享组件与响应式样式契约。

## 4. 本机部署

部署前在线备份：

```text
backups/predeploy/ui-fl-06-20260715-142441/ledger.db
quick_check=ok
users|ledgers|transactions|settlements=2|2|40|2
```

运行结果：

```text
URL=http://localhost:38088
image=ledger-two:1.2.0-rc-ui-fl-06
deployment_channel=staging
schema_version=19
import_xlsx_enabled=true
db=ok
```

标准 Dockerfile 全量构建在 E 盘大上下文传输阶段无输出，已主动终止。本机验收镜像以既有固定镜像为运行基底，注入当前 Linux 后端二进制和 production 前端构建；它仅用于本机 staging，不替代 NAS 发布候选的标准全量构建。

## 5. 浏览器验收

真实 `userA` QA 账号只执行登录、读取、范围切换、复制和打开/取消确认框，没有在主实例生成结算记录。

```text
1440px: innerWidth=1440, scrollWidth=1440
375px: innerWidth=375, scrollWidth=375
theme=fresh-light
全部未结/仅本月切换=passed
paid/share/raw_net/settlement/final_net=visible
影响账单与历史结算=visible
Clipboard 拒绝后的手动文本=passed
复制前后 settlement 数量=2 -> 2
确认框双方/金额/影响说明/明确按钮=passed
```

## 6. 隔离写路径

写入仅发生在 `backups/acceptance/ui-fl-06-runtime-20260715/` 的数据库副本和临时 `38090` 容器中。临时容器已停止，副本保留在 Git 忽略目录。

```text
before: shared_expenses=7, settlements=2, balance=13200 cents
after:  shared_expenses=7, settlements=3, balance=0 cents
acceptance settlement=1
shared bill id/amount/status/updated_at unchanged=true
clone quick_check=ok
```

主实例复核：

```text
quick_check=ok
transactions=40
settlements=2
ui-fl-06-isolated-acceptance=0
```

因此，真实登记路径只新增独立 settlement 和对应流水，没有修改历史共同支出，也没有污染本机 staging 原数据。

## 7. 视觉证据

- `settlement-fresh-light-1440.png`：桌面结算结论、解释表和 Fresh Light 层级。
- `settlement-fresh-light-375.png`：375px 转账行动与复制失败手动文本，无横向溢出。
- `settlement-confirm-375.png`：移动确认框的双方、金额、备注、审计说明和明确确认动作。

## 8. 后续边界

1. UI-FL-06 门禁关闭，UI 主线下一任务为 UI-FL-07 设置与元数据。
2. Task50 准备已开启但仍缺 PRD、技术契约、UI、OpenAPI、migration 和验收矩阵冻结；当前只执行 P.1/P.2 文档准备，不进入编码。
3. Task49X 的 NAS schema 19 staging、production 备份和逐批真实导入确认继续由 v1.2 发布计划管理。
