package source

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"streaming-audio-redes/internal/audio/pace"
	"streaming-audio-redes/internal/audio/pcm"
	"streaming-audio-redes/internal/audio/wav"
	"streaming-audio-redes/internal/protocol"
)

// File transmite um arquivo de áudio em loop com pacing em tempo real.
type File struct {
	path          string
	meta          protocol.AudioMeta
	pcmDataOffset int64
}

// NewFile detecta metadados e prepara transmissão em loop.
func NewFile(path string) (*File, error) {
	meta, pcmOffset, _, err := wav.DetectMeta(path)
	if err != nil {
		return nil, err
	}
	return &File{
		path:          path,
		meta:          meta,
		pcmDataOffset: pcmOffset,
	}, nil
}

func (f *File) Meta() protocol.AudioMeta {
	return f.meta
}

func (f *File) Close() error {
	return nil
}

func (f *File) Stream(ctx context.Context, emit func([]byte) error) error {
	clock := pace.New(f.meta)
	chunkBytes := pcm.AlignedChunkSize(f.meta.Channels)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := f.streamOnce(ctx, emit, clock, chunkBytes); err != nil {
			return err
		}
		clock.Reset(f.meta)
	}
}

func (f *File) streamOnce(ctx context.Context, emit func([]byte) error, clock *pace.Clock, chunkBytes int) error {
	file, err := os.Open(f.path)
	if err != nil {
		return err
	}
	defer file.Close()

	if f.pcmDataOffset > 0 {
		if _, err := file.Seek(f.pcmDataOffset, io.SeekStart); err != nil {
			return fmt.Errorf("pular cabeçalho WAV: %w", err)
		}
	}

	buf := make([]byte, chunkBytes)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, err := file.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if err := emit(chunk); err != nil {
				return err
			}
			if waitErr := clock.Wait(ctx, n); waitErr != nil {
				return waitErr
			}
		}
		if err == io.EOF {
			log.Printf("[servidor] fim do arquivo — reiniciando loop")
			return nil
		}
		if err != nil {
			return err
		}
	}
}
