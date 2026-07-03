package sshsess

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
)

// Minimal SOCKS5 server for dynamic port forwarding (v0.8.0).
//
// Only the no-authentication method and the CONNECT command are supported —
// enough for a browser/debug proxy tunnelled over SSH. The byte-parsing here is
// the bug-prone part, so it is factored into small pure functions that are unit
// tested directly (see socks_test.go); the accept/serve loop that wires these
// to an ssh.Client lives in tunnel.go.

const (
	socksVersion5 = 0x05
	socksNoAuth   = 0x00
	socksCmdConn  = 0x01
	socksRepOK    = 0x00
	socksRepErr   = 0x01

	socksATYPv4     = 0x01
	socksATYPDomain = 0x03
	socksATYPv6     = 0x04
)

// socksNegotiate performs the method-selection handshake, accepting only the
// no-authentication method. It reads the client greeting and writes the reply.
func socksNegotiate(rw io.ReadWriter) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(rw, header); err != nil {
		return err
	}
	if header[0] != socksVersion5 {
		return fmt.Errorf("socks: unsupported version 0x%02x", header[0])
	}
	nMethods := int(header[1])
	if nMethods > 0 {
		methods := make([]byte, nMethods)
		if _, err := io.ReadFull(rw, methods); err != nil {
			return err
		}
	}
	// Reply: version 5, no-auth.
	if _, err := rw.Write([]byte{socksVersion5, socksNoAuth}); err != nil {
		return err
	}
	return nil
}

// socksReadRequest reads a SOCKS5 request and returns the CONNECT target as a
// "host:port" string. It rejects any command other than CONNECT.
func socksReadRequest(r io.Reader) (string, error) {
	head := make([]byte, 4) // VER, CMD, RSV, ATYP
	if _, err := io.ReadFull(r, head); err != nil {
		return "", err
	}
	if head[0] != socksVersion5 {
		return "", fmt.Errorf("socks: unsupported version 0x%02x", head[0])
	}
	if head[1] != socksCmdConn {
		return "", fmt.Errorf("socks: unsupported command 0x%02x (only CONNECT)", head[1])
	}
	host, err := socksReadHost(r, head[3])
	if err != nil {
		return "", err
	}
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(r, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)
	return net.JoinHostPort(host, strconv.Itoa(int(port))), nil
}

// socksReadHost reads the address portion for the given ATYP byte.
func socksReadHost(r io.Reader, atyp byte) (string, error) {
	switch atyp {
	case socksATYPv4:
		b := make([]byte, 4)
		if _, err := io.ReadFull(r, b); err != nil {
			return "", err
		}
		return net.IP(b).String(), nil
	case socksATYPv6:
		b := make([]byte, 16)
		if _, err := io.ReadFull(r, b); err != nil {
			return "", err
		}
		return net.IP(b).String(), nil
	case socksATYPDomain:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return "", err
		}
		n := int(lenBuf[0])
		if n == 0 {
			return "", errors.New("socks: empty domain")
		}
		domain := make([]byte, n)
		if _, err := io.ReadFull(r, domain); err != nil {
			return "", err
		}
		return string(domain), nil
	default:
		return "", fmt.Errorf("socks: unsupported address type 0x%02x", atyp)
	}
}

// socksWriteReply writes a minimal reply with the given status code and a
// zeroed IPv4 bind address (clients ignore the bind field for CONNECT).
func socksWriteReply(w io.Writer, rep byte) error {
	// VER, REP, RSV, ATYP=IPv4, BND.ADDR(4)=0, BND.PORT(2)=0
	_, err := w.Write([]byte{socksVersion5, rep, 0x00, socksATYPv4, 0, 0, 0, 0, 0, 0})
	return err
}
