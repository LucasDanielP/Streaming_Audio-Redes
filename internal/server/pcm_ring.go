package server

import "sync"

// pcmRing guarda PCM recente com teto (evita crescimento infinito).
type pcmRing struct {
	mu      sync.Mutex
	data    []byte
	maxSize int
}

func newPCMRing(maxSize int) *pcmRing {
	return &pcmRing{maxSize: maxSize}
}

func (r *pcmRing) Write(p []byte) {
	if len(p) == 0 {
		return
	}
	dup := make([]byte, len(p))
	copy(dup, p)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = append(r.data, dup...)
	if r.maxSize > 0 && len(r.data) > r.maxSize {
		r.data = r.data[len(r.data)-r.maxSize:]
	}
}

func (r *pcmRing) Read(n int) []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]byte, n)
	if len(r.data) >= n {
		copy(out, r.data[:n])
		r.data = r.data[n:]
		return out
	}
	copy(out, r.data)
	r.data = nil
	return out
}

func (r *pcmRing) Clear() {
	r.mu.Lock()
	r.data = nil
	r.mu.Unlock()
}
