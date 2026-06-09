package client

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"streaming-audio-redes/internal/netx"
	"streaming-audio-redes/internal/protocol"
)

// State controla a reprodução/gravação local do fluxo.
type State int

const (
	StatePlaying State = iota
	StatePaused
	StateStopped
)

// Listener conecta ao servidor, grava e/ou reproduz o fluxo recebido.
type Listener struct {
	addr       string
	outputPath string
	playLive   bool
	saveToFile bool

	mu     sync.Mutex
	state  State
	file   *os.File
	meta   protocol.AudioMeta
	player *streamPlayer

	packetsReceived uint64
	bytesWritten    uint64
	pcmDataSize     uint32
	hasWAVHeader    bool
}

// New cria um cliente ouvinte.
func New(addr, outputPath string, playLive bool) *Listener {
	save := outputPath != "" && outputPath != "-"
	return &Listener{
		addr:       addr,
		outputPath: outputPath,
		playLive:   playLive,
		saveToFile: save,
		state:      StatePlaying,
	}
}

// Run conecta ao servidor com comandos no terminal (modo CLI).
func (l *Listener) Run() error {
	conn, err := net.Dial("tcp", l.addr)
	if err != nil {
		return fmt.Errorf("conexão com servidor: %w", err)
	}
	return l.serve(conn, true)
}

// RunHeadless conecta e recebe o fluxo sem prompt de terminal (para GUI).
func (l *Listener) RunHeadless() error {
	conn, err := net.Dial("tcp", l.addr)
	if err != nil {
		return fmt.Errorf("conexão com servidor: %w", err)
	}
	return l.serve(conn, false)
}

func (l *Listener) serve(conn net.Conn, interactive bool) error {
	defer conn.Close()
	netx.EnableLowLatency(conn)

	log.Printf("[cliente] conectado a %s", l.addr)

	meta, err := protocol.ReadMetaFrame(conn)
	if err != nil {
		return fmt.Errorf("receber metadados: %w", err)
	}
	l.meta = meta
	log.Printf("[cliente] metadados recebidos | codec=%s container=%s %dHz %dch %dbit | fonte=%s",
		meta.Codec, meta.Container, meta.SampleRate, meta.Channels, meta.BitsPerSample, meta.Source)

	if l.playLive {
		player, err := newStreamPlayer(meta)
		if err != nil {
			return err
		}
		l.player = player
		defer l.player.Close()
		go l.player.startAfterPrebuffer()
	}

	if l.saveToFile {
		if err := l.prepareOutput(); err != nil {
			return err
		}
		if err := l.startRecording(); err != nil {
			return err
		}
		if err := l.initOutputFromMeta(); err != nil {
			return err
		}
		defer l.closeOutput()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- l.receiveLoop(conn)
	}()

	if interactive {
		go l.commandLoop()
	}

	err = <-errCh
	if l.getState() == StateStopped {
		log.Printf("[cliente] encerrado pelo usuário")
		return nil
	}
	return err
}

func (l *Listener) prepareOutput() error {
	if err := os.MkdirAll(dirOf(l.outputPath), 0o755); err != nil {
		return fmt.Errorf("criar diretório de saída: %w", err)
	}
	log.Printf("[cliente] cada pacote será gravado imediatamente em %s", l.outputPath)
	return nil
}

func (l *Listener) initOutputFromMeta() error {
	if l.meta.Codec != "pcm" {
		log.Printf("[cliente] codec %q — gravando fluxo bruto (sem cabeçalho WAV local)", l.meta.Codec)
		return nil
	}

	hdr := buildWAVHeader(l.meta)
	l.mu.Lock()
	f := l.file
	l.mu.Unlock()
	if f == nil {
		return fmt.Errorf("arquivo de saída não aberto")
	}
	if _, err := f.Write(hdr); err != nil {
		return fmt.Errorf("gravar cabeçalho WAV: %w", err)
	}
	l.mu.Lock()
	l.hasWAVHeader = true
	l.bytesWritten = uint64(len(hdr))
	l.mu.Unlock()
	log.Printf("[cliente] cabeçalho WAV criado a partir dos metadados")
	return f.Sync()
}

func (l *Listener) startRecording() error {
	f, err := os.Create(l.outputPath)
	if err != nil {
		return fmt.Errorf("criar arquivo de saída: %w", err)
	}
	l.mu.Lock()
	l.file = f
	l.bytesWritten = 0
	l.pcmDataSize = 0
	l.hasWAVHeader = false
	l.mu.Unlock()
	log.Printf("[cliente] gravação iniciada em %s", l.outputPath)
	return nil
}

