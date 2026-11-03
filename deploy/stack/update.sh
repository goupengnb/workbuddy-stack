#!/bin/sh
# 升级栈到指定版本：sh update.sh 2.11.10
# 步骤：备份 → 改 .env 版本 → 拉镜像 → 重建 → 等健康检查
set -eu
cd "$(dirname "$0")"

NEW="${1:-}"
if [ -z "$NEW" ]; then
  echo "用法: sh update.sh <版本号>（例如 2.11.10）" >&2
  exit 2
fi

sh backup.sh

if [ -f .env ] && grep -q '^WB_VERSION=' .env; then
  sed -i "s/^WB_VERSION=.*/WB_VERSION=$NEW/" .env
else
  echo "WB_VERSION=$NEW" >> .env
fi

docker compose pull
docker compose up -d

i=0
while [ $i -lt 30 ]; do
  if docker exec workbuddy2api wget -qO- http://127.0.0.1:7863/healthz >/dev/null 2>&1 \
     && docker inspect -f '{{.State.Health.Status}}' workbuddy-manager 2>/dev/null | grep -q healthy; then
    echo "升级完成：$NEW"
    exit 0
  fi
  i=$((i + 1))
  sleep 2
done

echo "升级后健康检查未通过，请查看 docker compose logs" >&2
exit 1
