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

## 上游改了版本怎么办

反风控的本质是「出站身份与官方客户端一致」。上游一升版，版本号、域名、头集合
都可能变，不跟就会被判成旧客户端。

所有版本/域名常量集中在**一个文件**：

```
upstream/internal/upstream/identity.json
```

Go 主进程（`go:embed`）、Python 任务脚本（运行时读取）、登录流程都从它取。
只改这一处，三条通道同时生效。

### 每周自动检查

`.github/workflows/identity-watch.yml` 每周一比一次，三方对照：

| 来源 | 用途 |
| --- | --- |
| `identity.json` | 当前在用的身份 |
| npm `@tencent-ai/codebuddy-code` | 官方 CLI 版本 |
| `misakano7545/workbuddy2api-panel` | 社区真机抓包常量 |

有差异自动开 issue。手动跑：

```sh
cd upstream
python scripts/check_identity.py          # 人读
python scripts/check_identity.py --json   # 机器读，退出码 0=一致 1=有差异 2=失败
```

### 跟进顺序

1. 先看**头集合**，不要先改版本号。官方常在升版时同时增删请求头，只改版本号会
   留下「版本新、头旧」的组合，比不升更可疑。
2. 确认 CN/global 域名是否仍是 `workbuddy.cn` / `workbuddy.ai`。
3. 改 `identity.json`。
4. `go test ./...` 与 `python scripts/check_identity.py` 都要过。
5. 发版前用小号跑一轮真实对话，观察是否出现 11128 或 11140。

### 为什么不自动改

自动改会在「上游已升版、社区还没抓包」的窗口期把网关推到没人验证过的形态。
宁可滞后几天，也不要拿盲猜的身份去撞风控。

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