func (l *Listener) writePCM(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if l.player != nil {
		l.player.Write(data)
	}

	if !l.saveToFile {
		return nil
	}

	l.mu.Lock()
	f := l.file
	l.mu.Unlock()
	if f == nil {
		return nil
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("gravar áudio: %w", err)
	}

	l.mu.Lock()
	l.bytesWritten += uint64(len(data))
	if l.hasWAVHeader {
		l.pcmDataSize += uint32(len(data))
	}
	pcm := l.pcmDataSize
	hasHeader := l.hasWAVHeader
	f = l.file
	l.mu.Unlock()

	if hasHeader {
		return updateWAVSizes(f, pcm)
	}
	return f.Sync()
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

		frameType, payload, err := protocol.ReadTypedFrame(conn)
		if err != nil {
			if l.getState() == StateStopped {
				return nil
			}
			return fmt.Errorf("leitura do fluxo: %w", err)
		}

		switch frameType {
		case protocol.FrameTypeMeta:
			if err := l.applyMeta(payload); err != nil {
				log.Printf("[cliente] aviso ao aplicar metadados: %v", err)
			}
			continue
		case protocol.FrameTypeAudio:
			// segue abaixo
		default:
			continue
		}

		frame := payload
		l.mu.Lock()
		l.packetsReceived++
		pkt := l.packetsReceived
		state := l.state
		l.mu.Unlock()

		if pkt%50 == 0 || pkt <= 3 {
			extra := ""
			if l.player != nil {
				extra = fmt.Sprintf(" | buffer_play: %d bytes", l.player.BufferedBytes())
			}
			log.Printf("[cliente] pacote #%d recebido (%d bytes) | estado: %s%s",
				pkt, len(frame), stateLabel(state), extra)
		}

		if state != StatePlaying || len(frame) == 0 {
			continue
		}

		if err := l.writePCM(frame); err != nil {
			return err
		}
	}
}

func (l *Listener) applyMeta(payload []byte) error {
	meta, err := protocol.ParseMeta(payload)
	if err != nil {
		return err
	}

	l.mu.Lock()
	l.meta = meta
	playLive := l.playLive
	player := l.player
	l.mu.Unlock()

	log.Printf("[cliente] metadados atualizados | fonte=%s | %dHz %dch %dbit",
		meta.Source, meta.SampleRate, meta.Channels, meta.BitsPerSample)

	if !playLive {
		return nil
	}

	if player == nil {
		p, err := newStreamPlayer(meta)
		if err != nil {
			return err
		}
		l.mu.Lock()
		l.player = p
		l.mu.Unlock()
		go p.startAfterPrebuffer()
		return nil
	}

	if err := player.ApplyMeta(meta); err != nil {
		return err
	}
	return nil
}

func (l *Listener) commandLoop() {
	fmt.Println()
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  pause   (p) — pausa gravação e reprodução local")
	fmt.Println("  resume  (r) — continua gravação e reprodução")
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
			if l.player != nil {
				l.player.SetPaused(true)
			}
			log.Printf("[cliente] pausado — fluxo continua chegando, gravação/reprodução suspensas")
		case "resume", "r", "continue", "continuar":
			l.setState(StatePlaying)
			if l.player != nil {
				l.player.SetPaused(false)
			}
			log.Printf("[cliente] retomado")
		case "stop", "s", "parar":
			l.setState(StateStopped)
			log.Printf("[cliente] parando...")
			return
		case "status":
			l.mu.Lock()
			var durationSec float64
			if l.meta.SampleRate > 0 && l.meta.Channels > 0 && l.meta.BitsPerSample > 0 {
				durationSec = float64(l.pcmDataSize) / float64(l.meta.SampleRate*uint32(l.meta.Channels)*uint32(l.meta.BitsPerSample)/8)
			}
			l.mu.Unlock()
			msg := fmt.Sprintf("[cliente] estado=%s pacotes=%d", stateLabel(l.getState()), l.packetsReceived)
			if l.saveToFile {
				msg += fmt.Sprintf(" bytes_gravados=%d (~%.1f s)", l.pcmDataSize, durationSec)
			}
			if l.player != nil {
				msg += fmt.Sprintf(" buffer_play=%d bytes", l.player.BufferedBytes())
			}
			log.Print(msg)
		case "":
		default:
			fmt.Printf("comando desconhecido: %q (use pause, resume ou stop)\n", cmd)
		}
	}
}

// Stats resume o estado atual do ouvinte (útil para GUI).
type Stats struct {
	State       State
	StateLabel  string
	Packets     uint64
	PCMBytes    uint32
	DurationSec float64
	BufferBytes int
	LatencyMs   float64
	Source      string
}

func (l *Listener) Stats() Stats {
	l.mu.Lock()
	defer l.mu.Unlock()

	var durationSec float64
	if l.meta.SampleRate > 0 && l.meta.Channels > 0 && l.meta.BitsPerSample > 0 {
		durationSec = float64(l.pcmDataSize) / float64(l.meta.SampleRate*uint32(l.meta.Channels)*uint32(l.meta.BitsPerSample)/8)
	}
	buf := 0
	var latencyMs float64
	if l.player != nil {
		buf = l.player.BufferedBytes()
		if l.meta.SampleRate > 0 && l.meta.Channels > 0 && l.meta.BitsPerSample > 0 {
			bps := float64(l.meta.SampleRate) * float64(l.meta.Channels) * float64(l.meta.BitsPerSample) / 8
			latencyMs = float64(buf) / bps * 1000
		}
	}
	return Stats{
		State:       l.state,
		StateLabel:  stateLabel(l.state),
		Packets:     l.packetsReceived,
		PCMBytes:    l.pcmDataSize,
		DurationSec: durationSec,
		BufferBytes: buf,
		LatencyMs:   latencyMs,
		Source:      l.meta.Source,
	}
}

// Pause suspende gravação e reprodução local.
func (l *Listener) Pause() {
	l.setState(StatePaused)
	if l.player != nil {
		l.player.SetPaused(true)
	}
}

// Resume retoma gravação e reprodução local.
func (l *Listener) Resume() {
	l.setState(StatePlaying)
	if l.player != nil {
		l.player.SetPaused(false)
	}
}

// Stop encerra a sessão do ouvinte.
func (l *Listener) Stop() {
	l.setState(StateStopped)
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
