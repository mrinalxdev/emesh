package protocol

import (
	"bytes"
	// "encoding/binary" 
	"fmt"
)
type Header struct {
	From      [16]byte 
	ClockLen  uint8    
	_         [7]byte  
	Clock     Vector
}

type Vector map[string]uint64

func (v Vector) Copy() Vector {
	c := make(Vector, len(v))
	for k, val := range v {
		c[k] = val
	}
	return c
}

func (h Header) String() string {
	return fmt.Sprintf("Header{From:%s, Clock:%v}", string(bytes.TrimRight(h.From[:], "\x00")), h.Clock)
}

const MaxNodes = 32 