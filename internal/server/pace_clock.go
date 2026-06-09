package server

import (
	"context"
	"time"

	"streaming-audio-redes/internal/protocol"
)

// paceClock mantém o envio sincronizado com o relógio (evita drift por overhead de rede).
type paceClock struct {
	anchor    time.Time
	sentBytes uint64
	meta      protocol.AudioMeta
}

func newPaceClock(meta protocol.AudioMeta) *paceClock {
	return &paceClock{
		anchor: time.Now(),
		meta:   meta,
	}
}

func samePCMFormat(a, b protocol.AudioMeta) bool {
	return a.Codec == b.Codec &&
		a.SampleRate == b.SampleRate &&
		a.Channels == b.Channels &&
		a.BitsPerSample == b.BitsPerSample
}

func (p *paceClock) Reset(meta protocol.AudioMeta) {
	p.anchor = time.Now()
	p.sentBytes = 0
	p.meta = meta
}

func (p *paceClock) Wait(ctx context.Context, chunkBytes int) error {
	if chunkBytes <= 0 {
		return nil
	}
	p.sentBytes += uint64(chunkBytes)
	bps := PCMBytesPerSecond(p.meta)
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
