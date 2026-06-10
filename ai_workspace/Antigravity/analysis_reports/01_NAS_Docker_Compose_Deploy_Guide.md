# 群晖 NAS Docker Compose 部署与验证指南

本指南详细说明如何在群晖 NAS (DSM 7.x) 上通过 Container Manager（即 Docker 升级版）部署并验证 LedgerTwo 项目。

---

## 1. 部署架构设计

通过 Docker 挂载数据卷，确保容器升级/重启时账目数据绝对不丢失，其容器与 NAS 目录映射拓扑结构如下：

```mermaid
graph TD
    subgraph Synology NAS Host [/volume1/docker/ledger-two/]
        H_db[(data/ledger.db)]
        H_bk[(backups/)]
        H_up[(uploads/)]
        H_lg[(logs/)]
    end

    subgraph Docker Container [/app/]
        C_db[(data/ledger.db)]
        C_bk[(backups/)]
        C_up[(uploads/)]
        C_lg[(logs/)]
    end

    H_db -->|Mounts to| C_db
    H_bk -->|Mounts to| C_bk
    H_up -->|Mounts to| C_up
    H_lg -->|Mounts to| C_lg
```

---

## 2. 准备步骤

### 2.1 创建 NAS 物理目录

1. 登录群晖 DSM，打开 **File Station**。
2. 在 `docker` 共享文件夹下创建一个新文件夹，命名为 `ledger-two`。
3. 在 `ledger-two` 目录下创建以下四个子目录：
   - `data`：存放 SQLite 账本文件。
   - `backups`：存放日/周/月/手动备份文件。
   - `uploads`：存放未来导入账单的附件。
   - `logs`：存放系统运行日志。

> [!WARNING]
> **权限设置**：请右键点击 `ledger-two` 文件夹，选择 **属性 -> 权限**，确保 `System` 或运行 Container Manager 的管理员账号拥有对该目录的**读取和写入**权限，否则 SQLite 会抛出 `readonly database` 异常。

### 2.2 准备配置文件

将本地项目根目录下的 [docker-compose.yml](file:///Users/polar/code/study_space/code/project/ledger_two/docker-compose.yml) 与 [.env.example](file:///Users/polar/code/study_space/code/project/ledger_two/.env.example) 复制到群晖 NAS 的 `/volume1/docker/ledger-two/` 目录下。

将 `.env.example` 重命名为 `.env`，并修改配置如下：

```text
APP_ENV=production
PORT=8080
DB_DSN=/app/data/ledger.db
JWT_SECRET=【请在此处修改为一个足够长且随机的字符串，例如 32位 秘钥】
BACKUP_DIR=/app/backups
TZ=Asia/Shanghai
```

---

## 3. 安装与运行

群晖 DSM 7.x 提供了直观的 Web 管理页面：

### 方法 A：通过群晖 DSM 界面操作 (推荐)

1. 打开群晖 **Container Manager**。
2. 导航至 **项目 (Project)**，点击 **新增**。
3. 设置项目名称（如 `ledger-two`），并在“路径”中选择刚才创建的 `/volume1/docker/ledger-two` 目录。
4. 来源选择 **使用现有的 docker-compose.yml**。
5. 确认配置无误，点击 **下一步 -> 应用**。系统会自动拉取镜像并启动容器。

### 方法 B：通过 SSH 命令行操作

若您熟悉 SSH，也可以登入群晖执行：

```bash
cd /volume1/docker/ledger-two
docker-compose up -d
```

---

## 4. 部署验证流程 (验收标准)

部署成功后，必须通过以下四步完成线上健康状态的校验：

### Step 1: 健康检查确认
访问大屏健康探测端点（默认群晖映射端口为 `8088`）：
```text
http://【NAS_IP】:8088/api/healthz
```
预期响应：
```json
{
  "success": true,
  "data": {
    "status": "ok",
    "db": "ok",
    "version": "0.2.0"
  }
}
```
*提示：如果响应中 `"db": "error"`，代表数据库文件权限不足，需按照 2.1 节重新修改文件夹权限。*

### Step 2: 登录与Onboarding配置
打开 `http://【NAS_IP】:8088`，页面会自动重定向到 `/init` 账本引导页。输入账本名称和两个成员的信息，系统将自动创建初始分类与账户。

### Step 3: 重启持久性校验
1. 在大屏记账一笔（如 100元）。
2. 在群晖 Container Manager 的“项目”中，点击 **重启** 该项目。
3. 重启成功后，再次刷新页面，验证刚刚记录的 100 元流水依然存在，证明数据成功落地在 NAS 的 `data/ledger.db` 中，未写入容器易失层。

### Step 4: 手动备份物理确认
1. 进入“系统设置”页面，点击 **立即创建手动安全备份** 按钮。
2. 弹出高风险确认弹窗，点击 **确认**。
3. 打开 NAS File Station 查阅 `backups/manual/` 目录，确认是否生成了带有当前时间戳的 `.db` 备份镜像。
