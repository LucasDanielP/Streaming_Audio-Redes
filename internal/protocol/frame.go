package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ChunkSize é o tamanho de cada bloco PCM (múltiplo de frame estéreo 16-bit).
// 4096 bytes ≈ 23 ms em PCM 44.1 kHz estéreo.
const ChunkSize = 4096

func writeAll(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		p = p[n:]
	}
	return nil
}

// WriteTypedFrame envia [4 bytes tamanho][1 byte tipo][payload].
// O tamanho inclui o byte de tipo.
func WriteTypedFrame(w io.Writer, frameType byte, payload []byte) error {
	size := uint32(1 + len(payload))
	header := make([]byte, 5)
	binary.BigEndian.PutUint32(header, size)
	header[4] = frameType
	if err := writeAll(w, header); err != nil {
		return err
	}
	if len(payload) > 0 {
		return writeAll(w, payload)
	}
	return nil
}

// ReadTypedFrame lê um frame tipado da conexão.
func ReadTypedFrame(r io.Reader) (frameType byte, payload []byte, err error) {
	header := make([]byte, 4)
	if _, err = io.ReadFull(r, header); err != nil {
		return 0, nil, err
	}
	size := binary.BigEndian.Uint32(header)
	if size == 0 {
		return 0, []byte{}, nil
	}
	if size < 1 {
		return 0, nil, fmt.Errorf("frame inválido: tamanho %d", size)
	}
	buf := make([]byte, size)
	if _, err = io.ReadFull(r, buf); err != nil {
		return 0, nil, err
	}
	return buf[0], buf[1:], nil
}

// WriteMetaFrame envia metadados de áudio ao cliente.
func WriteMetaFrame(w io.Writer, meta AudioMeta) error {
	data, err := meta.Marshal()
	if err != nil {
		return err
	}
	return WriteTypedFrame(w, FrameTypeMeta, data)
}

// WriteAudioFrame envia um bloco de áudio.
func WriteAudioFrame(w io.Writer, payload []byte) error {
	return WriteTypedFrame(w, FrameTypeAudio, payload)
}

// ReadMetaFrame lê e valida o frame inicial de metadados.
func ReadMetaFrame(r io.Reader) (AudioMeta, error) {
	frameType, payload, err := ReadTypedFrame(r)
	if err != nil {
		return AudioMeta{}, err
	}
	if frameType != FrameTypeMeta {
		return AudioMeta{}, fmt.Errorf("esperado frame de metadados (0x01), recebido 0x%02x", frameType)
	}
	return ParseMeta(payload)
}
