package hosts

import (
	"encoding/json"
	"fmt"
	"time"
)

// Safe host export/import.
//
// Security: the export format is a strict whitelist. It carries only non-secret
// host metadata. It NEVER contains a password, passphrase, encPassword,
// encPassphrase, or any private-key material. External key references are kept
// as plain filesystem paths (KeyPath) — the key file itself is never read,
// copied, or embedded.

// ExportFormat identifies the safe host-export document.
const ExportFormat = "ssh-terminal.hosts.safe-export"

// ExportVersion is the current export schema version.
const ExportVersion = 1

// SafeHost is the whitelist of host fields allowed to leave the app in a plain
// (unencrypted) export. Deliberately built as its own struct — not Host /
// storedHost — so no secret field can ever be added by accident.
type SafeHost struct {
	Name         string `json:"name"`
	Address      string `json:"address"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	AuthType     string `json:"authType"`
	KeyPath      string `json:"keyPath,omitempty"`
	ManagedKeyID string `json:"managedKeyId,omitempty"`
	Group        string `json:"group,omitempty"`
	Note         string `json:"note,omitempty"`
}

// Export is the top-level safe-export document.
type Export struct {
	Format     string     `json:"format"`
	Version    int        `json:"version"`
	ExportedAt string     `json:"exportedAt"`
	Hosts      []SafeHost `json:"hosts"`
}

// BuildExport produces a safe export document from all saved hosts. It sources
// hosts from List(), which already strips plaintext secrets, and then copies
// only whitelisted fields into SafeHost — so the result cannot contain secrets.
func BuildExport() (Export, error) {
	list, err := List()
	if err != nil {
		return Export{}, err
	}
	out := Export{
		Format:     ExportFormat,
		Version:    ExportVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Hosts:      make([]SafeHost, 0, len(list)),
	}
	for _, h := range list {
		out.Hosts = append(out.Hosts, SafeHost{
			Name:         h.Name,
			Address:      h.Address,
			Port:         h.Port,
			User:         h.User,
			AuthType:     h.AuthType,
			KeyPath:      h.KeyPath,
			ManagedKeyID: h.ManagedKeyID,
			Group:        h.Group,
			Note:         h.Note,
		})
	}
	return out, nil
}

// MarshalExport returns the pretty-printed JSON bytes of a safe export.
func MarshalExport() ([]byte, error) {
	exp, err := BuildExport()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(exp, "", "  ")
}

// ParseExport validates and decodes a safe-export document. It rejects unknown
// formats and unsupported versions with a friendly error, so an accidental
// wrong-file import fails cleanly instead of corrupting the host list.
func ParseExport(data []byte) (Export, error) {
	var exp Export
	if err := json.Unmarshal(data, &exp); err != nil {
		return Export{}, fmt.Errorf("无法解析导入文件（不是有效的 JSON）")
	}
	if exp.Format != ExportFormat {
		return Export{}, fmt.Errorf("文件格式无法识别（缺少 %q 标记）", ExportFormat)
	}
	if exp.Version <= 0 || exp.Version > ExportVersion {
		return Export{}, fmt.Errorf("不支持的导出版本: %d", exp.Version)
	}
	return exp, nil
}
