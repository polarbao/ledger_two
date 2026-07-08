# LedgerTwo v1.1 计时与归档分类展示验收

日期：2026-07-08  
环境：本机 WSL2 Docker，`http://localhost:38088`  
浏览器：Chrome headless via CDP  
视口：375 x 812

## 1. 覆盖范围

本轮用于补齐 v1.1 冻结前剩余的本地 UI 证据：

1. 普通支出从打开记账抽屉到提交成功不超过 10 秒。
2. 共同支出从打开记账抽屉到提交成功不超过 20 秒。
3. 共同支出默认 equal split，双方各承担 3300 分。
4. 创建账单引用分类后归档该分类，历史流水卡片仍展示分类名和已归档提示。
5. 375px 移动端无横向溢出。

## 2. 本轮发现与修复

首次验收发现历史流水引用归档分类后，移动端卡片只显示 `已设分类`。

修复方式：

1. 后端 `TransactionResponse` 增加 `category_name` 与 `category_is_archived`。
2. 交易列表与详情查询通过 `LEFT JOIN categories` 返回分类展示字段。
3. 前端流水页优先使用交易 DTO 自带的分类展示字段，分类列表映射仅作为兜底。
4. 后端 `TestTransactionFlow` 增加归档分类展示字段断言。

## 3. 验收结果

证据文件：

```text
metrics.json
screenshots/01-after-timing-submit.png
screenshots/02-archived-category-history.png
```

关键结论：

```text
ordinary_10s_pass=true
shared_20s_pass=true
archived_category_display_pass=true
mobile_overflow_pass=true
```

计时结果：

```text
普通支出：124ms
共同支出：233ms
```

说明：以上是自动化最短路径技术耗时，用于证明前端交互、请求和成功回读链路没有性能阻断；不等同于人工手动录入耗时。

## 4. 已运行验证

```bash
wsl bash -lc "cd /mnt/e/__Code/__Prj/ledge_two/ledger_two/backend && go test ./internal/... -count=1"
corepack pnpm --dir frontend build
git diff --check
wsl bash -lc "cd /mnt/e/__Code/__Prj/ledge_two/ledger_two && sudo docker compose up -d --build"
Chrome CDP v1.1 timing and archived category acceptance
```

`pnpm build` 仍有主 chunk 超过 500KB 的既有警告，已归入后续性能专项，不阻塞 v1.1 收口。

## 5. 剩余事项

本地 v1.1 主要冻结证据已收口。仍建议在正式冻结前补 NAS 地址下 UI 复核：

1. 登录页可登录。
2. Dashboard 正常加载。
3. 流水列表和记账抽屉正常。
4. 结算中心可展示和复制文案。
5. 设置页可进入分类、标签、账户管理。
6. 附件上传和受控读取正常。
7. 手动备份创建和下载正常。
