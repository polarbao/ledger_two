#!/bin/bash
# deploy_to_nas.sh - 辅助将项目推送至群晖并一键构建部署的脚本
set -e

NAS_IP="192.168.0.115"
DEFAULT_USER="admin"

echo "============================================="
echo "       LedgerTwo 群晖 NAS 远程部署助手       "
echo "============================================="
echo "NAS 目标 IP: $NAS_IP"
read -p "请输入群晖 NAS 的 SSH 登录用户名 [$DEFAULT_USER]: " NAS_USER
NAS_USER=${NAS_USER:-$DEFAULT_USER}

read -p "请输入群晖 NAS 的部署根目录 [/volume1/docker/ledger-two]: " NAS_DIR
NAS_DIR=${NAS_DIR:-/volume1/docker/ledger-two}

echo -e "\n1. 开始在群晖上创建必要的持久化目录与权限配置..."
# 通过 SSH 创建目录并设置权限
ssh -p 22 "${NAS_USER}@${NAS_IP}" "sudo mkdir -p ${NAS_DIR}/data ${NAS_DIR}/backups ${NAS_DIR}/uploads ${NAS_DIR}/logs ${NAS_DIR}/app && sudo chmod -R 777 ${NAS_DIR}"

echo -e "\n2. 开始同步项目源代码与配置文件至群晖 (将自动过滤临时文件与 node_modules)..."
# 使用 rsync 过滤大文件并传输
if command -v rsync >/dev/null 2>&1; then
  rsync -avz --delete \
    --exclude='node_modules/' \
    --exclude='.git/' \
    --exclude='frontend/dist/' \
    --exclude='backend/data/' \
    --exclude='*.db' \
    --exclude='.gemini/' \
    --exclude='.vscode/' \
    --exclude='.codex/' \
    --exclude='.dish/' \
    --exclude='.qodo/' \
    ./ "${NAS_USER}@${NAS_IP}:${NAS_DIR}/app/"
else
  echo "本地未安装 rsync，改用 scp 传输，这可能需要稍长时间..."
  # 如果没有 rsync，创建一个临时的排除包并用 scp 传输
  TEMP_TAR="/tmp/ledger-two-deploy.tar.gz"
  tar --exclude='node_modules' --exclude='.git' --exclude='frontend/dist' --exclude='backend/data' --exclude='*.db' -czf "$TEMP_TAR" .
  scp "$TEMP_TAR" "${NAS_USER}@${NAS_IP}:${NAS_DIR}/"
  ssh "${NAS_USER}@${NAS_IP}" "tar -xzf ${NAS_DIR}/ledger-two-deploy.tar.gz -C ${NAS_DIR}/app/ && rm -f ${NAS_DIR}/ledger-two-deploy.tar.gz"
fi

echo -e "\n3. 初始化群晖端的配置文件..."
# 检查 NAS 上是否存在 .env，若不存在则拷贝一份
ssh "${NAS_USER}@${NAS_IP}" "if [ ! -f ${NAS_DIR}/app/.env ]; then cp ${NAS_DIR}/app/.env.example ${NAS_DIR}/app/.env && echo '已自动生成默认 .env 文件，请记得稍后在 NAS 上修改 JWT_SECRET'; fi"

# 把 docker-compose.yml 放到 /volume1/docker/ledger-two 根目录
ssh "${NAS_USER}@${NAS_IP}" "cp ${NAS_DIR}/app/docker-compose.yml ${NAS_DIR}/docker-compose.yml && cp ${NAS_DIR}/app/.env.example ${NAS_DIR}/.env"

echo -e "\n4. 尝试在群晖 NAS 上远程执行 Docker Compose 构建与拉起..."
echo "将会请求 sudo 权限以运行 docker，请输入群晖密码："
ssh -t "${NAS_USER}@${NAS_IP}" "cd ${NAS_DIR} && sudo docker compose down && sudo docker compose up -d --build"

echo -e "\n============================================="
echo "部署指令已远程发送！"
echo "请访问以下地址验证是否部署成功："
echo "健康检查：http://${NAS_IP}:8088/api/healthz"
echo "系统主页：http://${NAS_IP}:8088"
echo "============================================="
