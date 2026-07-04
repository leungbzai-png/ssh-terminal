// Package localfs provides read-only browsing of the local filesystem for the
// SFTP two-pane UI (v1.1.0). It lists directories, resolves the home directory
// and drive/filesystem roots, and computes parent directories with correct
// root detection. It never follows symlinks for traversal, never writes, and
// never persists anything.
package localfs

import (
	"os"
	"path/filepath"
	"runtime"
)

// Entry describes one local filesystem item. Its JSON tags are identical to
// sftpx.FileEntry so the frontend can reuse the single `FileEntry` TypeScript
// type for both panes. ModTime is Unix seconds (matching sftpx.FileEntry).
type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime int64  `json:"modTime"`
	IsDir   bool   `json:"isDir"`
	IsLink  bool   `json:"isLink"`
}

// List returns the contents of a local directory. It uses Lstat so symlinks are
// identified (IsLink) rather than followed, and reports local (native) paths.
// Entries whose metadata cannot be read are skipped rather than aborting the
// whole listing.
func List(dir string) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(des))
	for _, de := range des {
		full := filepath.Join(dir, de.Name())
		fi, lerr := os.Lstat(full)
		if lerr != nil {
			// Unreadable entry (e.g. permissions, race with deletion): skip it
			// instead of failing the entire directory listing.
			continue
		}
		out = append(out, Entry{
			Name:    de.Name(),
			Path:    full,
			Size:    fi.Size(),
			Mode:    fi.Mode().String(),
			ModTime: fi.ModTime().Unix(),
			IsDir:   fi.IsDir(),
			IsLink:  fi.Mode()&os.ModeSymlink != 0,
		})
	}
	return out, nil
}

// Home returns the current user's home directory, or an error if it cannot be
// determined.
func Home() (string, error) {
	return os.UserHomeDir()
}

// Roots returns the filesystem roots the local pane can start from. On Windows
// this is every existing drive root (e.g. "C:\\", "D:\\"), probed A:..Z:. On
// POSIX it is "/". At least one root is returned where the platform provides
// one.
func Roots() ([]string, error) {
	if runtime.GOOS != "windows" {
		return []string{string(filepath.Separator)}, nil
	}
	var roots []string
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		if _, err := os.Stat(root); err == nil {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		// Fall back to the system drive so the pane is never empty.
		if sd := os.Getenv("SystemDrive"); sd != "" {
			roots = append(roots, sd+`\`)
		}
	}
	return roots, nil
}

// Parent returns the parent directory of dir and whether dir is itself a
// filesystem root. When isRoot is true the parent is "" and the caller should
// present the roots list instead (Roots).
//
// filepath.Dir is idempotent at each platform's root ("C:\\" -> "C:\\",
// "/" -> "/"), so comparing it against the (cleaned) input both detects the
// root and avoids an infinite parent loop on Windows drive roots.
func Parent(dir string) (parent string, isRoot bool) {
	cleaned := filepath.Clean(dir)
	p := filepath.Dir(cleaned)
	if p == cleaned {
		return "", true
	}
	return p, false
}
