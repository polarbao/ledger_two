# Mac Air 开发环境配置

## 推荐组合

Mac Air 建议采用原生 macOS 开发：

```text
VSCode + Codex + Homebrew + Go + Node.js LTS + pnpm + Docker Desktop + SQLite
```

日常开发使用 Go 本地后端 + Vite 本地前端；Docker 主要用于上线前验证和群晖 NAS 部署一致性。

## 1. 安装 Homebrew

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

Apple Silicon Mac 的 Homebrew 通常位于 `/opt/homebrew`。

## 2. 安装基础工具

```bash
brew install git go node pnpm sqlite make curl wget
brew install --cask visual-studio-code docker tailscale
```

可选工具：

```bash
brew install --cask github desktop dbeaver-community
```

## 3. Go 环境

```bash
go version
go env -w GOPROXY=https://goproxy.cn,direct
go env -w CGO_ENABLED=1
```

安装后端开发工具：

```bash
go install github.com/air-verse/air@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

确保 Go 工具在 PATH：

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

## 4. Node.js / pnpm

```bash
corepack enable
corepack prepare pnpm@latest --activate
node -v
pnpm -v
```

## 5. Docker Desktop

安装后打开 Docker Desktop，确认 Docker Engine 正常运行：

```bash
docker version
docker compose version
```

## 6. VSCode 插件

必装：

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

默认访问：

```text
http://localhost:5173
```
