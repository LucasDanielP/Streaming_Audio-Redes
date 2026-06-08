package client

import (
	"io"
	"sync"
)

// pcmBuffer é um buffer thread-safe que cresce conforme pacotes chegam.
// O player lê dele em loop contínuo — novos bytes são consumidos automaticamente.
type pcmBuffer struct {
	mu     sync.Mutex
	cond   *sync.Cond
	data   []byte
	closed bool
	paused bool
}

func newPCMBuffer() *pcmBuffer {
	b := &pcmBuffer{}
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
	b.cond.Broadcast()
}

func (b *pcmBuffer) SetPaused(paused bool) {
	b.mu.Lock()
	b.paused = paused
	b.cond.Broadcast()
	b.mu.Unlock()
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
