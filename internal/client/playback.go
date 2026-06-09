package client

import (
	"fmt"
	"log"
	"time"

	"github.com/ebitengine/oto/v3"
	"streaming-audio-redes/internal/protocol"
)

const prebufferMs = 150

type streamPlayer struct {
	meta   protocol.AudioMeta
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

	return buildStreamPlayer(meta)
}

func buildStreamPlayer(meta protocol.AudioMeta) (*streamPlayer, error) {
	maxBytes := pcmMaxBufferBytes(meta)
	buf := newPCMBuffer(maxBytes)

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
	bps := int(meta.SampleRate) * int(meta.Channels) * 2
	hwBuf := bps / 10 // ~100 ms no driver — reduz underruns
	if hwBuf < 4096 {
		hwBuf = 4096
	}
	player.SetBufferSize(hwBuf)

	sp := &streamPlayer{meta: meta, buf: buf, player: player}
	log.Printf("[cliente] reprodução | %dHz %dch | buffer %dms | pré-buffer %dms",
		meta.SampleRate, meta.Channels, defaultMaxBufferMs, prebufferMs)
	return sp, nil
}

func (sp *streamPlayer) startAfterPrebuffer() {
	minBytes := pcmBytesForMs(sp.meta, prebufferMs)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sp.buf.Len() >= minBytes {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	sp.player.Play()
}

func pcmBytesForMs(meta protocol.AudioMeta, ms int) int {
	if meta.SampleRate == 0 || meta.Channels == 0 || meta.BitsPerSample == 0 {
		return 0
	}
	bps := int(meta.SampleRate) * int(meta.Channels) * int(meta.BitsPerSample) / 8
	return bps * ms / 1000
}

func pcmMaxBufferBytes(meta protocol.AudioMeta) int {
	return pcmBytesForMs(meta, defaultMaxBufferMs)
}

func (sp *streamPlayer) Write(pcm []byte) {
	if len(pcm) == 0 {
		return
	}
	sp.buf.Write(pcm)
}

func (sp *streamPlayer) SetPaused(paused bool) {
	sp.buf.SetPaused(paused)
	if sp.player == nil {
		return
	}
	if paused {
		sp.player.Pause()
		return
	}
	sp.player.Reset()
	sp.player.Play()
}

func (sp *streamPlayer) ApplyMeta(meta protocol.AudioMeta) error {
	if !sameAudioFormat(sp.meta, meta) {
		old := *sp
		next, err := buildStreamPlayer(meta)
		if err != nil {
			return err
		}
		*sp = *next
		old.Close()
		go sp.startAfterPrebuffer()
		log.Printf("[cliente] formato alterado: %dHz %dch | %s", meta.SampleRate, meta.Channels, meta.Source)
		return nil
	}
	sp.meta = meta
	log.Printf("[cliente] fonte no ar: %s", meta.Source)
	return nil
}

func (sp *streamPlayer) BufferedBytes() int {
	return sp.buf.Len()
}

func (sp *streamPlayer) Close() {
	if sp.buf != nil {
		sp.buf.Close()
	}
	if sp.player != nil {
		_ = sp.player.Close()
	}
}

func sameAudioFormat(a, b protocol.AudioMeta) bool {
	return a.Codec == b.Codec &&
		a.SampleRate == b.SampleRate &&
		a.Channels == b.Channels &&
		a.BitsPerSample == b.BitsPerSample
}
