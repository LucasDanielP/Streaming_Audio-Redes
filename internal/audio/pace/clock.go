package pace

import (
	"context"
	"time"

	"streaming-audio-redes/internal/audio/pcm"
	"streaming-audio-redes/internal/protocol"
)

// Clock mantém o envio sincronizado com o relógio (evita drift por overhead).
type Clock struct {
	anchor    time.Time
	sentBytes uint64
	meta      protocol.AudioMeta
}

// New cria um relógio de pacing para o formato informado.
func New(meta protocol.AudioMeta) *Clock {
	return &Clock{
		anchor: time.Now(),
		meta:   meta,
	}
}

// SameFormat indica se dois metadados PCM são compatíveis para pacing contínuo.
func SameFormat(a, b protocol.AudioMeta) bool {
	return a.Codec == b.Codec &&
		a.SampleRate == b.SampleRate &&
		a.Channels == b.Channels &&
		a.BitsPerSample == b.BitsPerSample
}

// Meta retorna o formato PCM associado ao relógio.
func (p *Clock) Meta() protocol.AudioMeta {
	return p.meta
}

func (p *Clock) Reset(meta protocol.AudioMeta) {
	p.anchor = time.Now()
	p.sentBytes = 0
	p.meta = meta
}

func (p *Clock) Wait(ctx context.Context, chunkBytes int) error {
	if chunkBytes <= 0 {
		return nil
	}
	p.sentBytes += uint64(chunkBytes)
	bps := pcm.BytesPerSecond(p.meta)
	if bps == 0 {
		return nil
	}

	target := time.Duration(p.sentBytes * uint64(time.Second) / bps)
	elapsed := time.Since(p.anchor)
	if elapsed >= target {
		return nil
	}

	timer := time.NewTimer(target - elapsed)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
