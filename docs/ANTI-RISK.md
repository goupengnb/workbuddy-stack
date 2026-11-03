# 出站身份与反风控基线

这份文档记录本分支对「上游如何识别网关」的判断依据、当前的出站形态，以及可调的
杠杆。所有结论都来自实抓包、上游错误码和本机运行数据，不来自猜测。

## 1. 当前运行基线（2026-11-03）

21 个 CN 账号，单机家宽出口，7 天窗口：

| 指标 | 数值 |
| --- | --- |
| 请求数 | 4849 |
| 非 200 | 0 |
| `account_risk_events` | 0 |
| prompt 缓存命中 | 97.8% |
| ≥20 万 token 的请求 | 602 |
| 单请求最大 prompt | 36.3 万 token |
| 峰值单日账面 token | 2.53 亿（含缓存命中） |

结论：存量请求没有触发 11140，当前出站头不是瓶颈；风险面在「单号消耗曲线」和
「出口集中度」。

## 2. 出站头基线

chat（`/v2/chat/completions`）当前发送：

- `User-Agent: WorkBuddy/<client_version> WorkBuddy/<client_version> CLI/<cli_version>`
- `X-IDE-Name` / `X-IDE-Type`: `WorkBuddy`，`X-IDE-Version`: client_version
- `X-Product: SaaS`
- `X-Agent-Intent: craft`、`X-Agent-Purpose: conversation`、`X-Agent-Type: main`
- `X-Private-Data: true`
- `X-Domain: www.codebuddy.cn`（CN）
- 会话头族：`X-Conversation-*`、`X-Request-ID`、`X-Root-Request-ID`、`X-Trace-ID`、B3

**不发送**：`X-Machine-ID`、`X-Session-ID`（这两个只在 billing/反馈域）、
`X-Device-Token`（没有真实 Turing Shield token 时不伪造）、`X-No-*` 占位头。

依据：

- 5.8.3/5.8.4 桌面端抓包：对话请求不带 `X-Machine-ID` / `X-Session-ID`，那是反馈
  上报用的；网关若每请求都带按 uid 恒定的派生值，反而是最明显的网关特征。
- `X-Product` 官方取值是 `SaaS`，`X-IDE-*` 才是 `WorkBuddy`；旧版 `X-Product:
  WorkBuddy` 是自创组合。
- 官方桌面端不发 `X-No-Enterprise-Id` / `X-No-Department-Info`，缺省即无企业。

## 3. 错误码语义（不要混）

| 码 | 含义 | 处理 |
| --- | --- | --- |
| `11-128` + `request illegal` | 渠道指纹校验失败，请求不像官方客户端 | 修头/换通道，**不换号、不计风控** |
| `11140` + 异常行为文案 | 账号级风控，token 仍有效 | 冷却 + 换号，重登无效 |
| `11102` | 该后端无此模型 | 模型级负缓存，切模型 |
| `11150` | `reasoning_effort` 非法 | 参数修正 |

把 11-128 当封号，会把健康号池打成冷却；把 11140 当登录失效，会反复重登。

## 4. 可调杠杆（按收益排序）

1. **出口分散**：`config.json` 的 `proxies` 定义命名线路，账号 JSON 顶层 `proxy`
   绑定线路名。保留少数据主力号直连以维持家宽延迟，其余走独立出口。
2. **单号消耗曲线**：`pool.large_request_bytes` + `pool.large_request_cooldown`
   控制超大上下文轮换；`max_in_flight` 控制并发。当前 2 MiB / 180s / 2 并发。
3. **调度错峰**：`schedule.random_delay_minutes`、`account_delay_*` 已开，签到分散
   在 9/21 点。
4. **会话粘性**：`session_sticky.ttl=30m`，保证同会话落同一账号，缓存不碎。

## 5. 不要做的事

- 伪造 `X-Device-Token`：上游会校验，假 token 比没有更容易被判异常。
- 随机化 UA：官方客户端 UA 是确定性的，随机化本身就是特征。
- 在 11140 上反复重试或重登：账号级限制，重试只会加重。
- 多号共用同一份 `X-Machine-ID` / 会话 ID：等于把账号串成一台设备。

## 6. 观测方法

```sh
# 账号级风控事件（面板已落库，正常应为 0）
docker exec workbuddy-manager python3 -c \
  "import sqlite3;print(sqlite3.connect('/app/data/manager.db').execute('select count(*) from account_risk_events').fetchone())"

# 日志里的风控/渠道校验
docker logs --since 24h workbuddy2api 2>&1 | grep -Ei '11140|11-128|appeal|illegal' | tail -20
```

面板「账号」页的冷却/禁用状态比日志更早反映问题：出现 `disabled` 且 `err_total`
上涨，先查该号最近 24 小时的 prompt 规模，再决定是否降并发或换出口。
