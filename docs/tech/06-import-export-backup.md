# 技术：导入、导出、备份恢复

## 1. 模块目标

导入、导出、备份恢复模块用于保证账务数据可迁移、可备份、可恢复，并为后续微信/支付宝 CSV 导入与规则自动化打基础。

## 2. 导出设计

### 2.1 CSV 导出

用途：人工查看、Excel/Numbers 分析、轻量迁移。

建议接口：

```text
GET /api/export/transactions.csv?month=2026-06
```

CSV 字段建议：

```text
发生时间,类型,标题,金额分,金额元,分类,标签,付款人,归属人,可见性,备注
```

### 2.2 JSON 全量导出

用途：迁移、调试、二次开发。

建议接口：

```text
GET /api/export/full.json
```

JSON 应包含：

- users，脱敏。
- categories。
- tags。
- accounts。
- transactions。
- transaction_splits。
- settlements。
- audit_logs。
- app_settings。

不得导出明文密码或 session secret。

### 2.3 SQLite 备份下载

用途：完整恢复。

建议接口：

```text
POST /api/admin/backup
GET /api/admin/backups
GET /api/admin/backups/{filename}
```

## 3. SQLite 安全备份

不要在写入过程中直接复制数据库文件。推荐两种方式：

### 3.1 VACUUM INTO

```sql
VACUUM INTO '/app/backups/daily/ledger-two-2026-06-11.db';
```

### 3.2 SQLite Backup API

适合后续在 Go 中实现更稳妥的在线备份。

## 4. 备份目录结构

```text
backups/
  daily/
  weekly/
  monthly/
  manual/
```

## 5. 保留策略

- 每日备份保留 30 天。
- 每周备份保留 12 周。
- 每月备份保留 12 个月。
- 手动备份默认不自动删除，除非用户确认。

## 6. 导入设计

后续 v0.4 支持 CSV 导入，流程：

```text
上传文件 -> 解析预览 -> 字段映射 -> 去重检查 -> 用户确认 -> 批量写入
```

导入相关表：

```text
import_batches
import_items
import_rules
```

## 7. 去重策略

建议生成 import_hash：

```text
source + occurred_at + amount + merchant + account + raw_description
```

同一 import_hash 不允许重复写入。

## 8. 规则自动化

规则示例：

```text
商户包含 星巴克 -> 分类：餐饮，标签：咖啡
商户包含 滴滴 -> 分类：交通，标签：打车
```

规则命中后仍需在导入预览页让用户确认。

## 9. 恢复策略

恢复属于高风险操作。建议流程：

1. 上传或选择备份文件。
2. 系统校验文件格式。
3. 显示备份时间、账单数量、用户数量。
4. 要求用户输入确认文本。
5. 当前数据库先自动备份。
6. 停止写入。
7. 执行恢复。
8. 重启或重新加载数据库连接。

Demo 到 v0.2 可以先只提供备份和下载，恢复流程先写文档，不急于做 UI 操作。

## 10. 验收标准

- 手动备份可生成 SQLite 备份文件。
- 备份文件可下载。
- CSV 导出金额、类型、分类、标签正确。
- JSON 导出不包含明文密码或密钥。
- 备份失败时有明确错误码。
- NAS 部署文档明确 data、backups、uploads 都需要备份。
