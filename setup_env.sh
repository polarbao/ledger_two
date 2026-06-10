#!/bin/bash
set -e

echo "=== Installing Go 1.26.4 ==="
if [ ! -f go1.26.4.linux-amd64.tar.gz ]; then
  wget https://golang.google.cn/dl/go1.26.4.linux-amd64.tar.gz
fi
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
rm -f go1.26.4.linux-amd64.tar.gz

# 配置 Go 环境变量
if ! grep -q "GOPROXY" ~/.bashrc; then
  echo 'export PATH=$PATH:/usr/local/go/bin:$(go env GOPATH)/bin' >> ~/.bashrc
  echo 'export GOPROXY=https://goproxy.cn,direct' >> ~/.bashrc
fi

# 临时启用当前 session 的 Go 环境变量以供后续脚本使用
export PATH=$PATH:/usr/local/go/bin
export GOPROXY=https://goproxy.cn,direct

echo "=== Installing NVM & Node ==="
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

# 加载 NVM 环境变量
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

nvm install --lts
nvm use --lts

echo "=== Enabling pnpm ==="
corepack enable
corepack prepare pnpm@latest --activate

echo "=== Installing backend tools ==="
# 安装开发必备辅助工具
go install github.com/air-verse/air@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

echo "=== Env Setup Complete! ==="
