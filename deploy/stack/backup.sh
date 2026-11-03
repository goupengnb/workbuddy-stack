#!/bin/sh
# 备份栈的凭据与数据（不含镜像）。用法：sh backup.sh
set -eu
cd "$(dirname "$0")"

STAMP=$(date +%Y%m%d%H%M%S)
OUT="backups/workbuddy-$STAMP.tar.gz"
mkdir -p backups

tar czf "$OUT" \
  workbuddy2api/auths \
  workbuddy2api/config.json \
  workbuddy-manager/data \
  compose.yaml \
  .env 2>/dev/null || true

chmod 600 "$OUT"
echo "已备份到 $OUT"

# 只保留最近 14 份
ls -1t backups/workbuddy-*.tar.gz 2>/dev/null | tail -n +15 | xargs -r rm -f
