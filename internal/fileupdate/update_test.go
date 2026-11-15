package fileupdate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectConcurrentEditAndPreservePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	release, err := Lock(path)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := Lock(path); err == nil {
		other()
		t.Fatal("second writer acquired lock")
	}
	release()
	release, err = Lock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := Replace(path, []byte("stale"), []byte("new")); err == nil {
		t.Fatal("accepted stale content")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "original" {
		t.Fatal("changed original on conflict")
	}
	if err := Replace(path, raw, []byte("updated")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v", info.Mode())
	}
}
