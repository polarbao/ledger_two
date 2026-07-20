# 08 MVP 路线图、交付物与验收

## 1. 初版交付目标

交付一个可在群晖 NAS 上部署的双人共享账本 Web 应用。

必须支持：

1. 两个用户登录。
2. 记录普通支出和收入。
3. 记录共同支出。
4. 平均分摊和仅付款人承担。
5. 自动计算谁欠谁。
6. 生成结算记录。
7. 首页总览、流水页、结算页、统计页、设置页。
8. Docker Compose 部署。
9. SQLite 自动备份。
10. CSV/JSON 导出。

## 2. 里程碑

### M1：基础工程

- 建立仓库结构
- 后端 Go 服务启动
- 前端 Vite 项目启动
- Dockerfile 和 compose 初版
- SQLite migrations

### M2：认证与初始化

- 初始化向导
- 创建两个用户
- 登录/退出
- 修改密码
- 当前用户 API

### M3：基础账单

- 分类管理
- 标签管理
- 账户管理
- 普通支出 CRUD
- 收入 CRUD
- 流水列表
- 账单详情

### M4：共同支出与分摊

- 创建共同支出
- 平均分摊
- 仅付款人承担
- 分摊明细展示
- 修改共同支出后重算

### M5：结算中心

- 计算双方净额
- 结算中心页面
- 生成结算记录
- 历史结算记录

### M6：统计与首页

- Dashboard API
- 核心数据卡片
- 最近流水
- 分类统计
- 成员统计
- 标签统计
- 趋势统计

### M7：部署与备份

- 群晖部署文档
- 自动备份
- 手动备份
- CSV 导出
- JSON 导出
- 健康检查

## 3. 推荐开发顺序

```text
1. DB schema
2. migrations
3. auth
4. init wizard
5. category/tag/account
6. transaction CRUD
7. frontend dashboard shell
8. transaction list/detail/form
9. split service
10. shared expense form
11. settlement service/page
12. reports
13. Docker/NAS deployment
14. backup/export
```

## 4. 验收测试

### 4.1 登录

- polar 登录成功
- lynn 登录成功
- 错误密码失败
- 退出后不能访问 API

### 4.2 个人账单

- 新增支出成功
- 新增收入成功
- 编辑金额成功
- 删除后列表不显示
- private 对方不可见
- partner_readable 对方可见不可编辑

### 4.3 共同账单

- polar 支付 200，两人平摊，lynn 欠 polar 100
- lynn 支付 80，两人平摊，polar 欠 lynn 40
- 净额为 lynn 欠 polar 60
- lynn 结算 60 后，双方结清

### 4.4 UI

- 桌面端左侧导航可用
- 移动端底部导航可用
- 点击流水打开详情抽屉
- 点击 + 打开记账抽屉/弹窗
- 删除账单有确认提示
- 结算有确认提示
- 离线时显示服务异常

### 4.5 部署

- Docker Compose 启动成功
- 重启后数据不丢失
- `/healthz` 正常
- 自动备份生成文件
- CSV/JSON 导出可用

## 5. 风险清单

| 风险 | 处理 |
|---|---|
| SQLite 文件损坏 | 每日备份 + WAL + 第二备份位置 |
| NAS 磁盘故障 | 不把 NAS 作为唯一副本 |
| 金额计算误差 | 全部使用整数分 |
| 分摊不平 | 后端校验 splits 总额 |
| 权限混乱 | 明确 private/partner_readable/shared |
| UI 表单太复杂 | MVP 先做平均分摊和仅付款人承担 |

## 6. 初版文档输出清单

- PRD
- UI 交互设计稿
- 技术设计文档
- 技术实现文档
- 前端设计文档
- NAS 部署文档
- 数据库与 API 文档
- MVP 路线图
