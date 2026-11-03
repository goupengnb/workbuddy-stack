# 一体化部署

管理端 + 上游 2api 的容器编排，镜像来自本仓库的 Release 工作流。

## 首次部署

```sh
cd deploy/stack
cp .env.stack.example .env
mkdir -p workbuddy2api/auths workbuddy2api/data workbuddy-manager/data
cp ../../upstream/config.example.json workbuddy2api/config.json

# 容器以 uid 10001 运行，卷必须可写
chown -R 10001:10001 workbuddy2api workbuddy-manager

docker compose up -d
```

改 `workbuddy2api/config.json` 里的 `api_key`，再 `docker compose restart workbuddy2api`。
面板地址 `http://<host>:8888`，口令取 `.env` 的 `WB_ADMIN_PASSWORD`。

## 日常

| 操作 | 命令 |
| --- | --- |
| 备份凭据与数据 | `sh backup.sh` |
| 升级到新版本 | `sh update.sh 2.11.10` |
| 看日志 | `docker compose logs -f workbuddy2api` |
| 只重启上游 | `docker compose restart workbuddy2api` |

备份默认留 14 份，位于 `backups/`，已 chmod 600。

## 注意

- `workbuddy2api/auths/`、`data/`、`workbuddy-manager/data/` 都是运行时数据，已在根
  `.gitignore` 里排除，不要提交。
- 面板内「一键更新上游」需要挂载 `docker.sock`；本编排默认不挂，升级统一走
  `update.sh`，权限面更小。
- 两个镜像都支持 `linux/amd64` 与 `linux/arm64`（OpenWrt / 树莓派 / ARM 服务器可直接拉）。
