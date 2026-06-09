package client

import (
	"io"
	"sync"
)

// Buffer de jitter: evita underruns (áudio travado) sem acumular segundos de atraso.
const defaultMaxBufferMs = 600

type pcmBuffer struct {
	mu       sync.Mutex
	cond     *sync.Cond
	data     []byte
	maxBytes int
	closed   bool
	paused   bool
}

func newPCMBuffer(maxBytes int) *pcmBuffer {
	b := &pcmBuffer{maxBytes: maxBytes}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *pcmBuffer) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.paused {
		return
	}
	b.data = append(b.data, p...)
	b.trimLocked()
	b.cond.Broadcast()
}

func (b *pcmBuffer) trimLocked() {
	if b.maxBytes <= 0 || len(b.data) <= b.maxBytes {
		return
	}
	drop := len(b.data) - b.maxBytes
	b.data = b.data[drop:]
}

func (b *pcmBuffer) Clear() {
	b.mu.Lock()
	b.data = nil
	b.mu.Unlock()
}

func (b *pcmBuffer) SetPaused(paused bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.paused && !paused {
		b.data = nil
	}
	b.paused = paused
	b.cond.Broadcast()
}

func (b *pcmBuffer) Close() {
	b.mu.Lock()
	b.closed = true
	b.cond.Broadcast()
	b.mu.Unlock()
}

func (b *pcmBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.data)
}

func (b *pcmBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for {
		if len(b.data) > 0 && !b.paused {
			n := copy(p, b.data)
			b.data = b.data[n:]
			return n, nil
		}
		if b.closed {
			return 0, io.EOF
		}
		b.cond.Wait()
	}
}
