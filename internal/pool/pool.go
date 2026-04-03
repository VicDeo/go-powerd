package pool

import "sync"

type Buffer struct {
	data []byte
}

// Data returns a pointer to the data
func (b *Buffer) Data() []byte {
	return b.data
}

// Reset prepares the buffer to reuse
func (b *Buffer) Reset() {
	b.data = b.data[:0]
}

// Bytes returns the buffer with full capacity e.g. for io
func (b *Buffer) Bytes() []byte {
	return b.data[:cap(b.data)]
}

// SetLen trims the buffer to match number of bytes actually read
func (b *Buffer) SetLen(n int) {
	b.data = b.data[:n]
}

var p = sync.Pool{
	New: func() any {
		return &Buffer{data: make([]byte, 4096)}
	},
}

// Get takes a pointer to buffer from the pool
func Get() *Buffer {
	obj := p.Get()
	return obj.(*Buffer)
}

// Put returns the buffer back to the pool
func Put(b *Buffer) {
	b.Reset()
	p.Put(b)
}
