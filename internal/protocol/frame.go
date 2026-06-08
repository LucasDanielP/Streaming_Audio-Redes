package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ChunkSize é o tamanho máximo de cada bloco de áudio enviado pelo servidor.
const ChunkSize = 4096

// WriteTypedFrame envia [4 bytes tamanho][1 byte tipo][payload].
// O tamanho inclui o byte de tipo.
func WriteTypedFrame(w io.Writer, frameType byte, payload []byte) error {
	size := uint32(1 + len(payload))
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, size)
	if _, err := w.Write(header); err != nil {
		return err
	}
	if _, err := w.Write([]byte{frameType}); err != nil {
		return err
	}
	if len(payload) > 0 {
		_, err := w.Write(payload)
		return err
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

// ReadAudioFrame lê o próximo bloco de áudio do fluxo.
func ReadAudioFrame(r io.Reader) ([]byte, error) {
	frameType, payload, err := ReadTypedFrame(r)
	if err != nil {
		return nil, err
	}
	if frameType != FrameTypeAudio {
		return nil, fmt.Errorf("esperado frame de áudio (0x02), recebido 0x%02x", frameType)
	}
	return payload, nil
}
