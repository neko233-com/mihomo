package constant

import (
	"path/filepath"
	"testing"
)

func TestCacheFileOverride(t *testing.T) {
	oldHome, oldCache := Path.homeDir, Path.cacheFile
	defer func() { Path.homeDir, Path.cacheFile = oldHome, oldCache }()
	SetHomeDir(t.TempDir())
	SetCacheFile("")
	if got, want := Path.Cache(), filepath.ToSlash(filepath.Join(Path.homeDir, "cache.db")); filepath.ToSlash(got) != want {
		t.Fatalf("default %q want %q", got, want)
	}
	alternative := filepath.Join(Path.homeDir, "worker-cache.db")
	SetCacheFile(alternative)
	if got := Path.Cache(); got != alternative {
		t.Fatalf("override %q", got)
	}
	SetCacheFile("")
	if got := Path.Cache(); got == alternative {
		t.Fatal("override did not clear")
	}
}
