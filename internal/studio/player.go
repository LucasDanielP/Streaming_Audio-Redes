package studio

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"streaming-audio-redes/internal/audio/pcm"
	"streaming-audio-redes/internal/audio/wav"
	"streaming-audio-redes/internal/protocol"
)

// MusicProgress descreve a posição de reprodução da faixa atual.
type MusicProgress struct {
	PositionSec float64
	DurationSec float64
	Playing     bool
	Track       string
}

// MusicPlayer lê uma faixa WAV com seek e posição mantida entre leituras.
type MusicPlayer struct {
	mu sync.Mutex

	path      string
	file      *os.File
	meta      protocol.AudioMeta
	pcmOffset int64
	pcmSize   uint32
	position  int64
	eof       bool

	pendingSeekSec float64 // >= 0: seek absoluto em segundos; < 0: nenhum
	hasPendingSeek bool
}

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{pendingSeekSec: -1}
}

func (p *MusicPlayer) Open(path string) error {
	meta, offset, pcmSize, err := wav.DetectMeta(path)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.file != nil {
		_ = p.file.Close()
		p.file = nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		_ = f.Close()
		return fmt.Errorf("posicionar WAV: %w", err)
	}

	p.path = path
	p.file = f
	p.meta = meta
	p.pcmOffset = offset
	p.pcmSize = pcmSize
	p.position = 0
	p.eof = false
	p.pendingSeekSec = -1
	p.hasPendingSeek = false
	return nil
}

func (p *MusicPlayer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.file != nil {
		_ = p.file.Close()
		p.file = nil
	}
	p.path = ""
	p.eof = false
	p.position = 0
	p.pcmSize = 0
}

func (p *MusicPlayer) IsOpen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.file != nil && !p.eof
}

func (p *MusicPlayer) CurrentPath() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.path
}

func (p *MusicPlayer) Meta() protocol.AudioMeta {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.meta
}

func (p *MusicPlayer) Progress() MusicProgress {
	p.mu.Lock()
	defer p.mu.Unlock()

	bps := pcm.BytesPerSecond(p.meta)
	var posSec, durSec float64
	if bps > 0 {
		posSec = float64(p.position) / float64(bps)
		durSec = float64(p.pcmSize) / float64(bps)
	}

	return MusicProgress{
		PositionSec: posSec,
		DurationSec: durSec,
		Playing:     p.file != nil && !p.eof,
		Track:       filepath.Base(p.path),
	}
}

// SeekTo posiciona a faixa em um instante absoluto (segundos).
func (p *MusicPlayer) SeekTo(seconds float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.file == nil {
		return
	}
	if seconds < 0 {
		seconds = 0
	}
	p.pendingSeekSec = seconds
	p.hasPendingSeek = true
	p.eof = false
}

// SeekForward avança a faixa em segundos relativos à posição atual.
func (p *MusicPlayer) SeekForward(seconds float64) {
	if seconds <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.file == nil {
		return
	}
	bps := pcm.BytesPerSecond(p.meta)
	if bps == 0 {
		return
	}
	cur := float64(p.position) / float64(bps)
	p.pendingSeekSec = cur + seconds
	p.hasPendingSeek = true
	p.eof = false
}

func (p *MusicPlayer) Read(n int) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]byte, n)
	if p.file == nil {
		return out
	}

	p.applyPendingSeekLocked()
	if p.eof {
		return out
	}

	total := 0
	for total < n {
		read, err := p.file.Read(out[total:])
		if read > 0 {
			total += read
			p.position += int64(read)
		}
		if err == io.EOF {
			p.eof = true
			break
		}
		if err != nil {
			p.eof = true
			log.Printf("[música] leitura em %s: %v", filepath.Base(p.path), err)
			break
		}
	}
	if p.position >= int64(p.pcmSize) {
		p.eof = true
	}
	return out
}

func (p *MusicPlayer) applyPendingSeekLocked() {
	if !p.hasPendingSeek || p.file == nil {
		return
	}

	bps := pcm.BytesPerSecond(p.meta)
	if bps == 0 {
		p.hasPendingSeek = false
		p.pendingSeekSec = -1
		return
	}

	sec := p.pendingSeekSec
	maxSec := float64(p.pcmSize) / float64(bps)
	if sec > maxSec {
		sec = maxSec
	}
	if sec < 0 {
		sec = 0
	}

	offset := int64(float64(bps) * sec)
	if _, err := p.file.Seek(p.pcmOffset+offset, io.SeekStart); err != nil {
		log.Printf("[música] seek em %s: %v", filepath.Base(p.path), err)
	} else {
		p.position = offset
		p.eof = offset >= int64(p.pcmSize)
	}

	p.hasPendingSeek = false
	p.pendingSeekSec = -1
}
