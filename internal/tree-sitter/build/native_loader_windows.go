//go:build windows

package build

import (
	"fmt"
	"syscall"
	"unsafe"
)

type nativeLibrary struct {
	dll *syscall.DLL
}

func openNativeLanguage(path, constructor string) (*nativeLibrary, unsafe.Pointer, error) {
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return nil, nil, fmt.Errorf("load native library %q: %w", path, err)
	}
	library := &nativeLibrary{dll: dll}
	procedure, err := dll.FindProc(constructor)
	if err != nil {
		_ = library.Close()
		return nil, nil, fmt.Errorf("resolve constructor %q in %q: %w", constructor, path, err)
	}
	pointer, _, callErr := procedure.Call()
	if pointer == 0 {
		_ = library.Close()
		return nil, nil, fmt.Errorf("constructor %q in %q returned a null TSLanguage: %v", constructor, path, callErr)
	}
	return library, unsafe.Pointer(pointer), nil
}

func (l *nativeLibrary) Close() error {
	if l == nil || l.dll == nil {
		return nil
	}
	err := l.dll.Release()
	l.dll = nil
	return err
}
