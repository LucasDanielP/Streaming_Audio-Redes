package broadcast

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
	/* Definindo as variáveis do radio - endereço da rádio, fonte de áudio, mutex para proteger o acesso aos clientes e o contador de pacotes */
	addr   string
	source Source
	mu      sync.RWMutex
	clients map[net.Conn]struct{} // -> map[net.Conn]struct{} é uma map de conexões TCP
	packetCount atomic.Uint64
}

/**
* Cria um novo servidor de rádio com o endereço e fonte de áudio informados.
* @param addr - endereço da rádio
* @param source - fonte de áudio
* @return *Radio - ponteiro para o servidor de rádio
*/
func New(addr string, source Source) *Radio {
	return &Radio{
		addr:    addr,
		source:  source,
		clients: make(map[net.Conn]struct{}),
	}
}

/**
* Inicia o listener TCP e o loop de transmissão de áudio.
* Não possui parâmetros de entrada.
* @return error - erro se o listener TCP falhar
* @return nil se o listener TCP for iniciado com sucesso
*/
func (r *Radio) Run() error {
	/* Definindo o defer para fechar a fonte de áudio se ela não for nil */
	defer func() {
		if r.source != nil {
			_ = r.source.Close()
		}
	}()

	/* Definindo as variáveis do meta - fonte de áudio, listener TCP e erro */
	meta := r.source.Meta()
	/* Definindo o listener TCP - aqui é criado o listener TCP que será usado para aceitar as conexões dos clientes */
	/* Retorna erro se o listener TCP falhar */
	ln, err := net.Listen("tcp", r.addr)
	/* Verificacão de erro - se o listener TCP falhar, retorna o erro */
	if err != nil {
		return err
	}
	/* Definindo o defer para fechar o listener TCP */
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
