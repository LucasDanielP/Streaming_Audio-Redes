package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"streaming-audio-redes/internal/protocol"
)

// FileSource lê um arquivo de áudio em loop com envio em tempo real.
type FileSource struct {
	path          string
	meta          protocol.AudioMeta
	pcmDataOffset int64
}

// NewFileSource detecta metadados e prepara transmissão em loop.
func NewFileSource(path string) (*FileSource, error) {
	meta, pcmOffset, _, err := detectAudioMeta(path)
	if err != nil {
		return nil, err
	}
	return &FileSource{
		path:          path,
		meta:          meta,
		pcmDataOffset: pcmOffset,
	}, nil
}

func (f *FileSource) Meta() protocol.AudioMeta {
	return f.meta
}

func (f *FileSource) Close() error {
	return nil
}

func (f *FileSource) Stream(ctx context.Context, emit func([]byte) error) error {
	clock := newPaceClock(f.meta)
	chunkBytes := alignedChunkBytes(f.meta.Channels)

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

func (f *FileSource) streamOnce(ctx context.Context, emit func([]byte) error, clock *paceClock, chunkBytes int) error {
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
