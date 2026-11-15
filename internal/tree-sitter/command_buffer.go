package treesitter

import (
	"runtime"
	"strings"
	"sync"
)

// tailBuffer retains a bounded diagnostic tail while consuming the whole stream.
type tailBuffer struct {
	mu         sync.Mutex
	data       []byte
	limit      int
	overflow   bool
	onOverflow func()
}

func newTailBuffer(limit int) *tailBuffer { return &tailBuffer{limit: limit} }
func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	if len(b.data)+n > b.limit {
		b.overflow = true
		if b.onOverflow != nil {
			b.onOverflow()
		}
	}
	if n >= b.limit {
		b.data = append(b.data[:0], p[n-b.limit:]...)
	} else {
		if excess := len(b.data) + n - b.limit; excess > 0 {
			b.data = append(b.data[:0], b.data[excess:]...)
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}
func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.data)
}
func envKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}
