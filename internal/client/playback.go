package client

import (
	"fmt"
	"log"

	"github.com/ebitengine/oto/v3"
	"streaming-audio-redes/internal/protocol"
)

// streamPlayer reproduz PCM em tempo real a partir de um buffer que cresce.
type streamPlayer struct {
	buf    *pcmBuffer
	player *oto.Player
}

func newStreamPlayer(meta protocol.AudioMeta) (*streamPlayer, error) {
	if meta.Codec != "pcm" {
		return nil, fmt.Errorf("reprodução ao vivo só suporta PCM (recebido: %s)", meta.Codec)
	}
	if meta.BitsPerSample != 16 {
		return nil, fmt.Errorf("reprodução ao vivo requer 16-bit (recebido: %d)", meta.BitsPerSample)
	}
	if meta.SampleRate == 0 || meta.Channels == 0 {
		return nil, fmt.Errorf("metadados de áudio inválidos")
	}

	buf := newPCMBuffer()
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   int(meta.SampleRate),
		ChannelCount: int(meta.Channels),
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		return nil, fmt.Errorf("inicializar áudio: %w", err)
	}
	<-ready

	player := ctx.NewPlayer(buf)
	player.Play()

	log.Printf("[cliente] reprodução ao vivo iniciada | %dHz %dch — buffer cresce conforme pacotes chegam",
		meta.SampleRate, meta.Channels)

	return &streamPlayer{buf: buf, player: player}, nil
}

func (sp *streamPlayer) Write(pcm []byte) {
	sp.buf.Write(pcm)
}

func (sp *streamPlayer) SetPaused(paused bool) {
	sp.buf.SetPaused(paused)
	if sp.player == nil {
		return
	}
	if paused {
		sp.player.Pause()
	} else {
		sp.player.Play()
	}
}

func (sp *streamPlayer) BufferedBytes() int {
	return sp.buf.Len()
}

func (sp *streamPlayer) Close() {
	sp.buf.Close()
	if sp.player != nil {
		_ = sp.player.Close()
	}
}
