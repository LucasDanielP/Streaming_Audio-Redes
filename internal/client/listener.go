package client

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"streaming-audio-redes/internal/protocol"
)

// State controla a reprodução/gravação local do fluxo.
type State int

const (
	StatePlaying State = iota
	StatePaused
	StateStopped
)

// Listener conecta ao servidor e grava o fluxo recebido.
type Listener struct {
	addr       string
	outputPath string

	mu    sync.Mutex
	state State
	file  *os.File

	packetsReceived uint64
	bytesWritten    uint64
}

// New cria um cliente ouvinte.
func New(addr, outputPath string) *Listener {
	return &Listener{
		addr:       addr,
		outputPath: outputPath,
		state:      StatePlaying,
	}
}

// Run conecta ao servidor, inicia leitura do fluxo e aceita comandos no stdin.
func (l *Listener) Run() error {
	conn, err := net.Dial("tcp", l.addr)
	if err != nil {
		return fmt.Errorf("conexão com servidor: %w", err)
	}
	defer conn.Close()

	log.Printf("[cliente] conectado a %s", l.addr)

	if err := l.openOutput(); err != nil {
		return err
	}
	defer l.closeOutput()

	errCh := make(chan error, 1)
	go func() {
		errCh <- l.receiveLoop(conn)
	}()

	go l.commandLoop()

	err = <-errCh
	if l.getState() == StateStopped {
		log.Printf("[cliente] encerrado pelo usuário")
		return nil
	}
	return err
}

func (l *Listener) openOutput() error {
	if err := os.MkdirAll(dirOf(l.outputPath), 0o755); err != nil {
		return fmt.Errorf("criar diretório de saída: %w", err)
	}
	f, err := os.Create(l.outputPath)
	if err != nil {
		return fmt.Errorf("criar arquivo de saída: %w", err)
	}
	l.mu.Lock()
	l.file = f
	l.mu.Unlock()
	log.Printf("[cliente] gravando fluxo em %s", l.outputPath)
	return nil
}

func (l *Listener) closeOutput() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}

func (l *Listener) receiveLoop(conn net.Conn) error {
	for {
		if l.getState() == StateStopped {
			return nil
		}

		frame, err := protocol.ReadFrame(conn)
		if err != nil {
			if l.getState() == StateStopped {
				return nil
			}
			return fmt.Errorf("leitura do fluxo: %w", err)
		}

		l.mu.Lock()
		l.packetsReceived++
		pkt := l.packetsReceived
		state := l.state
		f := l.file
		l.mu.Unlock()

		if pkt%50 == 0 || pkt <= 3 {
			log.Printf("[cliente] pacote #%d recebido (%d bytes) | estado: %s", pkt, len(frame), stateLabel(state))
		}

		if state == StatePlaying && f != nil && len(frame) > 0 {
			n, werr := f.Write(frame)
			if werr != nil {
				return fmt.Errorf("gravar áudio: %w", werr)
			}
			l.mu.Lock()
			l.bytesWritten += uint64(n)
			l.mu.Unlock()
		}
	}
}

func (l *Listener) commandLoop() {
	fmt.Println()
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  pause   (p) — pausa a gravação local")
	fmt.Println("  resume  (r) — continua a gravação")
	fmt.Println("  stop    (s) — encerra o cliente")
	fmt.Println()

	var stdin io.Reader = os.Stdin
	buf := make([]byte, 256)
	for {
		fmt.Print("> ")
		n, err := stdin.Read(buf)
		if err != nil {
			return
		}
		cmd := strings.TrimSpace(strings.ToLower(string(buf[:n])))
		switch cmd {
		case "pause", "p":
			l.setState(StatePaused)
			log.Printf("[cliente] pausado — fluxo continua chegando, gravação suspensa")
		case "resume", "r", "continue", "continuar":
			l.setState(StatePlaying)
			log.Printf("[cliente] retomado — gravando novamente")
		case "stop", "s", "parar":
			l.setState(StateStopped)
			log.Printf("[cliente] parando...")
			return
		case "status":
			l.mu.Lock()
			log.Printf("[cliente] estado=%s pacotes=%d bytes_gravados=%d",
				stateLabel(l.state), l.packetsReceived, l.bytesWritten)
			l.mu.Unlock()
		case "":
		default:
			fmt.Printf("comando desconhecido: %q (use pause, resume ou stop)\n", cmd)
		}
	}
}

func (l *Listener) getState() State {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.state
}

func (l *Listener) setState(s State) {
	l.mu.Lock()
	l.state = s
	l.mu.Unlock()
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

func stateLabel(s State) string {
	switch s {
	case StatePlaying:
		return "reproduzindo/gravando"
	case StatePaused:
		return "pausado"
	case StateStopped:
		return "parado"
	default:
		return "desconhecido"
	}
}
