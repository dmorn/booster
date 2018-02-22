package node

import (
	"sync"
	"fmt"
	"io"
	"errors"
	"net"

	"github.com/danielmorandini/booster-network/socks5"
)

type Tunnel struct {
	id []byte
	Target net.Addr

	sync.Mutex
	copies int // number of copies
	acks int // number of acknoledged copies
}

func NewTunnel(target net.Addr) *Tunnel {
	return &Tunnel {
		id: sha1Hash([]byte(target.String())),
		Target: target,
		copies: 1,
	}
}

func (t *Tunnel) ID() string {
	return fmt.Sprintf("%x", t.id)
}

func (t *Tunnel) Copies() int {
	t.Lock()
	defer t.Unlock()

	return t.copies
}

func (t *Tunnel) Acks() int {
	t.Lock()
	defer t.Unlock()

	return t.acks
}

func (t *Tunnel) Read(r io.Reader) error {
	buf := make([]byte, 20) // sha1 len
	if _, err := io.ReadFull(r, buf); err != nil {
		return errors.New("tunnel: unable to read identifier: " + err.Error() + " buffer: " + fmt.Sprintf("%v", buf))
	}

	copy(buf, t.id)

	host, err := socks5.ReadHost(r)
	if err != nil {
		return errors.New("tunnel: unable to decode host: " + err.Error())
	}
	port, err := socks5.ReadPort(r)
	if err != nil {
		return errors.New("tunnel: unable to decode p port: " + err.Error())
	}

	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return errors.New("node: unable to create addr: " + err.Error())
	}
	t.Target = addr

	buf = make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return errors.New("tunnel: unable to decode acks and copies: " + err.Error())
	}

	t.copies = int(buf[0])
	t.acks = int(buf[1])

	return nil
}

func (t *Tunnel) EncodeBinary() ([]byte, error) {
	if t == nil {
		return nil, errors.New("tunnel: trying to encode nil")
	}

	host, port, err := net.SplitHostPort(t.Target.String())
	if err != nil {
		return nil, err
	}

	hbuf, err := socks5.EncodeHostBinary(host)   // host buffer
	pbuf, err := socks5.EncodePortBinary(port) // proxy port buffer
	buf := make([]byte, len(hbuf)+len(pbuf)+len(t.id)+2)

	buf = append(buf, t.id...)
	buf = append(buf, hbuf...)
	buf = append(buf, pbuf...)
	buf = append(buf, byte(t.copies))
	buf = append(buf, byte(t.acks))

	return buf, nil
}
