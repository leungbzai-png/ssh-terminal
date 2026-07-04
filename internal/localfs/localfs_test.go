package localfs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestListFilesAndDirs(t *testing.T) {
	dir := t.TempDir()
	// A file with known content (size) and a subdirectory.
	content := []byte("hello localfs")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), content, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	entries, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	byName := map[string]Entry{}
	for _, e := range entries {
		byName[e.Name] = e
	}
	if len(byName) != 2 {
		t.Fatalf("expected 2 entries, got %d (%v)", len(byName), entries)
	}

	f, ok := byName["a.txt"]
	if !ok {
		t.Fatal("missing a.txt")
	}
	if f.IsDir {
		t.Error("a.txt should not be a dir")
	}
	if f.Size != int64(len(content)) {
		t.Errorf("a.txt size = %d, want %d", f.Size, len(content))
	}
	if f.Path != filepath.Join(dir, "a.txt") {
		t.Errorf("a.txt path = %q, want %q", f.Path, filepath.Join(dir, "a.txt"))
	}
	if f.Mode == "" {
		t.Error("a.txt mode should be non-empty")
	}
	if f.ModTime <= 0 {
		t.Errorf("a.txt modTime should be positive Unix seconds, got %d", f.ModTime)
	}

	d, ok := byName["sub"]
	if !ok {
		t.Fatal("missing sub")
	}
	if !d.IsDir {
		t.Error("sub should be a dir")
	}
	if d.IsLink {
		t.Error("sub should not be a symlink")
	}
}

func TestListError(t *testing.T) {
	_, err := List(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error listing a nonexistent directory")
	}
}

func TestListSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(dir, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		// Symlink creation often needs privileges/developer mode on Windows.
		t.Skipf("symlinks unavailable in this environment: %v", err)
	}
	entries, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.Name == "link.txt" {
			found = true
			if !e.IsLink {
				t.Error("link.txt should have IsLink=true (Lstat, not Stat)")
			}
			if e.IsDir {
				t.Error("a symlink to a file should not report IsDir")
			}
		}
	}
	if !found {
		t.Fatal("symlink entry not listed")
	}
}

func TestHome(t *testing.T) {
	h, err := Home()
	if err != nil {
		t.Fatalf("Home: %v", err)
	}
	if h == "" {
		t.Fatal("Home returned empty path with no error")
	}
	if fi, serr := os.Stat(h); serr != nil || !fi.IsDir() {
		t.Errorf("Home %q is not an existing directory (stat err=%v)", h, serr)
	}
}

func TestRoots(t *testing.T) {
	roots, err := Roots()
	if err != nil {
		t.Fatalf("Roots: %v", err)
	}
	if len(roots) == 0 {
		t.Fatal("Roots returned no roots")
	}
	for _, r := range roots {
		if r == "" {
			t.Error("Roots returned an empty root")
		}
		if _, serr := os.Stat(r); serr != nil {
			t.Errorf("root %q does not exist: %v", r, serr)
		}
	}
	if runtime.GOOS != "windows" {
		if len(roots) != 1 || roots[0] != string(filepath.Separator) {
			t.Errorf("POSIX roots = %v, want [%q]", roots, string(filepath.Separator))
		}
	}
}

func TestParentNested(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	p, isRoot := Parent(child)
	if isRoot {
		t.Fatal("nested child should not be a root")
	}
	if p != filepath.Clean(dir) {
		t.Errorf("Parent(child) = %q, want %q", p, filepath.Clean(dir))
	}
	// Trailing separator on the input must not change the result.
	p2, _ := Parent(child + string(filepath.Separator))
	if p2 != filepath.Clean(dir) {
		t.Errorf("Parent(child/) = %q, want %q", p2, filepath.Clean(dir))
	}
}

// TestParentTerminatesAtRoot walks upward from a temp dir and asserts the
// ascent reaches a root in a bounded number of steps (no infinite loop, which
// is the classic filepath.Dir("C:\\") == "C:\\" trap).
func TestParentTerminatesAtRoot(t *testing.T) {
	cur := t.TempDir()
	const maxDepth = 64
	steps := 0
	for {
		if steps > maxDepth {
			t.Fatalf("Parent did not terminate at a root within %d steps", maxDepth)
		}
		p, isRoot := Parent(cur)
		if isRoot {
			if p != "" {
				t.Errorf("at root, parent should be empty, got %q", p)
			}
			return // reached a root, terminated cleanly
		}
		if p == cur {
			t.Fatalf("Parent looped without reporting root at %q", cur)
		}
		cur = p
		steps++
	}
}

func TestParentRootDirect(t *testing.T) {
	var root string
	if runtime.GOOS == "windows" {
		roots, err := Roots()
		if err != nil || len(roots) == 0 {
			t.Skipf("no drive roots available: %v", err)
		}
		root = roots[0] // e.g. "C:\"
	} else {
		root = "/"
	}
	p, isRoot := Parent(root)
	if !isRoot {
		t.Errorf("Parent(%q) should report isRoot=true", root)
	}
	if p != "" {
		t.Errorf("Parent(%q) parent = %q, want empty", root, p)
	}
}
