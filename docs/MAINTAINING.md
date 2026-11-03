# 维护手册

## 仓库结构

```text
server/           管理端后端（FastAPI）
web/              管理端前端（Next.js）
upstream/         上游 workbuddy2api 源码（Go，随仓库维护）
deploy/stack/     一体化 compose、备份与升级脚本
docs/             本手册与反风控基线
.github/workflows ci.yml（测试）/ release.yml（发版 + 镜像）
```

`upstream/` 不是子模块：原公开仓库（`Sliverkiss/workbuddy2api`）已被作者删除，本仓库
把源码作为唯一事实来源维护，改动直接提交在这里。

## 同步上游管理端

```sh
git remote add upstream https://github.com/ithtelab/workbuddy-manager.git  # 只需一次
git fetch upstream
git merge upstream/main
```

冲突高发区是 `server/`、`web/`、`deploy/`。`upstream/`、`deploy/stack/`、`docs/ANTI-RISK.md`
是本仓库独有，不会冲突。

## 修改上游（2api）

```sh
cd upstream
go test ./...
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o wb2api ./cmd/server
```

改完必须跑全量测试；涉及出站头时同步更新 `internal/upstream/*_test.go` 的断言，别让
测试继续保护旧行为。

线上临时验证可以只替换二进制（bind mount 到 `/app/wb2api`），长期以镜像为准。

## 发版

```sh
git tag v2.11.9
git push origin v2.11.9
```

`release.yml` 会：

1. 构建前端静态产物、跑后端单测；
2. 校验 `upstream/` 能否编译、单测是否通过；
3. 用仓库内 `upstream/` 构建两个多架构镜像：
   `ghcr.io/<owner>/workbuddy2api:<version>`、`ghcr.io/<owner>/workbuddy-manager:<version>`；
4. 创建 **草稿** Release，核对后 `gh release edit v2.11.9 --draft=false` 发布。

首次发布后要去 GitHub Packages 把两个容器包设为 **public**，否则未登录的机器
`docker pull` 会 401。

## 部署与升级

```sh
cd deploy/stack
sh update.sh 2.11.9    # 备份 → 改 .env → 拉镜像 → 重建 → 健康检查
sh backup.sh           # 只备份
```

回滚：把 `.env` 的 `WB_VERSION` 改回旧版本，`docker compose up -d`；数据在
`workbuddy2api/auths`、`workbuddy2api/data`、`workbuddy-manager/data`，与镜像无关。

## 版本策略

- 跟上游管理端版本号走（`v2.11.x`），本仓库补丁不单独占号。
- 破坏性改动或需要重建数据时，在 `CHANGELOG.md` 顶部写「升级前必读」。
- 每个 tag 的镜像 digest 会写进 Release 说明，便于核对线上跑的是不是同一份。
