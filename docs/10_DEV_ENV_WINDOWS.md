# Windows PC 开发环境配置

## 推荐组合

Windows PC 建议采用：

```text
Windows 11 + WSL2 Ubuntu + VSCode Remote WSL + Docker Desktop WSL2 Backend
```

项目仓库建议放在 WSL 文件系统中：

```text
~/Code/ledger_two
```

不要放在 `/mnt/c/` 或 `C:\Users\...` 下，否则文件监听和 Docker volume 性能会变差。

## 1. 安装 WSL2

PowerShell 管理员执行：

```powershell
wsl --install
```

安装 Ubuntu 后重启并进入 Ubuntu。

## 2. Ubuntu 基础依赖

```bash
sudo apt update
sudo apt upgrade -y
sudo apt install -y build-essential curl wget git make unzip ca-certificates sqlite3 pkg-config
```

## 3. Go 环境

安装 Go 后执行：

```bash
go version
go env -w GOPROXY=https://goproxy.cn,direct
go env -w CGO_ENABLED=1
```

安装工具：

```bash
go install github.com/air-verse/air@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

加入 PATH：

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

## 4. Node.js + pnpm

推荐用 nvm：

```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/master/install.sh | bash
source ~/.bashrc
nvm install --lts
nvm use --lts
corepack enable
corepack prepare pnpm@latest --activate
```

检查：

```bash
node -v
pnpm -v
```

## 5. Docker Desktop

Windows 安装 Docker Desktop 后：

1. Settings -> General -> Use the WSL 2 based engine。
2. Settings -> Resources -> WSL Integration。
3. 开启 Ubuntu 集成。

在 Ubuntu 中检查：

```bash
docker version
docker compose version
```

## 6. VSCode 插件

Windows 端安装：

- Remote - WSL
- Codex
- Go
- ESLint
- Prettier
- Tailwind CSS IntelliSense
- Docker
- GitLens
- REST Client
- SQLite Viewer

## 7. 克隆项目

```bash
mkdir -p ~/Code
cd ~/Code
git clone https://github.com/polarbao/ledger_two.git
cd ledger_two
code .
```

VSCode 左下角应显示 `WSL: Ubuntu`。

## 8. 启动开发服务

后端：

```bash
cd backend
mkdir -p data/backups data/uploads
go mod tidy
go run ./cmd/server
```

前端：

```bash
cd frontend
pnpm install
pnpm dev
```

访问：

```text
http://localhost:5173
```

## 9. Windows 注意事项

- 仓库放在 WSL 内，不放 `/mnt/c/`。
- Docker Desktop 必须启用 WSL Integration。
- Git 换行统一 LF。
- 使用 VSCode Remote WSL 打开仓库。
- 端口异常时检查 Windows 防火墙和 Docker Desktop 状态。
