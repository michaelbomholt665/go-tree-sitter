//go:build (!cgo && !windows) || (!linux && !darwin && !windows)

package build

import (
	"errors"
	"unsafe"
)

type nativeLibrary struct{}

func openNativeLanguage(_, _ string) (*nativeLibrary, unsafe.Pointer, error) {
	return nil, nil, errors.New("native grammar validation requires cgo on this platform")
}

func (*nativeLibrary) Close() error { return nil }
