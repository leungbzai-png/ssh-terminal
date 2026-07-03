package sshsess

import (
	"errors"
	"net"
	"testing"
)

func TestClassifyErrorCategories(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want DiagCategory
	}{
		{"dns structured", &net.DNSError{Err: "no such host", Name: "bad.example"}, DiagDNS},
		{"dns text", errors.New("dial tcp: lookup bad.example: no such host"), DiagDNS},
		{"refused", errors.New("dial: dial tcp 1.2.3.4:22: connect: connection refused"), DiagTCP},
		{"unreachable", errors.New("dial tcp: network is unreachable"), DiagTCP},
		{"auth", errors.New("ssh: unable to authenticate, attempted methods [none publickey]"), DiagAuth},
		{"permission", errors.New("ssh: permission denied"), DiagAuth},
		{"key", errors.New("parse key: ssh: this private key is passphrase protected"), DiagKey},
		{"proxyjump", errors.New("proxy jump: dial bastion failed"), DiagProxyJump},
		{"forward", errors.New("local forward: listen tcp 127.0.0.1:8080: bind: address already in use"), DiagForward},
		{"handshake", errors.New("ssh: handshake failed: host key mismatch"), DiagHandshake},
		{"other", errors.New("something odd happened"), DiagOther},
	}
	for _, c := range cases {
		got, msg := classifyError(c.err)
		if got != c.want {
			t.Errorf("%s: classify = %q, want %q", c.name, got, c.want)
		}
		if msg == "" {
			t.Errorf("%s: empty message", c.name)
		}
	}
}

func TestClassifyTimeout(t *testing.T) {
	// A net.Error that reports Timeout() should classify as TCP.
	got, _ := classifyError(timeoutErr{})
	if got != DiagTCP {
		t.Fatalf("timeout classify = %q, want %q", got, DiagTCP)
	}
}

func TestDiagnoseErrorNil(t *testing.T) {
	if DiagnoseError(nil) != "" {
		t.Fatal("nil should diagnose to empty")
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }
