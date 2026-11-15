//go:build !windows

// Package filelock provides nonblocking interprocess file locks.
package filelock

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

func Lock(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("acquire lock %q: %w", path, err)
	}
	// Keep the inode in place so other processes lock the same file.
	return func() { _ = f.Close() }, nil
}
