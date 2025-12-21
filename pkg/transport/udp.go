package transport

import (
	"bytes"
	"net"
	"sync"

	"eventmesh/pkg/protocol"
)

type UDP struct {
	conn *net.UDPConn
	mu   sync.Mutex
}

func NewUDP(addr string) (*UDP, error) {
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return nil, err
	}
	return &UDP{conn: conn}, nil
}

func (u *UDP) LocalAddr() net.Addr {
	return u.conn.LocalAddr()
}

func (u *UDP) Send(to net.Addr, hdr protocol.Header, payload []byte) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	raw, err := protocol.EncodeHeader(hdr, payload)
	if err != nil {
		return err
	}
	_, err = u.conn.WriteTo(raw, to)
	return err
}

func (u *UDP) Receive() (protocol.Header, []byte, net.Addr, error) {
	buf := make([]byte, 65536) // max UDP
	n, addr, err := u.conn.ReadFrom(buf)
	if err != nil {
		return protocol.Header{}, nil, nil, err
	}
	hdr, payload, err := protocol.DecodeHeader(bytes.NewReader(buf[:n]))
	return hdr, payload, addr, err
}

func (u *UDP) Close() error {
	return u.conn.Close()
}