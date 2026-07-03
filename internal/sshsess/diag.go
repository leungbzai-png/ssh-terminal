package sshsess

import (
	"errors"
	"net"
	"strings"
)

// Connection diagnostics (v0.8.0).
//
// classifyError maps a low-level connection error to a coarse, user-facing
// category and a short readable message. It never includes secret material —
// it works purely from the error's structure and a small set of substrings.

// DiagCategory is a coarse classification of a connection failure.
type DiagCategory string

const (
	DiagDNS       DiagCategory = "dns"       // name resolution failed
	DiagTCP       DiagCategory = "tcp"       // connect refused / timed out / unreachable
	DiagHandshake DiagCategory = "handshake" // SSH transport handshake failed
	DiagAuth      DiagCategory = "auth"      // authentication rejected
	DiagKey       DiagCategory = "key"       // key/passphrase parse/load failure
	DiagProxyJump DiagCategory = "proxyjump" // failure reaching/through the bastion
	DiagForward   DiagCategory = "forward"   // port-forward listener failure
	DiagOther     DiagCategory = "other"
)

// classifyError inspects err (already scrubbed of secrets by the caller when it
// may contain any) and returns a category plus a short Chinese explanation.
func classifyError(err error) (DiagCategory, string) {
	if err == nil {
		return DiagOther, ""
	}
	msg := err.Error()
	low := strings.ToLower(msg)

	// Structured DNS error is the most reliable signal.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return DiagDNS, "无法解析主机名（DNS 失败）"
	}
	if strings.Contains(low, "no such host") || strings.Contains(low, "server misbehaving") {
		return DiagDNS, "无法解析主机名（DNS 失败）"
	}

	// Key / passphrase problems (raised before any network use).
	if strings.Contains(low, "parse key") || strings.Contains(low, "passphrase") ||
		strings.Contains(low, "cannot decode encrypted private key") ||
		strings.Contains(low, "load managed key") {
		return DiagKey, "私钥或口令无效"
	}

	// Bastion / proxy jump failures are tagged by the dial helper.
	if strings.Contains(low, "proxy jump") || strings.Contains(low, "jump host") {
		return DiagProxyJump, "无法通过跳板机连接"
	}

	// Port-forward listener failures.
	if strings.Contains(low, "forward") && (strings.Contains(low, "listen") || strings.Contains(low, "bind")) {
		return DiagForward, "端口转发监听失败（可能端口被占用）"
	}

	// Authentication rejected by the server.
	if strings.Contains(low, "unable to authenticate") || strings.Contains(low, "auth") && strings.Contains(low, "fail") ||
		strings.Contains(low, "permission denied") || strings.Contains(low, "no supported methods remain") {
		return DiagAuth, "认证失败（用户名 / 密码 / 密钥不正确）"
	}

	// Network-level connect failures.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return DiagTCP, "连接超时（主机不可达或端口未开放）"
	}
	if strings.Contains(low, "timeout") || strings.Contains(low, "timed out") {
		return DiagTCP, "连接超时（主机不可达或端口未开放）"
	}
	if strings.Contains(low, "connection refused") || strings.Contains(low, "refused") {
		return DiagTCP, "连接被拒绝（端口未监听）"
	}
	if strings.Contains(low, "network is unreachable") || strings.Contains(low, "no route to host") {
		return DiagTCP, "网络不可达"
	}

	// SSH transport handshake (after TCP, before/around auth).
	if strings.Contains(low, "handshake") || strings.Contains(low, "host key") ||
		strings.Contains(low, "ssh:") {
		return DiagHandshake, "SSH 握手失败"
	}

	return DiagOther, "连接失败"
}

// DiagnoseError returns a single readable string combining the category label
// and the explanation. The caller is responsible for redacting secrets first.
func DiagnoseError(err error) string {
	if err == nil {
		return ""
	}
	_, msg := classifyError(err)
	return msg
}
