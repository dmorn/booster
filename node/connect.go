package node

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/danielmorandini/booster-network/socks5"
)

// connect performs the steps required to pair with a remote node.
// laddr is the local booster address to dial with. raddr is the remote
// node address that as to be registered.
//
// returns the id of the connected node.
func (b *Booster) Connect(ctx context.Context, network, laddr, raddr string) (string, error) {
	_ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := b.DialContext(_ctx, network, laddr)
	if err != nil {
		return "", errors.New("booster: unable to contact booster " + laddr + " : " + err.Error())
	}
	defer conn.Close()

	abuf, err := socks5.EncodeAddressBinary(raddr)
	if err != nil {
		return "", err
	}

	buf := make([]byte, 0, 3+len(abuf))
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDConnect)
	buf = append(buf, BoosterFieldReserved)
	buf = append(buf, abuf...)

	if _, err := conn.Write(buf); err != nil {
		return "", errors.New("booster: unable to write connect request: " + err.Error())
	}

	buf = make([]byte, 4+20) // sha1 len
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", errors.New("booster: unable to read connect response: " + err.Error())
	}

	v := buf[0] // version
	if v != BoosterVersion1 {
		return "", errors.New("booster: unsupported booster version in connect response: " + string(v))
	}

	_ = buf[1]  // cmd
	r := buf[2] // response
	if r != BoosterRespSuccess {
		return "", errors.New("booster: connect request refused")
	}

	_ = buf[3] // reserved field
	id := fmt.Sprintf("%x", buf[4:])

	return id, nil
}

func (b *Booster) handleConnect(ctx context.Context, conn net.Conn) (*Node, error) {
	addr, err := socks5.ReadAddress(conn)
	if err != nil {
		return nil, errors.New("booster: " + err.Error())
	}

	paddr, err := b.Hello(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	host, bport, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, errors.New("booster: unable to handle connect: " + err.Error())
	}
	_, pport, err := net.SplitHostPort(paddr)
	if err != nil {
		return nil, errors.New("booster: unable to handle connect: " + err.Error())
	}

	rn, err := NewNode(host, pport, bport)
	if err != nil {
		return nil, err
	}
	if _, err := b.AddNode(rn); err != nil {
		return nil, err
	}

	bid, err := hex.DecodeString(rn.ID())
	if err != nil {
		return nil, errors.New("booster: " + err.Error())
	}

	buf := make([]byte, 0, len(bid)+4)
	buf = append(buf, BoosterVersion1)
	buf = append(buf, BoosterCMDConnect)
	buf = append(buf, BoosterRespSuccess)
	buf = append(buf, BoosterFieldReserved)
	buf = append(buf, bid...)

	if _, err := conn.Write(buf); err != nil {
		return nil, errors.New("booster: unable to write connect response: " + err.Error())
	}

	return rn, nil
}
