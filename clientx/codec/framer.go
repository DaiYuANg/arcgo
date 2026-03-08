package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	DefaultMaxFrameBytes uint32 = 4 * 1024 * 1024
)

type Framer interface {
	ReadFrame(r io.Reader) ([]byte, error)
	WriteFrame(w io.Writer, frame []byte) error
}

type LengthPrefixedFramer struct {
	MaxFrameBytes uint32
}

func NewLengthPrefixed(maxFrameBytes uint32) *LengthPrefixedFramer {
	if maxFrameBytes == 0 {
		maxFrameBytes = DefaultMaxFrameBytes
	}
	return &LengthPrefixedFramer{MaxFrameBytes: maxFrameBytes}
}

func (f *LengthPrefixedFramer) ReadFrame(r io.Reader) ([]byte, error) {
	if r == nil {
		return nil, errors.New("reader is nil")
	}

	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size > f.MaxFrameBytes {
		return nil, fmt.Errorf("frame too large: %d > %d", size, f.MaxFrameBytes)
	}
	if size == 0 {
		return []byte{}, nil
	}

	frame := make([]byte, size)
	if _, err := io.ReadFull(r, frame); err != nil {
		return nil, err
	}
	return frame, nil
}

func (f *LengthPrefixedFramer) WriteFrame(w io.Writer, frame []byte) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	if uint32(len(frame)) > f.MaxFrameBytes {
		return fmt.Errorf("frame too large: %d > %d", len(frame), f.MaxFrameBytes)
	}

	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(frame)))
	if err := writeFull(w, header[:]); err != nil {
		return err
	}
	return writeFull(w, frame)
}

func writeFull(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
