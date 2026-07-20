# Windows PC 开发环境详细配置教程 v0.3

## 1. 推荐方案

Windows 端推荐：

```text
Windows 11
  ├─ VSCode
  ├─ Codex VSCode Extension
  ├─ WSL2 Ubuntu 24.04 / 22.04
  ├─ Docker Desktop + WSL2 backend
  ├─ Git for Windows
  ├─ Windows Terminal / WezTerm
  └─ 在 WSL2 内安装 Go、Node、pnpm、SQLite
```

不要把 Go/Node 项目放在 Windows 盘再通过 WSL 访问。推荐放在：

```text
/home/polar/Projects/ledger_two
```

而不是：

```text
/mnt/c/Users/xxx/ledger_two
```

## 2. 安装 WSL2

用管理员 PowerShell：

```powershell
wsl --install
```

安装完成后重启。查看状态：

```powershell
wsl --status
wsl -l -v
```

如果没有 Ubuntu：

```powershell
wsl --install -d Ubuntu-24.04
```

进入 Ubuntu：

```powershell
wsl
```

## 3. 初始化 Ubuntu

```bash
sudo apt update
sudo apt upgrade -y
sudo apt install -y \
  build-essential \
  curl \
  wget \
  git \
  make \
  unzip \
  ca-certificates \
  sqlite3 \
  pkg-config \
  software-properties-common
```

## 4. 安装 Go

推荐使用官方 tar 包或 apt 新版本。简单方案：

```bash
sudo apt install -y golang-go
```

验证：

```bash
go version
```

如果 apt 版本太旧，再手动安装 Go 1.22+。

配置 Go：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w CGO_ENABLED=1

echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

安装后端工具：

```bash
go install github.com/air-verse/air@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

验证：

```bash
air -v
goose -version
sqlc version
```

## 5. 安装 Node.js + pnpm

推荐使用 nvm：

```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
source ~/.bashrc
nvm install --lts
nvm use --lts
corepack enable
corepack prepare pnpm@latest --activate
```

验证：

```bash
node -v
pnpm -v
```

## 6. Git 配置

在 WSL Ubuntu 内配置：

```bash
git config --global user.name "polarbao"
git config --global user.email "你的 GitHub 邮箱"
git config --global core.autocrlf input
git config --global init.defaultBranch main
```

## 7. GitHub SSH

在 WSL 内生成：

```bash
ssh-keygen -t ed25519 -C "你的 GitHub 邮箱"
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
cat ~/.ssh/id_ed25519.pub
```

复制输出内容，到 GitHub：

```text
GitHub -> Settings -> SSH and GPG keys -> New SSH key
```

测试：

```bash
ssh -T git@github.com
```

## 8. 克隆项目

在 WSL 内：

```bash
mkdir -p ~/Projects
cd ~/Projects
git clone git@github.com:polarbao/ledger_two.git
cd ledger_two
```

## 9. 安装 VSCode

Windows 上安装 VSCode，然后安装插件：

```text
Remote - WSL
Codex
Go
ESLint
Prettier
Tailwind CSS IntelliSense
Docker
GitLens
REST Client
SQLite Viewer
```

从 WSL 打开项目：

```bash
cd ~/Projects/ledger_two
code .
```

VSCode 左下角应显示：

```text
WSL: Ubuntu
```

## 10. 安装 Docker Desktop

Windows 安装 Docker Desktop。

设置中确认：

```text
Settings -> General -> Use the WSL 2 based engine
Settings -> Resources -> WSL Integration -> Enable Ubuntu
```

在 WSL 内验证：

```bash
docker version
docker compose version
```

## 11. Codex 配置

WSL 内创建全局配置：

```bash
mkdir -p ~/.codex
nano ~/.codex/config.toml
```

内容：

```toml
approval_policy = "on-request"
sandbox_mode = "workspace-write"

[features]
shell_snapshot = true
```

项目根目录应有：

```text
.codex/config.toml
AGENTS.md
```

使用 Codex 时要求它先读：

```text
docs/00-文档索引.md
AGENTS.md
docs/13-演示版本范围锁定.md
```

## 12. 本地开发启动

### 12.1 后端

```bash
cd ~/Projects/ledger_two/backend
mkdir -p data/backups data/uploads
go mod tidy
go run ./cmd/server
```

### 12.2 前端

另开 WSL 终端：

```bash
cd ~/Projects/ledger_two/frontend
pnpm install
pnpm dev --host 0.0.0.0
```

浏览器访问：

```text
http://localhost:5173
```

## 13. Docker 验证

仓库根目录：

```bash
cd ~/Projects/ledger_two
docker compose up -d --build
docker compose logs -f
```

停止：

```bash
docker compose down
```

## 14. Windows 常见问题

### 14.1 VSCode 打开的不是 WSL 项目

错误路径示例：

```text
C:\Users\xxx\ledger_two
```

正确路径应在 VSCode 左下角显示 WSL，终端路径类似：

```text
/home/polar/Projects/ledger_two
```

### 14.2 文件换行问题

确保：

```bash
git config --global core.autocrlf input
```

VSCode settings 中使用：

```json
"files.eol": "\n"
```

### 14.3 Docker 在 WSL 内不可用

检查 Docker Desktop：

```text
Settings -> Resources -> WSL Integration -> Enable integration with Ubuntu
```

然后重启 Docker Desktop 和 WSL：

```powershell
wsl --shutdown
```

### 14.4 端口访问失败

检查服务是否运行：

```bash
ss -ltnp | grep 8080
ss -ltnp | grep 5173
```

Vite 必须使用：

```bash
pnpm dev --host 0.0.0.0
```

### 14.5 不要在 /mnt/c 下开发

不要这样：

```bash
cd /mnt/c/Users/xxx/Desktop/ledger_two
```

推荐：

```bash
cd ~/Projects/ledger_two
```

性能、权限、Docker 挂载都会更稳定。
