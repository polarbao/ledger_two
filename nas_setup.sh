#!/bin/bash
# nas_setup.sh - 此脚本将在群晖 NAS 上由 root 权限运行
set -e

export PATH="/volume1/@appstore/ContainerManager/usr/bin:$PATH"
NAS_DIR="/volume1/docker/ledger-two"

echo "=== [NAS] 1. 创建物理目录 ==="
mkdir -p "${NAS_DIR}/data"
mkdir -p "${NAS_DIR}/backups"
mkdir -p "${NAS_DIR}/uploads"
mkdir -p "${NAS_DIR}/logs"
mkdir -p "${NAS_DIR}/app"

echo "=== [NAS] 2. 解压缩部署包 ==="
# 彻底删除并重建 app 目录，防止 macOS 隐藏垃圾文件 ._* 残留
rm -rf "${NAS_DIR}/app"
mkdir -p "${NAS_DIR}/app"
tar -xzf /tmp/ledger-two-deploy.tar.gz -C "${NAS_DIR}/app/"

echo "=== [NAS] 3. 授权所有目录 ==="
chmod -R 777 "${NAS_DIR}"

echo "=== [NAS] 4. 初始化配置文件 ==="
cp "${NAS_DIR}/app/docker-compose.yml" "${NAS_DIR}/docker-compose.yml"
sed -i 's/context: \./context: \.\/app/g' "${NAS_DIR}/docker-compose.yml"
if [ ! -f "${NAS_DIR}/.env" ]; then
  cp "${NAS_DIR}/app/.env.example" "${NAS_DIR}/.env"
  # 自动为用户生成一个随机的 JWT_SECRET 提高安全性
  RANDOM_SECRET=$(head -c 16 /dev/urandom | xxd -p)
  sed -i "s/replace-with-a-long-random-string/${RANDOM_SECRET}/g" "${NAS_DIR}/.env"
  echo "已自动生成高强度 JWT 密钥并配置完毕！"
fi

echo "=== [NAS] 5. 运行 Docker Compose 构建与拉起 ==="
cd "${NAS_DIR}"
# 执行构建
docker compose down || true
docker compose up -d --build

echo "=== [NAS] 6. 清理临时部署文件 ==="
rm -f /tmp/ledger-two-deploy.tar.gz
rm -f /tmp/nas_setup.sh

echo "=== [NAS] 部署圆满成功！ ==="
