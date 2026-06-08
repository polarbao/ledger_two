# Mac Air 开发环境详细配置教程 v0.3

## 1. 适用设备

本文档适用于 MacBook Air，Apple Silicon 或 Intel 均可。推荐在 macOS 上直接开发，不需要虚拟机。

## 2. 最终环境目标

```text
macOS
  ├─ VSCode
  ├─ Codex VSCode Extension
  ├─ Git + GitHub SSH
  ├─ Homebrew
  ├─ Go 1.22+
  ├─ Node.js LTS + pnpm
  ├─ SQLite CLI
  ├─ Docker Desktop
  ├─ Optional: DBeaver / SQLiteStudio
  └─ Optional: Tailscale
```

## 3. 安装 Homebrew

打开 Terminal：

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

Apple Silicon 通常需要加入 PATH：

```bash
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
source ~/.zprofile
```

验证：

```bash
brew --version
```

## 4. 安装基础工具

```bash
brew install git go node pnpm sqlite pkg-config make
brew install --cask visual-studio-code docker tailscale dbeaver-community
```

验证：

```bash
git --version
go version
node -v
pnpm -v
sqlite3 --version
```

## 5. 配置 Git

```bash
git config --global user.name "polarbao"
git config --global user.email "你的 GitHub 邮箱"
git config --global core.autocrlf input
git config --global init.defaultBranch main
```

## 6. 配置 GitHub SSH

生成密钥：

```bash
ssh-keygen -t ed25519 -C "你的 GitHub 邮箱"
```

启动 agent：

```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
```

复制公钥：

```bash
pbcopy < ~/.ssh/id_ed25519.pub
```

到 GitHub：

```text
GitHub -> Settings -> SSH and GPG keys -> New SSH key
```

测试：

```bash
ssh -T git@github.com
```

## 7. 克隆项目

推荐目录：

```bash
mkdir -p ~/Projects
cd ~/Projects
git clone git@github.com:polarbao/ledger_two.git
cd ledger_two
```

如果使用 HTTPS：

```bash
git clone https://github.com/polarbao/ledger_two.git
```

## 8. VSCode 打开项目

```bash
code .
```

如果 `code` 命令不可用：

```text
VSCode -> Command Palette -> Shell Command: Install 'code' command in PATH
```

## 9. 安装 VSCode 插件

打开项目后，VSCode 会根据 `.vscode/extensions.json` 推荐插件。至少安装：

```text
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

## 10. 配置 Codex

项目内已经包含：

```text
.codex/config.toml
AGENTS.md
```

建议全局也创建：

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

原则：

1. 不给 Codex 全盘写权限。
2. 命令执行需要确认。
3. 每次让 Codex 先读 `docs/00_DOCUMENT_INDEX.md` 和 `AGENTS.md`。

## 11. Go 后端工具

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w CGO_ENABLED=1

go install github.com/air-verse/air@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

确保 Go bin 在 PATH：

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zprofile
source ~/.zprofile
```

验证：

```bash
air -v
goose -version
sqlc version
```

## 12. Node / pnpm

如果 Homebrew 已安装 pnpm：

```bash
pnpm -v
```

也可以用 corepack：

```bash
corepack enable
corepack prepare pnpm@latest --activate
```

## 13. Docker Desktop

打开 Docker Desktop，等待状态变为 Running。

验证：

```bash
docker version
docker compose version
```

## 14. 本地启动方式

### 14.1 后端本地启动

```bash
cd ~/Projects/ledger_two/backend
mkdir -p data/backups data/uploads
go mod tidy
go run ./cmd/server
```

后端地址：

```text
http://localhost:8080
```

### 14.2 前端本地启动

另开终端：

```bash
cd ~/Projects/ledger_two/frontend
pnpm install
pnpm dev
```

前端地址：

```text
http://localhost:5173
```

### 14.3 Docker 启动

在仓库根目录：

```bash
docker compose up -d --build
docker compose logs -f
```

## 15. Mac 常见问题

### 15.1 `.vscode` / `.codex` 看不到

Finder 按：

```text
Command + Shift + .
```

### 15.2 SQLite CGO 报错

确保安装：

```bash
xcode-select --install
brew install pkg-config sqlite
```

### 15.3 端口被占用

```bash
lsof -i :8080
lsof -i :5173
```

结束进程：

```bash
kill -9 <PID>
```

### 15.4 Docker 挂载慢

项目尽量放在本机目录，例如：

```text
~/Projects/ledger_two
```

不要放在 iCloud 同步目录。
