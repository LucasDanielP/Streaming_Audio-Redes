package server

import "streaming-audio-redes/internal/protocol"

// PCMBytesPerSecond calcula a taxa de bytes do fluxo PCM.
func PCMBytesPerSecond(meta protocol.AudioMeta) uint64 {
	if meta.SampleRate == 0 || meta.Channels == 0 || meta.BitsPerSample == 0 {
		return 0
	}
	return uint64(meta.SampleRate) * uint64(meta.Channels) * uint64(meta.BitsPerSample) / 8
}
