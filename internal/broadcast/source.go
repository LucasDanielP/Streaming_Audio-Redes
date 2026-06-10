package broadcast

import (
	"context"

	"streaming-audio-redes/internal/protocol"
)

// Source fornece blocos PCM (ou outro codec) para a rádio transmitir.
type Source interface {
	Meta() protocol.AudioMeta
	// Stream envia chunks via emit até ctx ser cancelado ou ocorrer erro.
	Stream(ctx context.Context, emit func([]byte) error) error
	Close() error
}
