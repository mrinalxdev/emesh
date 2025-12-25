package protocol

import (
	"bytes"
	"encoding/gob"
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

//{comment} adding deltaEncode which returns the minimal diff
//against the base

func (v Vector) DeltaEncode(base Vector) Vector {
	d := make(Vector)
	
	for id, ts := range v {
		if base[id] != ts {
			d[id] = ts
		}
	}
	
	return d
}

// marshal binary implements encoding, for wiring for now
func (v Vector) MarshalBinary() ([]byte, error){
	var buf bytes.Buffer
	
	err := gob.NewEncoder(&buf).Encode(v)
	
	return buf.Bytes(), err
}

const MaxNodes = 32 