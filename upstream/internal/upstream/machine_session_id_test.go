package upstream

import (
	"regexp"
	"testing"

	"workbuddy2api/internal/auth"
)

// hex36Re 36 位 hex 合法格式断言（machineId/sessionId 派生形态）。
var hex36Re = regexp.MustCompile(`^[0-9a-f]{36}$`)

// TestDeriveAccountStableIDStable 同 uid + 同用途两次派生 → 恒同值（跨重启稳定语义：
// 固定盐纯派生，无随机源）。
func TestDeriveAccountStableIDStable(t *testing.T) {
	first := deriveAccountStableID("u1", "machine")
	second := deriveAccountStableID("u1", "machine")
	if first != second {
		t.Errorf("deriveAccountStableID(u1, machine) unstable: %q vs %q", first, second)
	}
}

// TestDeriveAccountStableIDDistinct 不同 uid → 不同值（账号间互异，防多号关联）。
func TestDeriveAccountStableIDDistinct(t *testing.T) {
	if deriveAccountStableID("u1", "machine") == deriveAccountStableID("u2", "machine") {
		t.Errorf("different uids derive same machine ID")
	}
}

// TestDeriveAccountStableIDPurposeIsolation 同 uid + "machine" vs "session" → 不同值
// （用途盐隔离：设备标识与账号固定会话标识不可碰撞）。
func TestDeriveAccountStableIDPurposeIsolation(t *testing.T) {
	if deriveAccountStableID("u1", "machine") == deriveAccountStableID("u1", "session") {
		t.Errorf("machine/session purposes derive same ID")
	}
}

// TestDeriveAccountStableIDFormat 输出恒为 36 hex（对齐 hub md5[:36] 同形态）。
func TestDeriveAccountStableIDFormat(t *testing.T) {
	for _, purpose := range []string{"machine", "session"} {
		got := deriveAccountStableID("uid-abc", purpose)
		if !hex36Re.MatchString(got) {
			t.Errorf("deriveAccountStableID(%q, %q) = %q, want 36 hex", "uid-abc", purpose, got)
		}
	}
}

// TestCommonHeadersOmitsMachineSessionID 官方 5.8.4 抓包：对话/刷新请求不带
// X-Machine-ID / X-Session-ID（只用于反馈上报），CommonHeaders 不得注入。
func TestCommonHeadersOmitsMachineSessionID(t *testing.T) {
	a := &auth.Auth{AccessToken: "at", UID: "u1"}
	req := mustRequest(t)
	c := &Client{}
	c.CommonHeaders(req, a)

	if got := req.Header.Get("X-Machine-ID"); got != "" {
		t.Errorf("X-Machine-ID = %q, want empty on chat/refresh path", got)
	}
	if got := req.Header.Get("X-Session-ID"); got != "" {
		t.Errorf("X-Session-ID = %q, want empty on chat/refresh path", got)
	}
}

// TestCommonHeadersEmptyUIDSkipsInjection uid 为空（匿名请求）→ 不注入两头且不 panic。
func TestCommonHeadersEmptyUIDSkipsInjection(t *testing.T) {
	req := mustRequest(t)
	c := &Client{}
	c.CommonHeaders(req, nil)

	if got := req.Header.Get("X-Machine-ID"); got != "" {
		t.Errorf("X-Machine-ID = %q, want empty (no auth)", got)
	}
	if got := req.Header.Get("X-Session-ID"); got != "" {
		t.Errorf("X-Session-ID = %q, want empty (no auth)", got)
	}
	req2 := mustRequest(t)
	c.CommonHeaders(req2, &auth.Auth{AccessToken: "at", UID: ""})
	if got := req2.Header.Get("X-Machine-ID"); got != "" {
		t.Errorf("X-Machine-ID = %q, want empty (empty uid)", got)
	}
	if got := req2.Header.Get("X-Session-ID"); got != "" {
		t.Errorf("X-Session-ID = %q, want empty (empty uid)", got)
	}
}

// TestChatHeadersOmitsMachineSessionID chat 出站不得带设备头；billing 域仍按
// 账号派生注入（两个头必须互异、跨账号互异）。
func TestChatHeadersOmitsMachineSessionID(t *testing.T) {
	a := &auth.Auth{AccessToken: "at", UID: "u9"}
	req := mustRequest(t)
	c := &Client{}
	c.ChatHeaders(req, a, "", ChatMeta{})

	if got := req.Header.Get("X-Machine-ID"); got != "" {
		t.Errorf("X-Machine-ID = %q, want empty on chat", got)
	}
	if got := req.Header.Get("X-Session-ID"); got != "" {
		t.Errorf("X-Session-ID = %q, want empty on chat", got)
	}

	breq := mustRequest(t)
	c.BillingHeaders(breq, a)
	machine := breq.Header.Get("X-Machine-ID")
	session := breq.Header.Get("X-Session-ID")
	if machine != deriveAccountStableID("u9", "machine") {
		t.Errorf("billing X-Machine-ID = %q, want account-derived", machine)
	}
	if session != deriveAccountStableID("u9", "session") {
		t.Errorf("billing X-Session-ID = %q, want account-derived", session)
	}
	if machine == session {
		t.Errorf("X-Machine-ID == X-Session-ID (must differ by purpose salt)")
	}
}
