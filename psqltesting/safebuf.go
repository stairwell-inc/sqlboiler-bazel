package psqltesting

import (
	"bytes"
	"sync"
)

// buffer wraps a bytes.Buffer and uses a mutex
// to serialize access to its Write and String
// methods.
type buffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write implements io.Writer.
func (b *buffer) Write(buf []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(buf)
}

// String implements fmt.Stringer.
func (b *buffer) String() (str string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
