package pcm

import "streaming-audio-redes/internal/protocol"

// AlignedChunkSize retorna ChunkSize alinhado ao frame PCM 16-bit.
func AlignedChunkSize(channels uint16) int {
	frame := int(channels) * 2
	if frame <= 0 {
		frame = 4
	}
	n := protocol.ChunkSize
	return (n / frame) * frame
}
