//go:build !windows

package commandexec

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCancellationTerminatesDescendants(t *testing.T) {
	dir := t.TempDir()
	ready, marker := filepath.Join(dir, "ready"), filepath.Join(dir, "orphan")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd, err := New(ctx, "sh", "-c", "(sleep 1; touch \"$2\") & echo ready > \"$1\"; wait", "sh", ready, marker)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- Run(cmd) }()
	deadline := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
waiting:
	for {
		select {
		case err := <-done:
			t.Fatalf("command exited before cancellation: %v", err)
		case <-deadline:
			t.Fatal("child never became ready")
		case <-ticker.C:
			if _, err := os.Stat(ready); err == nil {
				break waiting
			}
		}
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancellation")
		}
	case <-time.After(4 * time.Second):
		t.Fatal("cancelled command hung")
	}
	time.Sleep(1200 * time.Millisecond)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("descendant survived cancellation")
	}
}
