// Package portable resolves filesystem paths relative to the executable,
// so the whole app is movable as a folder without losing its data.
package portable

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	once    sync.Once
	baseDir string
	dataDir string
)

func resolve() {
	exe, err := os.Executable()
	if err != nil {
		// Fallback to cwd; should not happen on a healthy system.
		baseDir, _ = os.Getwd()
	} else {
		// Resolve symlinks so the layout follows the real binary location.
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			baseDir = filepath.Dir(real)
		} else {
			baseDir = filepath.Dir(exe)
		}
	}
	dataDir = filepath.Join(baseDir, "data")
	_ = os.MkdirAll(dataDir, 0o755)
}

// BaseDir returns the directory containing the executable.
func BaseDir() string {
	once.Do(resolve)
	return baseDir
}

// DataDir returns <exe-dir>/data, guaranteed to exist.
func DataDir() string {
	once.Do(resolve)
	return dataDir
}

// DataPath joins parts under DataDir.
func DataPath(parts ...string) string {
	return filepath.Join(append([]string{DataDir()}, parts...)...)
}
