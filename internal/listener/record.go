package listener

import (
	"os"

	"streaming-audio-redes/internal/audio/wav"
	"streaming-audio-redes/internal/protocol"
)

func buildWAVHeader(meta protocol.AudioMeta) []byte {
	return wav.BuildPCMHeader(meta)
}

func updateWAVSizes(f *os.File, pcmBytes uint32) error {
	return wav.UpdatePCMSizes(f, pcmBytes)
}
