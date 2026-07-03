package sshsess

import (
	"bytes"
	"testing"
)

func TestSocksNegotiateNoAuth(t *testing.T) {
	// Client greeting: v5, 1 method, no-auth.
	in := bytes.NewBuffer([]byte{0x05, 0x01, 0x00})
	out := &bytes.Buffer{}
	rw := &readWriter{r: in, w: out}
	if err := socksNegotiate(rw); err != nil {
		t.Fatalf("negotiate: %v", err)
	}
	got := out.Bytes()
	if len(got) != 2 || got[0] != 0x05 || got[1] != 0x00 {
		t.Fatalf("reply = %v, want [5 0]", got)
	}
}

func TestSocksNegotiateRejectsBadVersion(t *testing.T) {
	in := bytes.NewBuffer([]byte{0x04, 0x01, 0x00})
	rw := &readWriter{r: in, w: &bytes.Buffer{}}
	if err := socksNegotiate(rw); err == nil {
		t.Fatal("expected error for non-v5 greeting")
	}
}

func TestSocksReadRequestIPv4(t *testing.T) {
	// VER, CMD=CONNECT, RSV, ATYP=IPv4, 1.2.3.4, port 80
	req := []byte{0x05, 0x01, 0x00, 0x01, 1, 2, 3, 4, 0x00, 0x50}
	target, err := socksReadRequest(bytes.NewReader(req))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if target != "1.2.3.4:80" {
		t.Fatalf("target = %q, want 1.2.3.4:80", target)
	}
}

func TestSocksReadRequestDomain(t *testing.T) {
	host := "example.com"
	req := []byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}
	req = append(req, []byte(host)...)
	req = append(req, 0x01, 0xBB) // port 443
	target, err := socksReadRequest(bytes.NewReader(req))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if target != "example.com:443" {
		t.Fatalf("target = %q, want example.com:443", target)
	}
}

func TestSocksReadRequestIPv6(t *testing.T) {
	req := []byte{0x05, 0x01, 0x00, 0x04}
	ip := []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}
	req = append(req, ip...)
	req = append(req, 0x1F, 0x90) // port 8080
	target, err := socksReadRequest(bytes.NewReader(req))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if target != "[2001:db8::1]:8080" {
		t.Fatalf("target = %q, want [2001:db8::1]:8080", target)
	}
}

func TestSocksReadRequestRejectsBind(t *testing.T) {
	// CMD=0x02 (BIND) is unsupported.
	req := []byte{0x05, 0x02, 0x00, 0x01, 1, 2, 3, 4, 0x00, 0x50}
	if _, err := socksReadRequest(bytes.NewReader(req)); err == nil {
		t.Fatal("expected error for non-CONNECT command")
	}
}

func TestSocksReadRequestRejectsBadATYP(t *testing.T) {
	req := []byte{0x05, 0x01, 0x00, 0x09, 1, 2}
	if _, err := socksReadRequest(bytes.NewReader(req)); err == nil {
		t.Fatal("expected error for unknown address type")
	}
}

func TestSocksReadRequestTruncated(t *testing.T) {
	// Domain length says 20 but no bytes follow.
	req := []byte{0x05, 0x01, 0x00, 0x03, 20}
	if _, err := socksReadRequest(bytes.NewReader(req)); err == nil {
		t.Fatal("expected error for truncated domain")
	}
}

func TestSocksWriteReply(t *testing.T) {
	out := &bytes.Buffer{}
	if err := socksWriteReply(out, socksRepOK); err != nil {
		t.Fatalf("write reply: %v", err)
	}
	got := out.Bytes()
	if len(got) != 10 || got[0] != 0x05 || got[1] != socksRepOK || got[3] != socksATYPv4 {
		t.Fatalf("reply = %v", got)
	}
}

// readWriter adapts a separate reader and writer into an io.ReadWriter for the
// negotiate test.
type readWriter struct {
	r *bytes.Buffer
	w *bytes.Buffer
}

func (rw *readWriter) Read(p []byte) (int, error)  { return rw.r.Read(p) }
func (rw *readWriter) Write(p []byte) (int, error) { return rw.w.Write(p) }
