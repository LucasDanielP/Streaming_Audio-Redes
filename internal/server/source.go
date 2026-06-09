package server

import (
	"context"

	"streaming-audio-redes/internal/protocol"
)

// AudioSource fornece blocos PCM (ou outro codec) para a rádio transmitir.
type AudioSource interface {
	Meta() protocol.AudioMeta
	// Stream envia chunks via emit até ctx ser cancelado ou ocorrer erro.
	// FileSource reinicia o arquivo em loop; LiveSource captura continuamente; Studio mixa música + microfone.
	Stream(ctx context.Context, emit func([]byte) error) error
	Close() error
}
