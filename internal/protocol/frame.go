package protocol

import (
	"encoding/binary"
	"io"
)

// ChunkSize é o tamanho máximo de cada bloco de áudio enviado pelo servidor.
const ChunkSize = 4096

// WriteFrame envia um pacote no formato [4 bytes tamanho][payload].
func WriteFrame(w io.Writer, payload []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadFrame lê um pacote completo do stream.
func ReadFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header)
	if size == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
