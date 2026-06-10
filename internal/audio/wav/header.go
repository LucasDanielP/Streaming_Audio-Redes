package wav

import (
	"encoding/binary"
	"os"

	"streaming-audio-redes/internal/protocol"
)

const pcmHeaderSize = 44

// BuildPCMHeader monta cabeçalho WAV PCM padrão (44 bytes).
func BuildPCMHeader(meta protocol.AudioMeta) []byte {
	hdr := make([]byte, pcmHeaderSize)
	copy(hdr[0:4], "RIFF")
	copy(hdr[8:12], "WAVE")
	copy(hdr[12:16], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:20], 16)
	binary.LittleEndian.PutUint16(hdr[20:22], 1)
	binary.LittleEndian.PutUint16(hdr[22:24], meta.Channels)
	binary.LittleEndian.PutUint32(hdr[24:28], meta.SampleRate)
	byteRate := meta.SampleRate * uint32(meta.Channels) * uint32(meta.BitsPerSample) / 8
	binary.LittleEndian.PutUint32(hdr[28:32], byteRate)
	blockAlign := meta.Channels * meta.BitsPerSample / 8
	binary.LittleEndian.PutUint16(hdr[32:34], blockAlign)
	binary.LittleEndian.PutUint16(hdr[34:36], meta.BitsPerSample)
	copy(hdr[36:40], "data")
	return hdr
}

// UpdatePCMSizes finaliza tamanhos RIFF/data em um arquivo WAV aberto.
func UpdatePCMSizes(f *os.File, pcmBytes uint32) error {
	chunkSize := uint32(36) + pcmBytes
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, chunkSize)
	if _, err := f.WriteAt(buf, 4); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(buf, pcmBytes)
	if _, err := f.WriteAt(buf, 40); err != nil {
		return err
	}
	return f.Sync()
}
