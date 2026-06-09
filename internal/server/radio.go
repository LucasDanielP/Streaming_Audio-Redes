package server

import (
	"context"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"streaming-audio-redes/internal/netx"
	"streaming-audio-redes/internal/protocol"
)

// Radio é o servidor TCP que transmite áudio para todos os clientes conectados.
type Radio struct {
	addr   string
	source AudioSource

	mu      sync.RWMutex
	clients map[net.Conn]struct{}

	packetCount atomic.Uint64
}

// New cria um servidor de rádio com a fonte de áudio informada.
func New(addr string, source AudioSource) *Radio {
	return &Radio{
		addr:    addr,
		source:  source,
		clients: make(map[net.Conn]struct{}),
	}
}

// Run inicia o listener TCP e o loop de transmissão de áudio.
func (r *Radio) Run() error {
	defer func() {
		if r.source != nil {
			_ = r.source.Close()
		}
	}()

	meta := r.source.Meta()
	ln, err := net.Listen("tcp", r.addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("[servidor] rádio iniciada em %s | fonte: %s | formato: %s %dHz %dch %dbit",
		r.addr, meta.Source, meta.Codec, meta.SampleRate, meta.Channels, meta.BitsPerSample)

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
	netx.EnableLowLatency(conn)
	meta := r.source.Meta()
	if err := protocol.WriteMetaFrame(conn, meta); err != nil {
		_ = conn.Close()
		log.Printf("[servidor] falha ao enviar metadados para %s: %v", conn.RemoteAddr(), err)
		return
	}
	log.Printf("[servidor] metadados enviados para %s | codec=%s %dHz",
		conn.RemoteAddr(), meta.Codec, meta.SampleRate)

	r.mu.Lock()
	r.clients[conn] = struct{}{}
	count := len(r.clients)
	r.mu.Unlock()

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
		_, err := conn.Read(buf)
		if err != nil {
			r.removeClient(conn)
			return
		}
	}
}

func (r *Radio) broadcastLoop() {
	for {
		ctx := context.Background()
		err := r.source.Stream(ctx, func(chunk []byte) error {
			r.broadcast(chunk)
			return nil
		})
		if err != nil {
			log.Printf("[servidor] erro na transmissão: %v — tentando novamente em 2s", err)
			time.Sleep(2 * time.Second)
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
	var failed []net.Conn
	for _, conn := range conns {
		if err := protocol.WriteAudioFrame(conn, payload); err != nil {
			failed = append(failed, conn)
		}
	}

	if seq%50 == 0 || seq <= 3 {
		log.Printf("[servidor] pacote #%d enviado (%d bytes) para %d ouvinte(s)", seq, len(payload), len(conns))
	}

	for _, c := range failed {
		r.removeClient(c)
	}
}

// BroadcastMeta envia metadados atualizados a todos os ouvintes conectados.
func (r *Radio) BroadcastMeta(meta protocol.AudioMeta) {
	r.mu.RLock()
	conns := make([]net.Conn, 0, len(r.clients))
	for c := range r.clients {
		conns = append(conns, c)
	}
	r.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	var failed []net.Conn
	for _, conn := range conns {
		if err := protocol.WriteMetaFrame(conn, meta); err != nil {
			failed = append(failed, conn)
		}
	}

	log.Printf("[servidor] metadados atualizados | fonte=%s | %d ouvinte(s)", meta.Source, len(conns))

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
