#!/bin/bash
# run_tests.sh - 一键运行 LedgerTwo 前后端测试和静态质量检查
set -e

# 颜色控制
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}=== Starting LedgerTwo Quality Checks ===${NC}"

# 1. 运行 Go 后端测试
echo -e "\n${GREEN}--> Running Backend Tests...${NC}"
cd backend
if command -v go >/dev/null 2>&1; then
  go test -v ./...
  echo -e "${GREEN}[PASS] Backend tests passed!${NC}"
else
  echo -e "${RED}[WARN] Go CLI is not installed, skipping backend tests.${NC}"
fi
cd ..

# 2. 运行前端 Lint & Tests
echo -e "\n${GREEN}--> Running Frontend Lint, Tests, and Build...${NC}"
cd frontend
if command -v pnpm >/dev/null 2>&1; then
  echo -e "${GREEN}--> Installing Frontend Dependencies...${NC}"
  pnpm install --frozen-lockfile
  
  echo -e "${GREEN}--> Running Eslint...${NC}"
  pnpm run lint
  
  echo -e "${GREEN}--> Running Vitest...${NC}"
  pnpm test
  
  echo -e "${GREEN}--> Building Frontend...${NC}"
  pnpm run build
  echo -e "${GREEN}[PASS] Frontend checks passed!${NC}"
else
  echo -e "${RED}[WARN] pnpm CLI is not installed, skipping frontend checks.${NC}"
fi
cd ..

echo -e "\n${GREEN}=== LedgerTwo Quality Checks Completed! ===${NC}"
