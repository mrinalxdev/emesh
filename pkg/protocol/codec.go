package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

var (
	ErrOversizedClock = errors.New("clock exceeds max nodes")
	ErrShortHeader    = errors.New("short header")
)

func EncodeHeader(h Header, payload []byte) ([]byte, error) {
	if len(h.Clock) > MaxNodes {
		return nil, ErrOversizedClock
	}
	buf := new(bytes.Buffer)
	// fixed part
	if err := binary.Write(buf, binary.BigEndian, h.From); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, h.ClockLen); err != nil {
		return nil, err
	}
	buf.Write(make([]byte, 7)) // padding
	// variable clock
	for id, val := range h.Clock {
		var fixed [16]byte
		copy(fixed[:], id)
		if err := binary.Write(buf, binary.BigEndian, fixed); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, val); err != nil {
			return nil, err
		}
	}
	buf.Write(payload)
	return buf.Bytes(), nil
}

func DecodeHeader(r io.Reader) (Header, []byte, error) {
	var h Header
	if err := binary.Read(r, binary.BigEndian, &h.From); err != nil {
		return h, nil, err
	}
	if err := binary.Read(r, binary.BigEndian, &h.ClockLen); err != nil {
		return h, nil, err
	}
	if _, err := io.ReadFull(r, make([]byte, 7)); err != nil {
		return h, nil, err
	}
	h.Clock = make(Vector)
	for i := uint8(0); i < h.ClockLen; i++ {
		var id [16]byte
		var val uint64
		if err := binary.Read(r, binary.BigEndian, &id); err != nil {
			return h, nil, err
		}
		if err := binary.Read(r, binary.BigEndian, &val); err != nil {
			return h, nil, err
		}
		h.Clock[string(bytes.TrimRight(id[:], "\x00"))] = val
	}
	// rest is payload
	payload, err := io.ReadAll(r)
	return h, payload, err
}