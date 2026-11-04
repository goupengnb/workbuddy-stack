#!/usr/bin/env python3
"""检查官方客户端版本是否变了，并告诉你该改哪一处。

为什么需要它：反风控的本质是「你的出站身份与官方客户端一致」。官方一发新版，
版本号、域名、头集合都可能变；不跟就会被判成旧客户端/非官方客户端。

本脚本只做三件事，不做自动改写：
  1. 读本地 identity.json（唯一真值来源）
  2. 查官方 CLI 的 npm 最新版（@tencent-ai/codebuddy-code）
  3. 查社区真机抓包仓库的 headers.go 里的客户端版本常量
然后把差异打印出来，并给出「改哪个文件的哪一行」。

自动改写交给 CI/人：版本号一改，需要重新跑真机抓包确认头集合没变，不能盲升。

用法：
  python check_identity.py            # 只报告
  python check_identity.py --json     # 机器可读，供 cron 判断
退出码：0 = 一致；1 = 有差异；2 = 网络/解析失败
"""
from __future__ import annotations

import json
import os
import re
import sys
import urllib.error
import urllib.request

HERE = os.path.dirname(os.path.abspath(__file__))
IDENTITY = os.path.join(HERE, "..", "internal", "upstream", "identity.json")

NPM_PKG = "https://registry.npmjs.org/@tencent-ai/codebuddy-code"
PANEL_HEADERS = (
    "https://raw.githubusercontent.com/misakano7545/workbuddy2api-panel/main/"
    "internal/upstream/headers.go"
)


def fetch(url: str, timeout: int = 25) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": "wb-identity-check"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode("utf-8", "replace")


def local_identity() -> dict:
    with open(os.path.normpath(IDENTITY), encoding="utf-8") as fh:
        return json.load(fh)


def npm_latest() -> str:
    return json.loads(fetch(NPM_PKG))["dist-tags"]["latest"]


def panel_constants() -> dict:
    text = fetch(PANEL_HEADERS)
    out = {}
    for key, pattern in (
        ("client_version_cn", r"defaultClientVersionCN\s*=\s*\"([^\"]+)\""),
        ("client_version_global", r"defaultClientVersionGlobal\s*=\s*\"([^\"]+)\""),
        ("cli_version", r"defaultCliVersion\s*=\s*\"([^\"]+)\""),
        ("origin_cn", r"originRefererCN\s*=\s*\"([^\"]+)\""),
    ):
        m = re.search(pattern, text)
        if m:
            out[key] = m.group(1)
    return out


def main() -> int:
    as_json = "--json" in sys.argv
    try:
        ident = local_identity()
        latest = npm_latest()
        panel = panel_constants()
    except (urllib.error.URLError, KeyError, ValueError, OSError) as exc:
        print("检查失败：%s" % exc, file=sys.stderr)
        return 2

    diffs = []
    for key, remote in panel.items():
        mine = ident.get(key)
        if mine and remote and mine != remote:
            diffs.append({"field": key, "local": mine, "upstream": remote})

    # npm 的 CLI 版本只用于提示：不同通道发版节奏不同，不一定是问题
    cli_npm_ahead = latest != ident.get("cli_version")

    result = {
        "identity_file": os.path.normpath(IDENTITY),
        "local": {k: ident.get(k) for k in
                  ("client_version_cn", "client_version_global",
                   "cli_version", "origin_cn")},
        "npm_latest": latest,
        "panel": panel,
        "diffs": diffs,
        "npm_cli_differs": cli_npm_ahead,
        "verdict": "diff" if diffs else "same",
    }

    if as_json:
        print(json.dumps(result, ensure_ascii=False, indent=2))
    else:
        print("本地 identity.json:")
        for k, v in result["local"].items():
            print("  %-22s %s" % (k, v))
        print("官方 npm CLI 最新版: %s" % latest)
        print("社区真机抓包常量:")
        for k, v in panel.items():
            print("  %-22s %s" % (k, v))
        if diffs:
            print("\n需要跟进：")
            for d in diffs:
                print("  %s: 本地 %s → 上游 %s"
                      % (d["field"], d["local"], d["upstream"]))
            print("\n改这一处即可（Go / Python 脚本 / 登录流程都读它）：")
            print("  %s" % os.path.normpath(IDENTITY))
            print("改完先跑真机比对再发版：头集合与域名可能一起变，"
                  "盲升版本号反而更可疑。")
        else:
            print("\n一致，无需改动。")

    return 1 if diffs else 0


if __name__ == "__main__":
    sys.exit(main())
