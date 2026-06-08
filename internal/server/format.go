package server

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"streaming-audio-redes/internal/protocol"
)

func detectAudioMeta(path string) (protocol.AudioMeta, int64, error) {
	ext := strings.ToLower(filepath.Ext(path))
	meta := protocol.AudioMeta{Source: filepath.Base(path)}

	switch ext {
	case ".mp3":
		meta.Codec = "mp3"
		meta.Container = "mp3"
		return meta, 0, nil
	case ".wav":
		return parseWAVMeta(path, meta)
	default:
		return meta, 0, fmt.Errorf("formato não suportado: %s", ext)
	}
}

func parseWAVMeta(path string, meta protocol.AudioMeta) (protocol.AudioMeta, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return meta, 0, err
	}
	defer f.Close()

	hdr := make([]byte, 44)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return meta, 0, fmt.Errorf("ler cabeçalho WAV: %w", err)
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return meta, 0, fmt.Errorf("arquivo WAV inválido")
	}

	meta.Codec = "pcm"
	meta.Container = "wav"
	meta.Channels = binary.LittleEndian.Uint16(hdr[22:24])
	meta.SampleRate = binary.LittleEndian.Uint32(hdr[24:28])
	meta.BitsPerSample = binary.LittleEndian.Uint16(hdr[34:36])
	return meta, 44, nil
}
