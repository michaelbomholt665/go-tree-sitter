//go:build cgo && (linux || darwin)

package build

/*
#cgo linux LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

typedef const void *(*ts_language_constructor)(void);

static void *open_library(const char *path) {
	return dlopen(path, RTLD_NOW | RTLD_LOCAL);
}

static void *load_symbol(void *handle, const char *name) {
	dlerror();
	return dlsym(handle, name);
}

static const void *call_language_constructor(void *symbol) {
	return ((ts_language_constructor)symbol)();
}

static const char *loader_error(void) {
	return dlerror();
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type nativeLibrary struct {
	handle unsafe.Pointer
}

func openNativeLanguage(path, constructor string) (*nativeLibrary, unsafe.Pointer, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	handle := C.open_library(cPath)
	if handle == nil {
		return nil, nil, fmt.Errorf("load native library %q: %s", path, nativeLoaderError())
	}
	library := &nativeLibrary{handle: handle}
	cConstructor := C.CString(constructor)
	defer C.free(unsafe.Pointer(cConstructor))
	symbol := C.load_symbol(handle, cConstructor)
	if symbol == nil {
		_ = library.Close()
		return nil, nil, fmt.Errorf("resolve constructor %q in %q: %s", constructor, path, nativeLoaderError())
	}
	language := unsafe.Pointer(C.call_language_constructor(symbol))
	if language == nil {
		_ = library.Close()
		return nil, nil, fmt.Errorf("constructor %q in %q returned a null TSLanguage", constructor, path)
	}
	return library, language, nil
}

func nativeLoaderError() string {
	message := C.loader_error()
	if message == nil {
		return "unknown dynamic loader error"
	}
	return C.GoString(message)
}

func (l *nativeLibrary) Close() error {
	if l == nil || l.handle == nil {
		return nil
	}
	if result := C.dlclose(l.handle); result != 0 {
		return fmt.Errorf("close native library: %s", nativeLoaderError())
	}
	l.handle = nil
	return nil
}
