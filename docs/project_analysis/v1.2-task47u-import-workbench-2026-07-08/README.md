# LedgerTwo v1.2 Task47U 导入预览工作台验收记录

日期：2026-07-08

## 验收环境

- 本机 WSL2 Docker：`http://localhost:38088`
- 健康检查：`GET /api/healthz`
- Schema：`schema_version=14`
- 账号：本地 QA 账号 `userA`
- 视口：移动端 375px

## 覆盖路径

1. 登录本机环境。
2. 写入当前账本上下文。
3. 打开 `/import`。
4. 选择默认微信来源。
5. 上传 `docs/fixtures/imports/wechat-basic.csv`。
6. 验证预览批次统计、移动端行卡片、invalid 错误码和禁用提交提示。

## 验收证据

- `metrics.json`：CDP 指标与关键文案状态。
- `import-workbench-375.png`：375px 移动端导入预览截图。

## 结论

- 375px 下 `scrollWidth=375`、`innerWidth=375`，无横向溢出。
- 上传微信 fixture 后展示 5 张导入行卡片。
- 预览统计展示总行数、新增、疑似、错误、跳过。
- 页面明确展示预览未写入 `transactions`。
- invalid 行展示 `IMPORT_ROW_AMOUNT_INVALID` 和可行动原因。
- Task47 阶段提交入口保持 disabled，提示“预览阶段暂不可提交”。
