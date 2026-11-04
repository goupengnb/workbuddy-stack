package upstream

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

// identityJSON 出站客户端身份的唯一真值来源。
//
// 为什么单独抽一份：同一账号的请求来自三条通道（Go 主进程、Python 任务脚本、
// 登录流程）。版本号或域名只要有一处不同，上游看到的就是「同一台机器上跑着两个
// 不同版本的客户端」，这是比 IP 更硬的关联特征。把常量收进 JSON 后，三条通道
// 读同一份数据，改版本只改一个文件。
//
//go:embed identity.json
var identityJSON []byte

// Identity 客户端身份配置。
type Identity struct {
	ClientVersionCN string `json:"client_version_cn"`
	ClientVersionGL string `json:"client_version_global"`
	CliVersion      string `json:"cli_version"`
	OriginCN        string `json:"origin_cn"`
	OriginGL        string `json:"origin_global"`
	PrivateDataCN   string `json:"private_data_cn_personal"`
	PrivateDataGL   string `json:"private_data_global_personal"`
	Source          string `json:"source"`
	Updated         string `json:"updated"`
}

var (
	identityOnce sync.Once
	identity     Identity
)

// loadIdentity 解析内嵌的 identity.json。解析失败不 panic：回落下面的内置默认值，
// 保证网关在配置损坏时仍能启动（宁可版本旧一点，也不要起不来）。
func loadIdentity() Identity {
	identityOnce.Do(func() {
		id := Identity{
			ClientVersionCN: "5.9.3",
			ClientVersionGL: "5.9.4",
			CliVersion:      "2.165.0",
			OriginCN:        "https://www.workbuddy.cn",
			OriginGL:        "https://www.workbuddy.ai",
			PrivateDataCN:   "false",
			PrivateDataGL:   "true",
		}
		var parsed Identity
		if err := json.Unmarshal(identityJSON, &parsed); err == nil {
			if parsed.ClientVersionCN != "" {
				id.ClientVersionCN = parsed.ClientVersionCN
			}
			if parsed.ClientVersionGL != "" {
				id.ClientVersionGL = parsed.ClientVersionGL
			}
			if parsed.CliVersion != "" {
				id.CliVersion = parsed.CliVersion
			}
			if parsed.OriginCN != "" {
				id.OriginCN = strings.TrimSuffix(parsed.OriginCN, "/")
			}
			if parsed.OriginGL != "" {
				id.OriginGL = strings.TrimSuffix(parsed.OriginGL, "/")
			}
			if parsed.PrivateDataCN != "" {
				id.PrivateDataCN = parsed.PrivateDataCN
			}
			if parsed.PrivateDataGL != "" {
				id.PrivateDataGL = parsed.PrivateDataGL
			}
			id.Source = parsed.Source
			id.Updated = parsed.Updated
		}
		identity = id
	})
	return identity
}

// IdentityConfig 返回当前生效的身份配置（只读快照）。
func IdentityConfig() Identity {
	return loadIdentity()
}

// desktopVersionFor 按 realm 返回桌面端版本号。
func desktopVersionFor(global bool) string {
	id := loadIdentity()
	if global {
		return id.ClientVersionGL
	}
	return id.ClientVersionCN
}
