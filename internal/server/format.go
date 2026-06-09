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

func detectAudioMeta(path string) (protocol.AudioMeta, int64, uint32, error) {
	ext := strings.ToLower(filepath.Ext(path))
	meta := protocol.AudioMeta{Source: filepath.Base(path)}

	switch ext {
	case ".mp3":
		meta.Codec = "mp3"
		meta.Container = "mp3"
		return meta, 0, 0, nil
	case ".wav":
		return parseWAVMeta(path, meta)
	default:
		return meta, 0, 0, fmt.Errorf("formato não suportado: %s", ext)
	}
}

func parseWAVMeta(path string, meta protocol.AudioMeta) (protocol.AudioMeta, int64, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return meta, 0, 0, err
	}
	defer f.Close()

	riff := make([]byte, 12)
	if _, err := io.ReadFull(f, riff); err != nil {
		return meta, 0, 0, fmt.Errorf("ler cabeçalho WAV: %w", err)
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return meta, 0, 0, fmt.Errorf("arquivo WAV inválido")
	}

	meta.Codec = "pcm"
	meta.Container = "wav"

	for {
		hdr := make([]byte, 8)
		if _, err := io.ReadFull(f, hdr); err != nil {
			if err == io.EOF {
				break
			}
			return meta, 0, 0, fmt.Errorf("ler chunk WAV: %w", err)
		}

		chunkID := string(hdr[0:4])
		chunkSize := binary.LittleEndian.Uint32(hdr[4:8])
		dataOffset, err := f.Seek(0, io.SeekCurrent)
		if err != nil {
			return meta, 0, 0, err
		}

		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return meta, 0, 0, fmt.Errorf("chunk fmt inválido")
			}
			fmtData := make([]byte, 16)
			if _, err := io.ReadFull(f, fmtData); err != nil {
				return meta, 0, 0, err
			}
			meta.Channels = binary.LittleEndian.Uint16(fmtData[2:4])
			meta.SampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
			meta.BitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])
			if chunkSize > 16 {
				if _, err := f.Seek(int64(chunkSize-16), io.SeekCurrent); err != nil {
					return meta, 0, 0, err
				}
			}
		case "data":
			return meta, dataOffset, chunkSize, nil
		default:
			if _, err := f.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return meta, 0, 0, err
			}
		}

		if chunkSize%2 == 1 {
			if _, err := f.Seek(1, io.SeekCurrent); err != nil {
				return meta, 0, 0, err
			}
		}
	}

	return meta, 0, 0, fmt.Errorf("chunk data não encontrado em %s", filepath.Base(path))
}
