package server

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"streaming-audio-redes/internal/protocol"
)

// Radio é o servidor TCP que transmite áudio em loop para todos os clientes.
type Radio struct {
	addr      string
	audioPath string
	chunkWait time.Duration

	mu      sync.RWMutex
	clients map[net.Conn]struct{}

	listenerCount atomic.Int64
	packetCount   atomic.Uint64
}

// New cria um servidor de rádio configurado.
func New(addr, audioPath string, chunkWait time.Duration) *Radio {
	return &Radio{
		addr:      addr,
		audioPath: audioPath,
		chunkWait: chunkWait,
		clients:   make(map[net.Conn]struct{}),
	}
}

// Run inicia o listener TCP e o loop de transmissão de áudio.
func (r *Radio) Run() error {
	ln, err := net.Listen("tcp", r.addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("[servidor] rádio iniciada em %s | arquivo: %s", r.addr, r.audioPath)

	go r.broadcastLoop()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[servidor] erro ao aceitar conexão: %v", err)
			continue
		}
		r.addClient(conn)
	}
}

func (r *Radio) addClient(conn net.Conn) {
	r.mu.Lock()
	r.clients[conn] = struct{}{}
	count := len(r.clients)
	r.mu.Unlock()

	r.listenerCount.Add(1)
	log.Printf("[servidor] cliente conectado: %s | ouvintes ativos: %d", conn.RemoteAddr(), count)

	go r.watchClient(conn)
}

func (r *Radio) removeClient(conn net.Conn) {
	r.mu.Lock()
	if _, ok := r.clients[conn]; !ok {
		r.mu.Unlock()
		return
	}
	delete(r.clients, conn)
	count := len(r.clients)
	r.mu.Unlock()

	_ = conn.Close()
	log.Printf("[servidor] cliente desconectado: %s | ouvintes ativos: %d", conn.RemoteAddr(), count)
}

func (r *Radio) watchClient(conn net.Conn) {
	buf := make([]byte, 1)
	for {
		// Descarta qualquer dado enviado pelo cliente; detecta desconexão por EOF.
		_, err := conn.Read(buf)
		if err != nil {
			r.removeClient(conn)
			return
		}
	}
}

func (r *Radio) broadcastLoop() {
	for {
		if err := r.streamFileOnce(); err != nil {
			log.Printf("[servidor] erro na transmissão: %v — tentando novamente em 2s", err)
			time.Sleep(2 * time.Second)
		}
	}
}

func (r *Radio) streamFileOnce() error {
	f, err := os.Open(r.audioPath)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, protocol.ChunkSize)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			r.broadcast(chunk)
			if r.chunkWait > 0 {
				time.Sleep(r.chunkWait)
			}
		}
		if err == io.EOF {
			r.mu.RLock()
			n := len(r.clients)
			r.mu.RUnlock()
			if n > 0 {
				log.Printf("[servidor] fim do arquivo — reiniciando loop (%d ouvinte(s))", n)
			}
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (r *Radio) broadcast(payload []byte) {
	r.mu.RLock()
	conns := make([]net.Conn, 0, len(r.clients))
	for c := range r.clients {
		conns = append(conns, c)
	}
	r.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	seq := r.packetCount.Add(1)
	var wg sync.WaitGroup
	var failedMu sync.Mutex
	var failed []net.Conn

	for _, conn := range conns {
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			if err := protocol.WriteFrame(c, payload); err != nil {
				failedMu.Lock()
				failed = append(failed, c)
				failedMu.Unlock()
			}
		}(conn)
	}
	wg.Wait()

	if seq%50 == 0 || seq <= 3 {
		log.Printf("[servidor] pacote #%d enviado (%d bytes) para %d ouvinte(s)", seq, len(payload), len(conns))
	}

	for _, c := range failed {
		r.removeClient(c)
	}
}

// Stats retorna métricas simples da rádio.
func (r *Radio) Stats() (listeners int, packets uint64) {
	r.mu.RLock()
	listeners = len(r.clients)
	r.mu.RUnlock()
	return listeners, r.packetCount.Load()
}
