# UI-FL-05E 本机固定镜像收口验收

日期：2026-07-15  
状态：通过  
范围：本机 WSL2 staging、真实 QA 账号、隔离数据库副本；未访问 NAS，未提交真实导入批次

## 1. 结论

UI-FL-05E 已完成源码、自动化、固定镜像和真实账号运行态收口，可以将 UI-FL-05 标记为完成，并进入 UI-FL-06。主题切换入口在登录页和应用壳层真实存在；本机运行实例已更新到包含 UI-FL-05E、支付宝零金额行提示和主题切换的固定镜像。

本轮运行验收发现并修复两个前端缺陷：

1. 编辑态仍显示禁用的“保存并继续”，与“编辑态不提供该动作”的冻结契约不一致。
2. 390px 编辑抽屉被 AppShell 全局“记一笔”按钮覆盖，付款人和分摊区域受到遮挡。

对应提交：

```text
67253bc  fix(ui): 收口原账单编辑底部操作
b83f766  fix(ui): 避免移动记账入口遮挡编辑抽屉
```

## 2. 数据保护

更新前已执行 SQLite 在线一致性备份：

```text
backups/predeploy/ui-fl-05e-6c32f4d-20260715-112952
schema_version=19
row_counts users|ledgers|transactions|settlements=2|2|40|2
quick_check=ok
```

写路径验收在 `backups/acceptance/ui-fl-05e-b83f766/` 的数据库副本和临时 `38090` 容器中执行。临时容器已删除，副本保留在 Git 忽略目录供本机复核。

原本机 staging 数据库验收后保持：

```text
PRAGMA quick_check=ok
transactions=40
ui-fl-05e-runtime 标签数量=0
```

因此，批量标签、软删除和编辑写入没有进入原本机 staging 数据库。

## 3. 部署结果

本机地址：`http://localhost:38088`  
运行镜像：`ledger-two:1.2.0-rc-b83f766`

Health 回读：

```text
version=1.2.0-rc
schema_version=19
deployment_channel=staging
import_xlsx_enabled=true
db=ok
```

标准多阶段 Docker 构建首次因 Docker Hub 连接被重置而失败。本机验收镜像随后使用既有 `ledger-two:1.2.0-rc` 最终镜像作为运行基底，注入当前 Linux 后端二进制和最新前端 production build。该镜像可用于本机固定版本验收，但不替代 NAS 发布窗口中的标准 Dockerfile 全量构建和镜像校验。

## 4. 浏览器验收

浏览器使用本机 Chrome DevTools Protocol 和真实 `userA` QA 账号。主实例只执行登录、读取、打开编辑器和截图，没有提交修改。

主题切换：

```text
before: theme=dark-glass, aria-label=切换到白绿色浅色主题
after:  theme=fresh-light, localStorage=ledger-two-ui-theme:fresh-light
390px:  innerWidth=390, scrollWidth=390, toggle=44x44
```

流水与编辑器：

```text
1440px / 390px 均无横向溢出
共同支出可从流水进入原账单编辑器
账单类型锁定说明可见
保存修改可见
保存并继续不可见
模板动作不可见
归档元数据说明可见
移动端全局记账按钮不再覆盖编辑抽屉
```

隔离副本写路径：

```text
ordinary metadata patch=passed
archived metadata retained=passed
shared split retained=passed
attachment references retained=passed
batch tag=passed
soft delete=passed
CSV export=passed
```

## 5. 证据

- `theme-toggle-fresh-light-390.png`：登录页切换到 Fresh Light 后的主题入口。
- `shared-editor-390.png`：390px 共同支出编辑器和双动作 Footer。
- `shared-editor-1440.png`：1440px 流水工作台与共同支出编辑器。

## 6. 后续边界

1. UI-FL-06 可开始实现；UI-FL-07 继续复用已冻结的共享组件。
2. Task49/Task49X 的开发实现与本机验收已完成，但 NAS schema 19 staging、production 发布和逐批真实导入确认仍由发布计划管理。
3. 当前本机镜像不能直接作为 NAS 候选；网络恢复后必须重新执行标准 Dockerfile 全量构建。
4. Fresh Light 继续保留显式切换，默认主题只能在 UI-FL-10 全局验收后决定是否切换。
